package actions_user

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"kasper/src/abstract/models/action"
	"kasper/src/abstract/models/core"
	"kasper/src/abstract/models/trx"
	"kasper/src/abstract/state"
	"kasper/src/core/module/actor/model/base"
	mainstate "kasper/src/core/module/actor/model/state"
	inputsusers "kasper/src/shell/api/inputs/users"
	models "kasper/src/shell/api/model"
	outputsusers "kasper/src/shell/api/outputs/users"
	"kasper/src/shell/utils/crypto"
	"log"
	"net"
	"net/http"
	"time"
)

type Actions struct {
	App core.ICore
}

func Install(a *Actions) error {
	a.App.ModifyState(false, func(trx trx.ITrx) {
		for _, godUsername := range a.App.Gods() {
			var user = models.User{}
			userId := ""
			userStr := trx.GetIndex("User", "username", "id", godUsername+"@"+a.App.Id())
			if userStr == "" {
				var (
					user    models.User
					session models.Session
				)
				user = models.User{Id: a.App.Tools().Storage().GenId(trx, a.App.Id()), Typ: "human", PublicKey: "", Username: godUsername + "@" + a.App.Id()}
				session = models.Session{Id: a.App.Tools().Storage().GenId(trx, a.App.Id()), UserId: user.Id}
				user.Push(trx)
				session.Push(trx)
				userId = user.Id
			} else {
				userId = user.Id
			}
			trx.PutLink("god::"+userId, "true")
		}
	})
	return nil
}

// Authenticate /users/authenticate check [ true false false ] access [ true false false false POST ]
func (a *Actions) Authenticate(state state.IState, _ inputsusers.AuthenticateInput) (any, error) {
	_, res, _ := a.App.Actor().FetchAction("/users/get").Act(mainstate.NewState(base.NewInfo("", ""), state.Trx()), inputsusers.GetInput{UserId: state.Info().UserId()})
	return outputsusers.AuthenticateOutput{Authenticated: true, User: res.(outputsusers.GetOutput).User}, nil
}

// Transfer /users/transfer check [ true false false ] access [ true false false false POST ]
func (a *Actions) Transfer(state state.IState, input inputsusers.TransferInput) (any, error) {
	user := models.User{Id: state.Info().UserId()}.Pull(state.Trx())
	if user.Balance < input.Amount {
		return nil, errors.New("your balance is not enough")
	}
	toUserId := state.Trx().GetIndex("User", "username", "id", input.ToUsername)
	if toUserId == "" {
		return nil, errors.New("target user not found")
	}
	toUser := models.User{Id: toUserId}.Pull(state.Trx())
	user.Balance -= input.Amount
	toUser.Balance += input.Amount
	user.Push(state.Trx())
	toUser.Push(state.Trx())
	return map[string]any{}, nil
}

// LockToken /users/lockToken check [ true false false ] access [ true false false false POST ]
func (a *Actions) LockToken(state state.IState, input inputsusers.LockTokenInput) (any, error) {
	user := models.User{Id: state.Info().UserId()}.Pull(state.Trx())
	if user.Balance < input.Amount {
		return nil, errors.New("your balance is not enough")
	}
	validators := a.App.Tools().Network().Chain().GetValidatorsOfMachineShard(input.ExecMachineId)
	str, err := json.Marshal(validators)
	if err != nil {
		return nil, err
	}
	if input.Amount < int64(len(validators)) {
		return nil, errors.New("amount to be locked can not be less than validators count")
	}
	user.Balance -= input.Amount
	user.Push(state.Trx())
	lockId := crypto.SecureUniqueString()
	state.Trx().PutJson("Json::User::"+state.Info().UserId(), "lockedTokens."+lockId, map[string]any{"amount": input.Amount, "validators": string(str)}, true)
	return map[string]any{"tokenId": lockId}, nil
}

var (
	zeroDialer net.Dialer
)

var netClient = &http.Client{
	Transport: &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return zeroDialer.DialContext(ctx, "tcp4", addr)
		},
	},
	Timeout: time.Duration(1000) * time.Second,
}

