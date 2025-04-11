package actions_user

import (
	"fmt"
	"kasper/src/abstract"
	moduleactormodel "kasper/src/core/module/actor/model"
	inputs_spaces "kasper/src/shell/api/inputs/spaces"
	inputsusers "kasper/src/shell/api/inputs/users"
	models "kasper/src/shell/api/model"
	outputsusers "kasper/src/shell/api/outputs/users"
	"kasper/src/shell/layer1/adapters"
	modulestate "kasper/src/shell/layer1/module/state"
	toolbox2 "kasper/src/shell/layer1/module/toolbox"
	"kasper/src/shell/utils/crypto"
	"kasper/src/shell/utils/future"
	"log"
	"strconv"

	"github.com/mitchellh/mapstructure"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func convertRowIdToCode(rowId uint) string {
	idStr := fmt.Sprintf("%d", rowId)
	for len(idStr) < 6 {
		idStr = "0" + idStr
	}
	var c = ""
	for i := 0; i < len(idStr); i++ {
		if i < 3 {
			digit, err := strconv.ParseInt(idStr[i:i+1], 10, 32)
			if err != nil {
				fmt.Println(err)
				return ""
			}
			c += string(rune('A' + digit))
		} else {
			c += idStr[i : i+1]
		}
	}
	return c
}

type Actions struct {
	Layer abstract.ILayer
}

func Install(s adapters.IStorage, a *Actions) error {
	toolbox := abstract.UseToolbox[*toolbox2.ToolboxL1](a.Layer.Tools())
	err_1 := s.Db().AutoMigrate(&models.User{})
	if err_1 != nil {
		return err_1
	}
	err_2 := s.Db().AutoMigrate(&models.Key{})
	if err_2 != nil {
		return err_2
	}
	err2 := s.Db().AutoMigrate(&models.Session{})
	if err2 != nil {
		return err2
	}
	s.DoTrx(func(trx adapters.ITrx) error {
		for _, godUsername := range a.Layer.Core().Gods() {
			var user = models.User{}
			userId := ""
			err := trx.Db().Where("username = ?", godUsername+"@"+a.Layer.Core().Id()).First(&user).Error
			if err != nil {
				log.Println(err)
				toolbox := abstract.UseToolbox[*toolbox2.ToolboxL1](a.Layer.Tools())
				var (
					user    models.User
					session models.Session
				)
				user = models.User{Metadata: datatypes.JSON([]byte(`{}`)), Id: toolbox.Cache().GenId(trx.Db(), a.Layer.Core().Id()), Typ: "human", PublicKey: "", Username: godUsername + "@" + a.Layer.Core().Id(), Name: "", Avatar: ""}
				err := trx.Db().Create(&user).Error
				if err != nil {
					panic(err)
				}
				trx.Db().First(&user)
				code := convertRowIdToCode(uint(user.Number))
				err2 := adapters.UpdateJson(func() *gorm.DB { return trx.Db().Model(&models.User{}).Where("id = ?", user.Id) }, &user, "metadata", "code", code)
				if err2 != nil {
					panic(err2)
				}
				session = models.Session{Id: toolbox.Cache().GenId(trx.Db(), a.Layer.Core().Id()), Token: crypto.SecureUniqueString(), UserId: user.Id}
				err3 := trx.Db().Create(&session).Error
				if err3 != nil {
					panic(err3)
				}
				_, _, errJoin := a.Layer.Actor().FetchAction("/spaces/join").Act(a.Layer.Sb().NewState(moduleactormodel.NewInfo(user.Id, "", "", ""), trx), inputs_spaces.JoinInput{SpaceId: "main@" + a.Layer.Core().Id()})
				if errJoin != nil {
					panic(errJoin)
				}
				trx.Mem().Put("auth::"+session.Token, fmt.Sprintf("human/%s", user.Id))
				trx.Mem().Put("code::"+code, user.Id)
				userId = user.Id
			} else {
				userId = user.Id
			}
			toolbox.Cache().Put("god::"+userId, "true")
		}
		return nil
	})
	users := []models.User{}
	s.Db().Model(&models.User{}).Find(&users)
	future.Async(func() {
		for _, user := range users {
			toolbox.Cache().Put("code::"+convertRowIdToCode(uint(user.Number)), user.Id)
		}
	}, false)
	return nil
}

// Authenticate /users/authenticate check [ true false false ] access [ true false false false POST ]
func (a *Actions) Authenticate(s abstract.IState, _ inputsusers.AuthenticateInput) (any, error) {
	state := abstract.UseState[modulestate.IStateL1](s)
	_, res, _ := a.Layer.Actor().FetchAction("/users/get").Act(a.Layer.Sb().NewState(moduleactormodel.NewInfo("", "", "", ""), state.Trx()), inputsusers.GetInput{UserId: state.Info().UserId()})
	return outputsusers.AuthenticateOutput{Authenticated: true, User: res.(outputsusers.GetOutput).User}, nil
}

// Login /users/login check [ false false false ] access [ true false false false POST ]
func (a *Actions) Login(s abstract.IState, input inputsusers.LoginInput) (any, error) {
	token := crypto.SecureUniqueString()
	state := abstract.UseState[modulestate.IStateL1](s)
	trx := state.Trx()
	_, res, err2 := a.Layer.Actor().FetchAction("/users/create").(abstract.ISecureAction).SecurelyAct(a.Layer.Core().Get(1), "", "", inputsusers.CreateInput{
		Username:  input.Username,
		Name:      "anonymous",
		Avatar:    "0",
		Token:     token,
		PublicKey: "",
	}, "")
	if err2 != nil {
		return nil, err2
	}
	var response outputsusers.CreateOutput
	mapstructure.Decode(res, &response)
	k := models.Key{Id: response.User.Id}
	err3 := trx.Db().First(&k).Error
	if err3 != nil {
		trx.ClearError()
		k.PublicKey = ""
		k.PrivateKey = ""
		e := trx.Db().Create(&k).Error
		if e != nil {
			log.Println(e)
		}
		_, _, errJoin := a.Layer.Actor().FetchAction("/spaces/join").Act(a.Layer.Sb().NewState(moduleactormodel.NewInfo(response.User.Id, "", "", ""), trx), inputs_spaces.JoinInput{SpaceId: "main@" + a.Layer.Core().Id()})
		if errJoin != nil {
			return nil, errJoin
		}
	}
	return outputsusers.LoginOutput{User: response.User, Session: response.Session, PrivateKey: k.PrivateKey}, nil
}

// Create /users/create check [ false false false ] access [ true false false false POST ]
func (a *Actions) Create(s abstract.IState, input inputsusers.CreateInput) (any, error) {
	toolbox := abstract.UseToolbox[*toolbox2.ToolboxL1](a.Layer.Tools())
	state := abstract.UseState[modulestate.IStateL1](s)
	var (
		user    models.User
		session models.Session
	)
	trx := state.Trx()
	findErr := trx.Db().Model(&models.User{}).Where("username = ?", input.Username+"@"+state.Dummy()).First(&user).Error
	if findErr == nil {
		trx.Db().Where("user_id = ?", user.Id).First(&session)
		return outputsusers.CreateOutput{User: user, Session: session}, nil
	}
	trx.ClearError()
	typ := "human"
	if input.Typ != "" {
		typ = input.Typ
	}
	user = models.User{Metadata: datatypes.JSON([]byte(`{}`)), Id: toolbox.Cache().GenId(trx.Db(), input.Origin()), Typ: typ, PublicKey: input.PublicKey, Username: input.Username + "@" + a.Layer.Core().Id(), Name: input.Name, Avatar: input.Avatar}
	err := trx.Db().Create(&user).Error
	if err != nil {
		return nil, err
	}
	trx.Db().First(&user)
	code := convertRowIdToCode(uint(user.Number))
	err2 := adapters.UpdateJson(func() *gorm.DB { return trx.Db().Model(&models.User{}).Where("id = ?", user.Id) }, &user, "metadata", "code", code)
	if err2 != nil {
		return nil, err2
	}
	session = models.Session{Id: toolbox.Cache().GenId(trx.Db(), input.Origin()), Token: input.Token, UserId: user.Id}
	err3 := trx.Db().Create(&session).Error
	if err3 != nil {
		return nil, err3
	}
	trx.Mem().Put("auth::"+session.Token, fmt.Sprintf("human/%s", user.Id))
	trx.Mem().Put("code::"+code, user.Id)
	return outputsusers.CreateOutput{User: user, Session: session}, nil
}

// UpdateMeta /player/updateMeta check [ true false false ] access [ true false false false POST ]
func (a *Actions) UpdateMeta(s abstract.IState, input inputsusers.UpdateMetaInput) (any, error) {
	var state = abstract.UseState[modulestate.IStateL1](s)
	trx := state.Trx()
	user := models.User{Id: state.Info().UserId()}
	err := trx.Db().First(&user).Error
	if err != nil {
		return nil, err
	}
	for key, value := range input.Data {
		err2 := adapters.UpdateJson(func() *gorm.DB { return trx.Db().Model(&models.User{}).Where("id = ?", state.Info().UserId()) }, &user, "metadata", key, value)
		if err2 != nil {
			log.Println(err2)
			trx.ClearError()
			continue
		}
	}
	return outputsusers.UpdateMetaOutput{}, nil
}

// Update /users/update check [ true false false ] access [ true false false false PUT ]
func (a *Actions) Update(s abstract.IState, input inputsusers.UpdateInput) (any, error) {
	state := abstract.UseState[modulestate.IStateL1](s)
	var user models.User
	trx := state.Trx()
	user = models.User{Id: state.Info().UserId()}
	err := trx.Db().First(&user).Error
	if err != nil {
		return nil, err
	}
	user.Name = input.Name
	user.Avatar = input.Avatar
	user.Username = input.Username + "@midopia"
	err2 := trx.Db().Save(&user).Error
	if err2 != nil {
		return nil, err2
	}
	return outputsusers.UpdateOutput{
		User: models.PublicUser{Id: user.Id, Type: user.Typ, Username: user.Username, Name: user.Name, Avatar: user.Avatar, PublicKey: user.PublicKey},
	}, nil
}

// Get /users/get check [ false false false ] access [ true false false false GET ]
func (a *Actions) Get(s abstract.IState, input inputsusers.GetInput) (any, error) {
	state := abstract.UseState[modulestate.IStateL1](s)
	var user models.User
	trx := state.Trx()
	user = models.User{Id: input.UserId}
	err := trx.Db().First(&user).Error
	if err != nil {
		return nil, err
	}
	return outputsusers.GetOutput{
		User: models.PublicUser{Id: user.Id, Type: user.Typ, Username: user.Username, Name: user.Name, Avatar: user.Avatar, PublicKey: user.PublicKey},
	}, nil
}

// Read /users/read check [ false false false ] access [ true false false false GET ]
func (a *Actions) Read(s abstract.IState, input inputsusers.ReadInput) (any, error) {
	state := abstract.UseState[modulestate.IStateL1](s)
	trx := state.Trx()
	users := []models.User{}
	err := trx.Db().Where("type = ?", input.Typ).Find(&users).Error
	if err != nil {
		return nil, err
	}
	publicUsers := []models.PublicUser{}
	for _, user := range users {
		publicUsers = append(publicUsers, models.PublicUser{Id: user.Id, Type: user.Typ, Username: user.Username, Name: user.Name, Avatar: user.Avatar, PublicKey: user.PublicKey})
	}
	return outputsusers.ReadOutput{
		Users: publicUsers,
	}, nil
}

// Delete /users/delete check [ true false false ] access [ true false false false DELETE ]
func (a *Actions) Delete(s abstract.IState, _ inputsusers.DeleteInput) (any, error) {
	state := abstract.UseState[modulestate.IStateL1](s)
	var user models.User
	trx := state.Trx()
	user = models.User{Id: state.Info().UserId()}
	err := trx.Db().First(&user).Error
	if err != nil {
		return nil, err
	}
	var sessions []models.Session
	trx.Db().Where("user_id = ?", user.Id).Find(&sessions)
	for _, session := range sessions {
		trx.Mem().Del("auth::" + session.Token)
		err := trx.Db().Delete(&session).Error
		if err != nil {
			log.Println(err)
			trx.ClearError()
		}
	}
	err2 := trx.Db().Delete(&user).Error
	if err2 != nil {
		return nil, err2
	}
	return outputsusers.DeleteOutput{User: user}, nil
}
