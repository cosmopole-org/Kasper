package actions_user

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"kasper/src/abstract/models/action"
	"kasper/src/abstract/models/core"
	"kasper/src/abstract/models/trx"
	"kasper/src/abstract/state"
	"kasper/src/core/module/actor/model/base"
	mainstate "kasper/src/core/module/actor/model/state"
	inputsusers "kasper/src/shell/api/inputs/users"
	models "kasper/src/shell/api/model"
	outputsusers "kasper/src/shell/api/outputs/users"

	"github.com/mitchellh/mapstructure"
)

type Actions struct {
	app core.ICore
}

func Install(a *Actions) error {
	a.app.ModifyState(false, func(trx trx.ITrx) {
		for _, godUsername := range a.app.Gods() {
			var user = models.User{}
			userId := ""
			userStr := trx.GetIndex("User", godUsername+"@"+a.app.Id(), "username", "id")
			if userStr == "" {
				var (
					user    models.User
					session models.Session
				)
				user = models.User{Id: a.app.Tools().Storage().GenId(a.app.Id()), Typ: "human", PublicKey: "", Username: godUsername + "@" + a.app.Id()}
				session = models.Session{Id: a.app.Tools().Storage().GenId(a.app.Id()), UserId: user.Id}
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
	_, res, _ := a.app.Actor().FetchAction("/users/get").Act(mainstate.NewState(base.NewInfo("", ""), state.Trx()), inputsusers.GetInput{UserId: state.Info().UserId()})
	return outputsusers.AuthenticateOutput{Authenticated: true, User: res.(outputsusers.GetOutput).User}, nil
}

// Login /users/register check [ false false false ] access [ true false false false POST ]
func (a *Actions) Register(state state.IState, input inputsusers.LoginInput) (any, error) {
	trx := state.Trx()
	if trx.HasIndex("User", input.Username+"@"+a.app.Id(), "username", "id") {
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
		sign := a.app.SignPacket(bin)
		_, res, err2 := a.app.Actor().FetchAction("/users/create").(action.ISecureAction).SecurelyAct("", "", bin, sign, req, a.app.Id())
		if err2 != nil {
			return nil, err2
		}
		var response outputsusers.CreateOutput
		mapstructure.Decode(res, &response)
		return outputsusers.LoginOutput{User: response.User, Session: response.Session, PrivateKey: privKey}, nil
	}
	return nil, errors.New("username already exist")
}

// Create /users/create check [ false false false ] access [ true false false false POST ]
func (a *Actions) Create(state state.IState, input inputsusers.CreateInput) (any, error) {
	var (
		user    models.User
		session models.Session
	)
	trx := state.Trx()
	if trx.HasIndex("User", input.Username+"@"+state.Dummy(), "username", "id") {
		return nil, errors.New("username already exists")
	}
	user = models.User{Id: a.app.Tools().Storage().GenId(input.Origin()), Typ: "human", PublicKey: input.PublicKey, Username: input.Username + "@" + state.Dummy()}
	session = models.Session{Id: a.app.Tools().Storage().GenId(input.Origin()), UserId: user.Id}
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