// Login /users/login check [ false false false ] access [ true false false false POST ]
func (a *Actions) Login(state state.IState, input inputsusers.LoginInput) (any, error) {
	var resp, err = netClient.Get("https://oauth2.googleapis.com/tokeninfo?id_token=" + (input.EmailToken))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err2 := io.ReadAll(resp.Body)
	if err2 != nil {
		return nil, err2
	}
	result := make(map[string]interface{})
	err3 := json.Unmarshal(body, &result)
	if err3 != nil {
		return nil, err3
	}
	email := ""
	if result["email"] != nil {
		email = result["email"].(string)
	} else {
		return nil, errors.New("access denied")
	}
	trx := state.Trx()
	userId := trx.GetLink("UserEmailToId::" + email)
	if userId != "" {
		user := models.User{Id: userId}.Pull(trx)
		session := models.Session{Id: trx.GetIndex("Session", "userId", "id", user.Id)}.Pull(trx)
		privKey := trx.GetLink("UserPrivateKey::" + user.Id)
		return outputsusers.LoginOutput{User: user, Session: session, PrivateKey: privKey}, nil
	}
	if !trx.HasIndex("User", "username", "id", input.Username+"@"+a.App.Id()) {
		key, err := rsa.GenerateKey(rand.Reader, 4096)
		if err != nil {
			return nil, err
		}
		pub := key.Public()
		keyPEM := pem.EncodeToMemory(
			&pem.Block{
				Type:  "RSA PRIVATE KEY",
				Bytes: x509.MarshalPKCS1PrivateKey(key),
			},
		)
		pubPEM := pem.EncodeToMemory(
			&pem.Block{
				Type:  "RSA PUBLIC KEY",
				Bytes: x509.MarshalPKCS1PublicKey(pub.(*rsa.PublicKey)),
			},
		)
		pubKey := string(pubPEM)
		pubKey = pubKey[len("-----BEGIN RSA PUBLIC KEY-----\n") : len(pubKey)-len("\n-----END RSA PUBLIC KEY-----\n")]
		privKey := string(keyPEM)
		privKey = privKey[len("-----BEGIN RSA PRIVATE KEY-----\n") : len(privKey)-len("\n-----END RSA PRIVATE KEY-----\n")]
		req := inputsusers.CreateInput{
			Username:  input.Username,
			PublicKey: pubKey,
		}
		bin, _ := json.Marshal(req)
		sign := a.App.SignPacket(bin)
		_, res, err2 := a.App.Actor().FetchAction("/users/create").(action.ISecureAction).SecurelyAct("", "", bin, sign, req, a.App.Id())
		if err2 != nil {
			return nil, err2
		}
		var response outputsusers.CreateOutput
		b, e := json.Marshal(res)
		if e != nil {
			log.Println(e)
			return nil, e
		}
		e = json.Unmarshal(b, &response)
		if e != nil {
			log.Println(e)
			return nil, e
		}
		trx.PutLink("UserPrivateKey::"+response.User.Id, privKey)
		trx.PutLink("UserEmailToId::" + email, response.User.Id)
		return outputsusers.LoginOutput{User: response.User, Session: response.Session, PrivateKey: privKey}, nil
	} else {
		return nil, errors.New("username already exist")
	}
}

// Create /users/create check [ false false false ] access [ true false false false POST ]
func (a *Actions) Create(state state.IState, input inputsusers.CreateInput) (any, error) {
	var (
		user    models.User
		session models.Session
	)
	trx := state.Trx()
	if trx.HasIndex("User", "username", "id", input.Username+"@"+state.Source()) {
		return nil, errors.New("username already exists")
	}
	user = models.User{Id: a.App.Tools().Storage().GenId(trx, input.Origin()), Typ: "human", Balance: 1000000000000000, PublicKey: input.PublicKey, Username: input.Username + "@" + state.Source()}
	session = models.Session{Id: a.App.Tools().Storage().GenId(trx, input.Origin()), UserId: user.Id}
	user.Push(trx)
	session.Push(trx)
	return outputsusers.CreateOutput{User: user, Session: session}, nil
}

// Get /users/get check [ false false false ] access [ true false false false GET ]
func (a *Actions) Get(state state.IState, input inputsusers.GetInput) (any, error) {
	trx := state.Trx()
	if !trx.HasObj("User", input.UserId) {
		return nil, errors.New("user not found")
	}
	user := models.User{Id: input.UserId}.Pull(trx)
	return outputsusers.GetOutput{User: user}, nil
}

// List /users/list check [ true false false ] access [ true false false false GET ]
func (a *Actions) List(state state.IState, input inputsusers.ListInput) (any, error) {
	trx := state.Trx()
	users, err := models.User{}.All(trx, input.Offset, input.Count, map[string]string{"type": "human"})
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return map[string]any{"users": users}, nil
}
