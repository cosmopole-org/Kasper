package actions_player

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"strconv"
	"time"

	game_inputs_player "kasper/src/plugins/game/inputs/player"
	game_model "kasper/src/plugins/game/model"

	"kasper/src/abstract"
	game_outputs_player "kasper/src/plugins/game/outputs/player"

	social_model "kasper/src/plugins/social/model"
	"kasper/src/shell/api/model"
	"kasper/src/shell/layer1/adapters"
	states "kasper/src/shell/layer1/module/state"
	"strings"

	"github.com/robertkrimen/otto"
	"gorm.io/gorm"
)

type Actions struct {
	Layer abstract.ILayer
}

func Install(s adapters.IStorage, a *Actions) error {
	return nil
}

var increaments = map[string]map[string]string{
	"hokm": {
		"video": "hokmgemad",
	},
}

// Inc /player/inc check [ true false false ] access [ true false false false POST ]
func (a *Actions) Inc(s abstract.IState, input game_inputs_player.IncInput) (any, error) {
	var state = abstract.UseState[states.IStateL1](s)
	trx := state.Trx()

	game, ok := increaments[input.GameKey]
	if !ok {
		return nil, errors.New("game not found")
	}
	incKeyData, ok2 := game[input.IncKey]
	if !ok2 {
		return nil, errors.New("increment key not found")
	}

	meta := game_model.Meta{Id: input.GameKey + "::buy"}
	err4 := trx.Db().First(&meta).Error
	if err4 != nil {
		return nil, err4
	}
	val := meta.Data[incKeyData].(string)
	data := strings.Split(val, ".")
	effects := map[string]float64{}
	for i := range data {
		if i%2 == 0 {
			number, err5 := strconv.ParseFloat(data[i+1], 64)
			if err5 != nil {
				fmt.Println(err5)
				continue
			}
			effects[data[i]] = number
		}
	}
	gameDataStr := ""
	trx.Db().Model(&model.User{}).Select(adapters.BuildJsonFetcher("metadata", input.GameKey)).Where("id = ?", state.Info().UserId()).First(&gameDataStr)
	trx.ClearError()
	userData := map[string]interface{}{}
	err6 := json.Unmarshal([]byte(gameDataStr), &userData)
	if err6 != nil {
		log.Println(err6)
		return nil, err6
	}
	for k, v := range effects {
		if v == 0 {
			continue
		}
		timeKey := "last" + (strings.ToUpper(string(k[0])) + k[1:]) + "Buy"
		now := int64(time.Now().UnixMilli())
		oldValRaw, ok := userData[k]
		if !ok {
			continue
		}
		oldVal := oldValRaw.(float64)
		newVal := v + oldVal
		lastBuyTimeRaw, ok2 := userData[timeKey]
		if k == "chat" && ok2 {
			lastBuyTime := lastBuyTimeRaw.(float64)
			if float64(now) < (lastBuyTime + (24 * 60 * 60 * 1000 * oldVal)) {
				newVal = math.Ceil(v + oldVal - ((float64(now) - lastBuyTime) / (24 * 60 * 60 * 1000)))
			} else {
				newVal = v
			}
		}
		err := adapters.UpdateJson(func() *gorm.DB { return trx.Db().Model(&model.User{}).Where("id = ?", state.Info().UserId()) }, &model.User{Id: state.Info().UserId()}, "metadata", input.GameKey+"."+k, newVal)
		if err != nil {
			log.Println(err)
			return map[string]any{}, err
		}
		err2 := adapters.UpdateJson(func() *gorm.DB { return trx.Db().Model(&model.User{}).Where("id = ?", state.Info().UserId()) }, &model.User{Id: state.Info().UserId()}, "metadata", input.GameKey+"."+timeKey, now)
		if err2 != nil {
			log.Println(err2)
			return map[string]any{}, err2
		}
	}
	return game_outputs_player.UpdateOutput{}, nil
}

