package actions_user

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"kasper/src/abstract"
	moduleactormodel "kasper/src/core/module/actor/model"
	inputsusers "kasper/src/shell/api/inputs/users"
	models "kasper/src/shell/api/model"
	outputsusers "kasper/src/shell/api/outputs/users"
	"kasper/src/shell/layer1/adapters"
	modulestate "kasper/src/shell/layer1/module/state"

	"github.com/mitchellh/mapstructure"
)

type Actions struct {
	Layer abstract.ILayer
}

func Install(s adapters.IStorage, a *Actions) error {
	s.DoTrx(func(trx abstract.ITrx) error {
		for _, godUsername := range a.Layer.Core().Gods() {
			var user = models.User{}
			userId := ""
			userStr := trx.GetIndex("User", godUsername+"@"+a.Layer.Core().Id(), "username", "id")
			if userStr == "" {
				var (
					user    models.User
					session models.Session
				)
				user = models.User{Id: trx.GenId(a.Layer.Core().Id()), Typ: "human", PublicKey: "", Username: godUsername + "@" + a.Layer.Core().Id()}
				trx.PutObj(user.Id, user)
				session = models.Session{Id: trx.GenId(a.Layer.Core().Id()), UserId: user.Id}
				trx.PutObj(session.Id, session)
				userId = user.Id
			} else {
				userId = user.Id
			}
			trx.PutLink("god::"+userId, "true")
		}
		return nil
	})
	return nil
}

// Authenticate /users/authenticate check [ true false false ] access [ true false false false POST ]
func (a *Actions) Authenticate(s abstract.IState, _ inputsusers.AuthenticateInput) (any, error) {
	state := abstract.UseState[modulestate.IStateL1](s)
	_, res, _ := a.Layer.Actor().FetchAction("/users/get").Act(a.Layer.Sb().NewState(moduleactormodel.NewInfo("", "", "", ""), state.Trx()), inputsusers.GetInput{UserId: state.Info().UserId()})
	return outputsusers.AuthenticateOutput{Authenticated: true, User: res.(outputsusers.GetOutput).User}, nil
}

// Login /users/register check [ false false false ] access [ true false false false POST ]
func (a *Actions) Register(s abstract.IState, input inputsusers.LoginInput) (any, error) {
	state := abstract.UseState[modulestate.IStateL1](s)
	trx := state.Trx()
	if trx.HasIndex("User", input.Username+"@"+a.Layer.Core().Id(), "username", "id") {
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
		_, res, err2 := a.Layer.Actor().FetchAction("/users/create").(abstract.ISecureAction).SecurelyAct(a.Layer.Core().Get(1), "", "", inputsusers.CreateInput{
			Username:  input.Username,
			PublicKey: pubKey,
		}, a.Layer.Core().Id())
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
func (a *Actions) Create(s abstract.IState, input inputsusers.CreateInput) (any, error) {
	state := abstract.UseState[modulestate.IStateL1](s)
	var (
		user    models.User
		session models.Session
	)
	trx := state.Trx()
	if trx.HasIndex("User", input.Username+"@"+state.Dummy(), "username", "id") {
		return nil, errors.New("username already exists")
	}
	user = models.User{Id: trx.GenId(input.Origin()), Typ: "human", PublicKey: input.PublicKey, Username: input.Username + "@" + state.Dummy()}
	trx.PutObj(user.Id, user)
	session = models.Session{Id: trx.GenId(input.Origin()), UserId: user.Id}
	trx.PutObj(session.Id, session)
	return outputsusers.CreateOutput{User: user, Session: session}, nil
}

// Get /users/get check [ false false false ] access [ true false false false GET ]
func (a *Actions) Get(s abstract.IState, input inputsusers.GetInput) (any, error) {
	state := abstract.UseState[modulestate.IStateL1](s)
	trx := state.Trx()
	// models.User{Id: input.UserId}.
	// user, err := adapters.ParseObj[models.User](trx.GetBytes("obj::User::" + ))
	if err != nil {
		return nil, err
	}
	return outputsusers.GetOutput{User: user}, nil
}
