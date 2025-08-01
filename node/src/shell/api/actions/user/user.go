package actions_user

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
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
	"strings"

	firebase "firebase.google.com/go/v4"
	"google.golang.org/api/option"
)

type Actions struct {
	App         core.ICore
	OauthCtx    context.Context
	firebaseApp *firebase.App
}

func (a *Actions) initFirebase() {
	opt := option.WithCredentialsFile("/app/serviceAccounts.json")
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		log.Fatalf("Error initializing Firebase app: %v\n", err)
	}
	a.firebaseApp = app
}

func Install(a *Actions) error {
	a.OauthCtx = context.Background()
	a.initFirebase()
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

// Login /users/login check [ false false false ] access [ true false false false POST ]
func (a *Actions) Login(state state.IState, input inputsusers.LoginInput) (any, error) {

	ctx := a.OauthCtx
	client, err := a.firebaseApp.Auth(ctx)
	if err != nil {
		log.Println(err)
		e := errors.New("error getting Auth client")
		log.Println(e)
		return nil, e
	}
	token, err := client.VerifyIDToken(ctx, input.EmailToken)
	if err != nil {
		log.Println(err)
		e := errors.New("invalid ID token")
		log.Println(e)
		return nil, e
	}
	email, ok := token.Claims["email"].(string)
	if !ok {
		e := errors.New("email claim not found or invalid")
		log.Println(e)
		return nil, e
	}

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
		priKeyRaw, pubKeyRaw := crypto.SecureKeyPairs("")
		priKey := string(priKeyRaw)
		pubKey := string(pubKeyRaw)
		req := inputsusers.CreateInput{
			Username:  input.Username,
			PublicKey: pubKey,
			Metadata:  input.Metadata,
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
		trx.PutLink("UserPrivateKey::"+response.User.Id, priKey)
		trx.PutLink("UserEmailToId::"+email, response.User.Id)
		trx.PutLink("UserIdToEmail::"+response.User.Id, email)
		log.Println("saving email:", "["+email+"]", "["+response.User.Id+"]")
		return outputsusers.LoginOutput{User: response.User, Session: response.Session, PrivateKey: priKey}, nil
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
	trx.PutJson("UserMeta::"+user.Id, "metadata", input.Metadata, false)
	meta, err := trx.GetJson("UserMeta::"+user.Id, "metadata.public.profile")
	if err != nil {
		log.Println(err)
		return nil, err
	}
	if meta["name"] == nil {
		err := errors.New("name can't be empty")
		log.Println(err)
		return nil, err
	}
	trx.PutIndex("User", "name", "id", user.Id+"->"+meta["name"].(string), []byte(state.Info().UserId()))
	return outputsusers.CreateOutput{User: user, Session: session}, nil
}

// Update /users/update check [ true false false ] access [ true false false false POST ]
func (a *Actions) Update(state state.IState, input inputsusers.UpdateInput) (any, error) {
	trx := state.Trx()
	meta, err := trx.GetJson("UserMeta::"+state.Info().UserId(), "metadata.public.profile")
	if err != nil {
		log.Println(err)
		return nil, err
	}
	trx.DelIndex("User", "name", "id", state.Info().UserId()+"->"+meta["name"].(string))
	trx.PutJson("UserMeta::"+state.Info().UserId(), "metadata", input.Metadata, true)
	meta, err = trx.GetJson("UserMeta::"+state.Info().UserId(), "metadata.public.profile")
	if err != nil {
		log.Println(err)
		return nil, err
	}
	if meta["name"] == nil {
		err := errors.New("name can't be empty")
		log.Println(err)
		return nil, err
	}
	trx.PutIndex("User", "name", "id", state.Info().UserId()+"->"+meta["name"].(string), []byte(state.Info().UserId()))
	return map[string]any{}, nil
}

// Meta /users/meta check [ true false false ] access [ true false false false GET ]
func (a *Actions) Meta(state state.IState, input inputsusers.MetaInput) (any, error) {
	trx := state.Trx()
	if !trx.HasObj("User", input.UserId) {
		return nil, errors.New("user not found")
	}
	if (state.Info().UserId() != input.UserId) && strings.HasPrefix(input.Path, "private.") {
		return nil, errors.New("access denied")
	}
	metadata, err := trx.GetJson("UserMeta::"+input.UserId, "metadata."+input.Path)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return map[string]any{"metadata": metadata}, nil
}

// Get /users/get check [ true false false ] access [ true false false false GET ]
func (a *Actions) Get(state state.IState, input inputsusers.GetInput) (any, error) {
	trx := state.Trx()
	if !trx.HasObj("User", input.UserId) {
		return nil, errors.New("user not found")
	}
	user := models.User{Id: input.UserId}.Pull(trx)
	result := map[string]any{
		"id":        user.Id,
		"publicKey": user.PublicKey,
		"type":      user.Typ,
		"username":  user.Username,
	}
	meta, err := trx.GetJson("UserMeta::"+input.UserId, "metadata.public.profile")
	if err != nil {
		log.Println(err)
		return nil, err
	}
	result["name"] = meta["name"]
	result["avatar"] = meta["avatar"]
	if input.UserId == state.Info().UserId() {
		result["balance"] = user.Balance
	}
	return outputsusers.GetOutput{User: result}, nil
}

// Find /users/find check [ true false false ] access [ true false false false GET ]
func (a *Actions) Find(state state.IState, input inputsusers.FindInput) (any, error) {
	trx := state.Trx()
	userId := trx.GetIndex("User", "username", "id", input.Username)
	if userId == "" {
		return nil, errors.New("user not found")
	}
	user := models.User{Id: userId}.Pull(trx)
	result := map[string]any{
		"id":        user.Id,
		"publicKey": user.PublicKey,
		"type":      user.Typ,
		"username":  user.Username,
	}
	meta, err := trx.GetJson("UserMeta::"+userId, "metadata.public.profile")
	if err != nil {
		log.Println(err)
		return nil, err
	}
	result["name"] = meta["name"]
	result["avatar"] = meta["avatar"]
	if userId == state.Info().UserId() {
		result["balance"] = user.Balance
	}
	return outputsusers.GetOutput{User: result}, nil
}

// List /users/list check [ true false false ] access [ true false false false GET ]
func (a *Actions) List(state state.IState, input inputsusers.ListInput) (any, error) {
	trx := state.Trx()
	users, err := models.User{}.Search(trx, input.Offset, input.Count, input.Query, map[string]string{"type": "human"})
	if err != nil {
		log.Println(err)
		return nil, err
	}
	results := []map[string]any{}
	for _, user := range users {
		result := map[string]any{
			"id":        user.Id,
			"publicKey": user.PublicKey,
			"type":      user.Typ,
			"username":  user.Username,
		}
		meta, err := trx.GetJson("UserMeta::"+user.Id, "metadata.public.profile")
		if err != nil {
			log.Println(err)
			return nil, err
		}
		result["name"] = meta["name"]
		result["avatar"] = meta["avatar"]
		results = append(results, result)
	}
	return map[string]any{"users": results}, nil
}
