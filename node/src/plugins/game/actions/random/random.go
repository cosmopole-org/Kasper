package actions_player

import (
	"encoding/json"
	"errors"
	"fmt"
	"kasper/src/abstract"
	game_inputs_daily "kasper/src/plugins/game/inputs/random"
	game_model "kasper/src/plugins/game/model"
	game_outputs_daily "kasper/src/plugins/game/outputs/random"
	"kasper/src/shell/api/model"
	"kasper/src/shell/layer1/adapters"
	states "kasper/src/shell/layer1/module/state"
	"log"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

type Actions struct {
	Layer abstract.ILayer
}

func Install(s adapters.IStorage, a *Actions) error {
	return nil
}

// Daily /random/daily check [ true false false ] access [ true false false false GET ]
func (a *Actions) Daily(s abstract.IState, input game_inputs_daily.DailyInput) (any, error) {
	var state = abstract.UseState[states.IStateL1](s)
	trx := state.Trx()

	gameKey := "hokm"

	rndIndex := rand.Intn(8) + 1

	gameDataStr := ""
	trx.Db().Model(&model.User{}).Select(adapters.BuildJsonFetcher("metadata", gameKey)).Where("id = ?", state.Info().UserId()).First(&gameDataStr)
	trx.ClearError()
	userData := map[string]interface{}{}
	err6 := json.Unmarshal([]byte(gameDataStr), &userData)
	if err6 != nil {
		log.Println(err6)
		return nil, err6
	}

	lastDR := float64(0)
	lastDRRaw, ok2 := userData["lastDailyRewardReset"]
	if ok2 {
		lastDR = lastDRRaw.(float64)
	}
	drCount := float64(0)
	drCountRaw, ok2 := userData["dailyRewardCount"]
	if ok2 {
		drCount = drCountRaw.(float64)
	}

	if lastDR == 0 {
		lastDR = float64(time.Now().UnixMilli() - 1)
		adapters.UpdateJson(func() *gorm.DB { return trx.Db().Model(&model.User{}).Where("id = ?", state.Info().UserId()) }, &model.User{Id: state.Info().UserId()}, "metadata", gameKey+".lastDailyRewardReset", lastDR)
		trx.ClearError()
	}

	drCount++

	if drCount > 2 {
		if (float64(time.Now().UnixMilli()) - lastDR) > (24 * 60 * 60 * 1000) {
			drCount = 1
			adapters.UpdateJson(func() *gorm.DB { return trx.Db().Model(&model.User{}).Where("id = ?", state.Info().UserId()) }, &model.User{Id: state.Info().UserId()}, "metadata", gameKey+".lastDailyRewardReset", time.Now().UnixMilli())
			trx.ClearError()
		} else {
			return nil, errors.New("daily reward limit reached")
		}
	}

	adapters.UpdateJson(func() *gorm.DB { return trx.Db().Model(&model.User{}).Where("id = ?", state.Info().UserId()) }, &model.User{Id: state.Info().UserId()}, "metadata", gameKey+".dailyRewardCount", drCount)
	trx.ClearError()

	_, ok := userData["newlyCreated"]

	if ok && (userData["newlyCreated"].(bool)) {
		rndIndex = 4
	}

	meta := game_model.Meta{Id: gameKey}
	trx.Db().First(&meta)
	trx.ClearError()
	val := meta.Data[fmt.Sprintf("dailyreward%d", rndIndex)].(string)

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

	user := model.User{Id: state.Info().UserId()}

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
		err := adapters.UpdateJson(func() *gorm.DB { return trx.Db().Model(&model.User{}).Where("id = ?", state.Info().UserId()) }, &user, "metadata", gameKey+"."+k, newVal)
		if err != nil {
			log.Println(err)
			return map[string]any{}, err
		}
		trx.ClearError()
		err2 := adapters.UpdateJson(func() *gorm.DB { return trx.Db().Model(&model.User{}).Where("id = ?", state.Info().UserId()) }, &user, "metadata", gameKey+"."+timeKey, now)
		if err2 != nil {
			log.Println(err2)
			return map[string]any{}, err2
		}
		trx.ClearError()
	}

	adapters.UpdateJson(func() *gorm.DB { return trx.Db().Model(&model.User{}).Where("id = ?", state.Info().UserId()) }, &user, "metadata", gameKey+".newlyCreated", false)
	trx.ClearError()

	adapters.UpdateJson(func() *gorm.DB { return trx.Db().Model(&model.User{}).Where("id = ?", state.Info().UserId()) }, &model.User{Id: state.Info().UserId()}, "metadata", gameKey+".lastDailyReward", time.Now().UnixMilli())
	trx.ClearError()

	return game_outputs_daily.DailyOutput{Number: rndIndex - 1}, nil
}
