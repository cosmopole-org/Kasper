package actions_player

import (
	"errors"
	"kasper/src/abstract"
	admin_inputs_meta "kasper/src/plugins/admin/inputs/meta"
	admin_outputs_meta "kasper/src/plugins/admin/outputs/meta"
	game_model "kasper/src/plugins/game/model"
	"kasper/src/shell/layer1/adapters"
	states "kasper/src/shell/layer1/module/state"
	"log"
)

type Actions struct {
	Layer abstract.ILayer
}

func Install(s adapters.IStorage, a *Actions) error {
	return s.AutoMigrate(&game_model.Meta{})
}

// Update /admin/meta/update check [ true false false ] access [ true false false false POST ]
func (a *Actions) Update(s abstract.IState, input admin_inputs_meta.UpdateInput) (any, error) {
	var state = abstract.UseState[states.IStateL1](s)
	if !state.Info().IsGod() {
		return nil, errors.New("access denied")
	}
	trx := state.Trx()
	id := input.GameKey
	if input.Tag != "" {
		id += "::" + input.Tag
	}
	meta := game_model.Meta{Id: id}
	err := trx.Db().First(&meta).Error
	if err != nil {
		log.Println(err)
		trx.ClearError()
	}
	if meta.Data == nil {
		meta.Data = map[string]interface{}{}
	}
	for key, value := range input.Data {
		meta.Data[key] = value
	}
	err2 := trx.Db().Save(&meta).Error
	if err2 != nil {
		return nil, err2
	}
	return admin_outputs_meta.UpdateOutput{}, nil
}

// Get /admin/meta/get check [ false false false ] access [ true false false false GET ]
func (a *Actions) Get(s abstract.IState, input admin_inputs_meta.GetInput) (any, error) {
	var state = abstract.UseState[states.IStateL1](s)
	id := input.GameKey
	if input.Tag != "" {
		id += "::" + input.Tag
	}
	trx := state.Trx()
	meta := game_model.Meta{Id: id}
	err := trx.Db().First(&meta).Error
	if err != nil {
		return nil, err
	}
	return admin_outputs_meta.GetOutput{Data: meta.Data}, nil
}

// Read /admin/meta/read check [ false false false ] access [ true false false false GET ]
func (a *Actions) Read(s abstract.IState, input admin_inputs_meta.GetInput) (any, error) {
	var state = abstract.UseState[states.IStateL1](s)
	trx := state.Trx()
	metas := []game_model.Meta{}
	trx.Db().Where("id like '" + input.GameKey + "::%'").Find(&metas)
	trx.ClearError()
	meta := game_model.Meta{Id: input.GameKey}
	err := trx.Db().First(&meta).Error
	if err == nil {
		metas = append(metas, meta)
	}
	trx.ClearError()
	return admin_outputs_meta.ReadOutput{Data: metas}, nil
}