// Update /player/update check [ true false false ] access [ true false false false POST ]
func (a *Actions) Update(s abstract.IState, input game_inputs_player.UpdateInput) (any, error) {
	var state = abstract.UseState[states.IStateL1](s)
	trx := state.Trx()
	for key, value := range input.Data {
		err := adapters.UpdateJson(func() *gorm.DB { return trx.Db().Model(&model.User{}).Where("id = ?", state.Info().UserId()) }, &model.User{Id: state.Info().UserId()}, "metadata", input.GameKey+"."+key, value)
		if err != nil {
			log.Println(err)
			return nil, err
		}
	}
	return game_outputs_player.UpdateOutput{}, nil
}

func runJs(code string, params map[string]any) float64 {
	vm := otto.New()
	for k, v := range params {
		vm.Set(k, v)
	}
	res, err := vm.Run(`(` + code + `)();`)
	if err != nil {
		log.Println(err)
		return 0
	}
	r, err := res.ToFloat()
	if err != nil {
		log.Println(err)
		return 0
	}
	return r
}

var jsStore = map[string]string{
	"hokm->league": `
		function() {
			var leagues = [
				{
					level: 0,
					start: 0,
				},
				{
					level: 1,
					start: 250,
				},
				{
					level: 2,
					start: 500,
				},
				{
					level: 3,
					start: 1000,
				},
				{
					level: 4,
					start: 2000,
				},
			];
			var shardLevel = 0;
			for (var i = leagues.length - 1; i >= 0; i--) {
				if (leagues[i].start <= score) {
					shardLevel = leagues[i].level;
					break;
				}
			}
			return shardLevel;
		}
	`,
	"hokm->maxXp": `
		function() {
			var temp = xp;
			var counter = 1;
			var totalMaxXp = 2;
			while (temp > 0) {
				var amount = Math.pow(2, counter);
				temp -= amount;
				if (temp >= 0) {
					counter++;
					totalMaxXp += Math.pow(2, counter);
				}
			}
			return totalMaxXp;
		}
	`,
	"hokm->level": `
		function() {
			var temp = xp;
			var counter = 1;
			var totalMaxXp = 2;
			while (temp > 0) {
				var amount = Math.pow(2, counter);
				temp -= amount;
				if (temp >= 0) {
					counter++;
					totalMaxXp += Math.pow(2, counter);
				}
			}
			return counter;
		}
	`,
	"hokm->chatCharge": `
		function() {
			var now = Date.now();
			return Math.max(0, Math.ceil(((lastChatBuy + (chat * 24 * 60 * 60 * 1000)) - now) / (24 * 60 * 60 * 1000)));
		}
	`,
}

var playerDataFilter = map[string][]string{
	"hokm": {
		"gem",
		"chat",
		"lastChatBuy",
		"banned",
		"maxXp",
		"purchase",
		"friendReqs",
		"unreadChats",
		"newlyCreated",
	},
}

