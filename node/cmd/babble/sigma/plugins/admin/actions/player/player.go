package actions_player

import (
	"encoding/json"
	"errors"
	"kasper/cmd/babble/sigma/abstract"
	models "kasper/cmd/babble/sigma/api/model"
	"kasper/cmd/babble/sigma/layer1/adapters"
	states "kasper/cmd/babble/sigma/layer1/module/state"
	admin_inputs_player "kasper/cmd/babble/sigma/plugins/admin/inputs/player"
	admin_model "kasper/cmd/babble/sigma/plugins/admin/model"
	admin_outputs_player "kasper/cmd/babble/sigma/plugins/admin/outputs/player"
	"log"
	"strconv"

	"gorm.io/gorm"
)

type Actions struct {
	Layer abstract.ILayer
}

func Install(s adapters.IStorage, a *Actions) error {
	return nil
}

// Update /admin/player/update check [ true false false ] access [ true false false false POST ]
func (a *Actions) Update(s abstract.IState, input admin_inputs_player.UpdateInput) (any, error) {
	var state = abstract.UseState[states.IStateL1](s)
	if !state.Info().IsGod() {
		return nil, errors.New("access denied")
	}
	trx := state.Trx()
	for key, value := range input.Data {
		err := adapters.UpdateJson(func() *gorm.DB { return trx.Db().Model(&models.User{}).Where("id = ?", input.UserId) }, &models.User{Id: input.UserId}, "metadata", input.GameKey+"."+key, value)
		if err != nil {
			return nil, err
		}
	}
	return admin_outputs_player.UpdateOutput{}, nil
}

// Get /admin/player/get check [ true false false ] access [ true false false false POST ]
func (a *Actions) Get(s abstract.IState, input admin_inputs_player.GetInput) (any, error) {
	var state = abstract.UseState[states.IStateL1](s)
	if !state.Info().IsGod() {
		return nil, errors.New("access denied")
	}
	trx := state.Trx()
	gameDataStr := ""
	err := trx.Db().Model(&models.User{}).Select(adapters.BuildJsonFetcher("metadata", input.GameKey)).Where("id = ?", input.UserId).First(&gameDataStr).Error
	if err != nil {
		log.Println(err)
		return admin_outputs_player.GetOutput{Data: map[string]interface{}{}}, nil
	} else {
		result := map[string]interface{}{}
		err2 := json.Unmarshal([]byte(gameDataStr), &result)
		if err2 != nil {
			log.Println(err2)
			return nil, err2
		}
		return admin_outputs_player.GetOutput{Data: result}, nil
	}
}

// List /admin/player/list check [ true false false ] access [ true false false false POST ]
func (a *Actions) List(s abstract.IState, input admin_inputs_player.ListInput) (any, error) {
	var state = abstract.UseState[states.IStateL1](s)
	if !state.Info().IsGod() {
		return nil, errors.New("access denied")
	}
	offset, err := strconv.ParseInt(input.Offset, 10, 32)
	if err != nil {
		offset = 0
	}
	count, err := strconv.ParseInt(input.Count, 10, 32)
	if err != nil {
		count = 1
	}
	var result = []admin_model.PlayerMini{}
	var users = []models.User{}
	trx := state.Trx()
	var totalCount int64 = 0
	trx.Db().Model(&models.User{}).Where(adapters.BuildJsonFetcher("metadata", input.GameKey + ".profile") + " is not null").Count(&totalCount)
	trx.Db().Where("username like ?", "%"+input.Query+"%").Or("id like ?", "%"+input.Query+"%").Or(adapters.BuildJsonFetcher("metadata", input.GameKey+".profile.name") + " like ?", "%"+input.Query+"%").Limit(int(count)).Offset(int(offset)).Find(&users)
	for _, user := range users {
		userData := user.Metadata
		dataStr, err := userData.MarshalJSON()
		data := map[string]any{}
		if err != nil {
			log.Println(err)
		} else {
			json.Unmarshal(dataStr, &data)
		}

		if data[input.GameKey] != nil && data[input.GameKey].(map[string]interface{})["profile"] != nil {
			coin, ok1 := data[input.GameKey].(map[string]interface{})["coin"]
			if !ok1 {
				coin = int64(0)
			}
			gem, ok2 := data[input.GameKey].(map[string]interface{})["gem"]
			if !ok2 {
				gem = int64(0)
			}
			energy, ok3 := data[input.GameKey].(map[string]interface{})["energy"]
			if !ok3 {
				energy = int32(0)
			}
			result = append(result, admin_model.PlayerMini{
				Id:         user.Id,
				Email:      user.Username[:(len(user.Username) - len("@midopia"))],
				PlayerName: data[input.GameKey].(map[string]interface{})["profile"].(map[string]interface{})["name"].(string),
				Coin:       coin,
				Gem:        gem,
				Energy:     energy,
			})
		}
	}
	return admin_outputs_player.ListOutput{Players: result, TotalCount: totalCount}, nil
}
