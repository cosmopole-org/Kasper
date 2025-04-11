package actions_auth

import (
	"errors"
	"kasper/src/abstract"
	admin_inputs_auth "kasper/src/plugins/admin/inputs/auth"
	admin_outputs_auth "kasper/src/plugins/admin/outputs/auth"
	"kasper/src/shell/api/model"
	"kasper/src/shell/layer1/adapters"
	states "kasper/src/shell/layer1/module/state"
	"os"
)

type Actions struct {
	Layer abstract.ILayer
}

func Install(s adapters.IStorage, a *Actions) error {
	return nil
}

// Login /admin/auth/login check [ false false false ] access [ true false false false POST ]
func (a *Actions) Login(s abstract.IState, input admin_inputs_auth.AdminInput) (any, error) {
	state := abstract.UseState[states.IStateL1](s)
	user := model.User{}
	trx := state.Trx()
	if input.Password != os.Getenv("AdminPassword") {
		return nil, errors.New("access denied")
	}
	err := trx.Db().Where("username = ?", input.Email).First(&user).Error
	if err != nil {
		return nil, err
	}
	if user.Id == "" {
		return nil, errors.New("access denied")
	}
	session := model.Session{}
	trx.Db().Where("user_id = ?", user.Id).First(&session)
	return admin_outputs_auth.LoginOutput{Token: session.Token}, nil
}

// ChangePass /admin/auth/changePass check [ true false false ] access [ true false false false POST ]
func (a *Actions) ChangePass(s abstract.IState, input admin_inputs_auth.ChangePassInput) (any, error) {
	state := abstract.UseState[states.IStateL1](s)
	if !state.Info().IsGod() {
		return nil, errors.New("access denied")
	}
	if len(input.Password) == 0 {
		return nil, errors.New("error: invalid password")
	}
	trx := state.Trx()
	user := model.User{Id: state.Info().UserId()}
	err := trx.Db().First(&user).Error
	if err != nil {
		return err, nil
	}
	trx.Db().Save(&user)
	return admin_outputs_auth.ChangePassOutput{}, nil
}
