package actions_user

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"github.com/coreos/go-oidc"
	"kasper/src/abstract/models/action"
	"kasper/src/abstract/models/core"
	"kasper/src/abstract/state"
	"kasper/src/core/module/actor/model/base"
	mainstate "kasper/src/core/module/actor/model/state"
	inputsusers "kasper/src/shell/api/inputs/users"
	models "kasper/src/shell/api/model"
	outputsusers "kasper/src/shell/api/outputs/users"
	"kasper/src/shell/utils/crypto"
	"log"
)

type Actions struct {
	App       core.ICore
	OauthProv *oidc.Provider
	OauthCtx  context.Context
}

func Install(a *Actions) error {
	a.OauthCtx = context.Background()
	provider, err := oidc.NewProvider(a.OauthCtx, issuer)
	if err != nil {
		panic("Failed to get provider: " + err.Error())
	}
	a.OauthProv = provider
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

// Mint /users/mint check [ true false false ] access [ true false false false POST ]
func (a *Actions) Mint(state state.IState, input inputsusers.MintInput) (any, error) {
	if state.Info().UserId() != "1@global" {
		return nil, errors.New("access denied")
	}
	username := models.User{Id: state.Trx().GetLink("UserEmailToId::" + input.ToUserEmail)}.Pull(state.Trx()).Username
	toUserId := state.Trx().GetIndex("User", "username", "id", username)
	if toUserId == "" {
		return nil, errors.New("target user not found")
	}
	toUser := models.User{Id: toUserId}.Pull(state.Trx())
	toUser.Balance += input.Amount
	toUser.Push(state.Trx())
	return map[string]any{}, nil
}

// CheckSign /users/checkSign check [ true false false ] access [ true false false false POST ]
func (a *Actions) CheckSign(state state.IState, input inputsusers.CheckSignInput) (any, error) {
	if state.Info().UserId() != "1@global" {
		return nil, errors.New("access denied")
	}
	data, err := base64.StdEncoding.DecodeString(input.Payload)
	if err != nil {
		log.Println(err)
		return map[string]any{"valid": false}, nil
	}
	if success, _, _ := a.App.Tools().Security().AuthWithSignature(input.UserId, data, input.Signature); success {
		email := state.Trx().GetLink("UserIdToEmail::" + input.UserId)
		return map[string]any{"valid": true, "email": email}, nil
	} else {
		return map[string]any{"valid": false}, nil
	}
}

// LockToken /users/lockToken check [ true false false ] access [ true false false false POST ]
func (a *Actions) LockToken(state state.IState, input inputsusers.LockTokenInput) (any, error) {
	user := models.User{Id: state.Info().UserId()}.Pull(state.Trx())
	if user.Balance < input.Amount {
		return nil, errors.New("your balance is not enough")
	}
	lockId := crypto.SecureUniqueString()
	if input.Type == "exec" {
		validators := a.App.Tools().Network().Chain().GetValidatorsOfMachineShard(input.Target)
		str, err := json.Marshal(validators)
		if err != nil {
			return nil, err
		}
		if input.Amount < int64(len(validators)) {
			return nil, errors.New("amount to be locked can not be less than validators count")
		}
		user.Balance -= input.Amount
		user.Push(state.Trx())
		state.Trx().PutJson("Json::User::"+state.Info().UserId(), "lockedTokens."+lockId, map[string]any{"type": "exec", "amount": input.Amount, "validators": string(str)}, true)
	} else if input.Type == "pay" {
		if !state.Trx().HasObj("User", input.Target) {
			return nil, errors.New("target user not acceptable")
		}
		user.Balance -= input.Amount
		user.Push(state.Trx())
		state.Trx().PutJson("Json::User::"+state.Info().UserId(), "lockedTokens."+lockId, map[string]any{"type": "pay", "amount": input.Amount, "userId": input.Target}, true)
	} else {
		return nil, errors.New("unknown lock type")
	}
	return map[string]any{"tokenId": lockId}, nil
}

// ConsumeLock /users/consumeLock check [ true false false ] access [ true false false false POST ]
func (a *Actions) ConsumeLock(state state.IState, input inputsusers.ConsumeLockInput) (any, error) {
	receiver := models.User{Id: state.Info().UserId()}.Pull(state.Trx())
	if input.Type == "pay" {
		if !state.Trx().HasObj("User", input.UserId) {
			return nil, errors.New("payer user not found")
		}
		if success, _, _ := a.App.Tools().Security().AuthWithSignature(input.UserId, []byte(input.LockId), input.Signature); success {
			sender := models.User{Id: input.UserId}.Pull(state.Trx())
			if payment, err := state.Trx().GetJson("Json::User::"+sender.Id, "lockedTokens."+input.LockId); err == nil {
				if typ, ok := payment["type"].(string); ok && (typ == "pay") {
					if amount, ok := payment["amount"].(float64); ok && (int64(amount) == input.Amount) {
						if target, ok := payment["userId"].(string); ok && (target == receiver.Id) {
							sender.Balance -= input.Amount
							sender.Push(state.Trx())
							receiver.Balance += input.Amount
							receiver.Push(state.Trx())
							state.Trx().DelJson("Json::User::"+sender.Id, "lockedTokens."+input.LockId)
							return map[string]any{"success": true}, nil
						} else {
							return nil, errors.New("you are not target")
						}
					} else {
						return nil, errors.New("amount of payment not matched")
					}
				} else {
					return nil, errors.New("type is not payment")
				}
			} else {
				return nil, errors.New("lock not found")
			}
		} else {
			return nil, errors.New("signature not verified")
		}
	} else {
		return nil, errors.New("unknown lock type")
	}
}

var (
	clientID = "94AKF0INP2ApjXud6TTirxyjoQxqNpEk"
	issuer   = "https://dev-epfxvx2scaq4cj3t.us.auth0.com/"
)

// Login /users/login check [ false false false ] access [ true false false false POST ]
func (a *Actions) Login(state state.IState, input inputsusers.LoginInput) (any, error) {

	verifier := a.OauthProv.Verifier(&oidc.Config{ClientID: clientID})

	idToken, err := verifier.Verify(a.OauthCtx, input.EmailToken)
	if err != nil {
		log.Println("Invalid token: " + err.Error())
		return nil, err
	}
	var claims struct {
		Email string `json:"email"`
	}
	if err = idToken.Claims(&claims); err != nil {
		log.Println("Failed to parse claims" + err.Error())
		return nil, err
	}
	email := claims.Email
	trx := state.Trx()
	userId := trx.GetLink("UserEmailToId::" + email)
	log.Println("fetching email:", "["+email+"]", "["+userId+"]")
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
		trx.PutLink("UserEmailToId::"+email, response.User.Id)
		trx.PutLink("UserIdToEmail::"+response.User.Id, email)
		log.Println("saving email:", "["+email+"]", "["+response.User.Id+"]")
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