// Get /player/get check [ true false false ] access [ true false false false POST ]
func (a *Actions) Get(s abstract.IState, input game_inputs_player.GetInput) (any, error) {
	var state = abstract.UseState[states.IStateL1](s)
	trx := state.Trx()

	gameDataStr := ""
	userId := input.UserId
	if userId == "" {
		userId = state.Info().UserId()
	}

	err := trx.Db().Model(&model.User{}).Select(adapters.BuildJsonFetcher("metadata", input.GameKey)).Where("id = ?", userId).First(&gameDataStr).Error
	if err != nil {
		log.Println(err)
		return game_outputs_player.GetOutput{Data: map[string]interface{}{}}, nil
	} else {
		rawResult := map[string]interface{}{}
		err2 := json.Unmarshal([]byte(gameDataStr), &rawResult)
		if err2 != nil {
			log.Println(err2)
			return nil, err2
		}
		result := map[string]any{}
		if userId == state.Info().UserId() {
			for k, v := range rawResult {
				result[k] = v
			}
		} else {
			filters := map[string]bool{}
			for _, f := range playerDataFilter[input.GameKey] {
				filters[f] = true
			}
			for k, v := range rawResult {
				if filters[k] {
					continue
				}
				result[k] = v
			}
		}
		funcWord := "func("
		jsWord := "js("
		countWord := "count("
		for k, v := range result {
			val, ok := v.(string)
			if ok {
				if strings.HasPrefix(val, funcWord) {
					content := val[len(funcWord) : len(val)-1]
					if strings.HasPrefix(content, jsWord) {
						fnName := content[len(jsWord) : len(content)-1]
						code := jsStore[input.GameKey+"->"+fnName]
						otuput := runJs(code, result)
						result[k] = otuput
					} else if strings.HasPrefix(content, countWord) {
						arg := content[len(countWord) : len(content)-1]
						if arg == "friends" {
							trx.ClearError()
							var interactions = []model.Interaction{}
							err := trx.Db().Select("*").Where("user_ids LIKE ?", userId+"|%").Or("user_ids LIKE ?", "%|"+userId+"::"+a.Layer.Core().Id()).Or("user_ids LIKE ?", "%|"+userId+"::global").Or("user_ids LIKE ?", "%|"+userId).Find(&interactions).Error
							if err != nil {
								log.Println(err)
								trx.ClearError()
								continue
							}
							countOfFriends := int64(0)
							for _, interaction := range interactions {
								if interaction.State["areFriends"] == "true" {
									countOfFriends++
								}
							}
							result[k] = countOfFriends
							trx.ClearError()
						} else if arg == "friendReqs" {
							trx.ClearError()
							var interactions = []model.Interaction{}
							err := trx.Db().Select("*").Where("user_ids LIKE ?", userId+"|%").Or("user_ids LIKE ?", "%|"+userId+"::"+a.Layer.Core().Id()).Or("user_ids LIKE ?", "%|"+userId+"::global").Or("user_ids LIKE ?", "%|"+userId).Find(&interactions).Error
							if err != nil {
								log.Println(err)
								trx.ClearError()
								continue
							}
							countOfFriendReqs := int64(0)
							for _, interaction := range interactions {
								var participantId string
								var userIds = interaction.UserIds
								if strings.HasSuffix(userIds, "::"+a.Layer.Core().Id()) {
									userIds = strings.Split(userIds, "::")[0]
								}
								var idPair = strings.Split(userIds, "|")
								if idPair[0] == userId {
									participantId = idPair[1]
								} else {
									participantId = idPair[0]
								}
								if interaction.State[fmt.Sprintf("%s::requested::%s", participantId, userId)] == "true" {
									countOfFriendReqs++
								}
							}
							result[k] = countOfFriendReqs
							trx.ClearError()
						} else if arg == "unreadChats" {
							trx.ClearError()
							var interactions = []model.Interaction{}
							err := trx.Db().Select("*").Where("user_ids LIKE ?", userId+"|%").Or("user_ids LIKE ?", "%|"+userId+"::"+a.Layer.Core().Id()).Or("user_ids LIKE ?", "%|"+userId+"::global").Or("user_ids LIKE ?", "%|"+userId).Find(&interactions).Error
							if err != nil {
								log.Println(err)
								trx.ClearError()
								continue
							}
							trx.ClearError()
							countOfUnreadChats := int64(0)
							for _, interaction := range interactions {
								ti := interaction.State["topicId"]
								if ti == nil {
									continue
								}
								topicId := ti.(string)
								seen := social_model.Seen{}
								trx.Db().Model(&social_model.Seen{}).Where("id = ?", topicId+"_"+userId).First(&seen)
								trx.ClearError()
								lastMsg := social_model.Message{}
								trx.Db().Model(&social_model.Message{}).Where("topic_id = ?", topicId).Last(&lastMsg)
								trx.ClearError()
								if (lastMsg.AuthorId == userId) || (lastMsg.Id == seen.LastMsgId) || (lastMsg.Id == "") {
									continue
								}
								countOfUnreadChats++
							}
							result[k] = countOfUnreadChats
							trx.ClearError()
						}
					}
				}
			}
		}
		return game_outputs_player.GetOutput{UserId: userId, Data: result}, nil
	}
}
