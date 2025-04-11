package actions_board

import (
	"encoding/json"
	"errors"
	"kasper/cmd/babble/sigma/abstract"
	"kasper/cmd/babble/sigma/api/model"
	"kasper/cmd/babble/sigma/layer1/adapters"
	states "kasper/cmd/babble/sigma/layer1/module/state"
	toolbox "kasper/cmd/babble/sigma/layer1/module/toolbox"
	game_inputs_board "kasper/cmd/babble/sigma/plugins/game/inputs/board"
	game_model "kasper/cmd/babble/sigma/plugins/game/model"
	game_outputs_board "kasper/cmd/babble/sigma/plugins/game/outputs/board"
	game_memory "kasper/cmd/babble/sigma/plugins/game/tools/memory"
	"log"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Actions struct {
	Layer abstract.ILayer
}

func Install(s adapters.IStorage, a *Actions) error {
	return s.AutoMigrate(&game_model.StoredFormula{})
}

type LbShard struct {
	Level string
	Start float64
}

type ShardGroup struct {
	Param  string
	Shards []LbShard
}

var lbShards = map[string]ShardGroup{
	"hokm_3": {
		Param: "score",
		Shards: []LbShard{
			{
				Level: "3_1",
				Start: 0,
			},
			{
				Level: "3_2",
				Start: 250,
			},
			{
				Level: "3_3",
				Start: 500,
			},
			{
				Level: "3_4",
				Start: 1000,
			},
			{
				Level: "3_5",
				Start: 2000,
			},
		},
	},
}

// Submit /board/submit check [ true false false ] access [ true false false false POST ]
func (a *Actions) Submit(s abstract.IState, input game_inputs_board.SubmitInput) (any, error) {
	var state = abstract.UseState[states.IStateL1](s)
	var tb = abstract.UseToolbox[toolbox.IToolboxL1](a.Layer.Tools())
	trx := state.Trx()

	sFormula := game_model.StoredFormula{}
	err := trx.Db().Model(&game_model.StoredFormula{}).
		Where("game_key = ?", input.GameKey).
		Where("level = ?", input.Level).
		First(&sFormula).
		Error
	if err != nil {
		return nil, err
	}
	trx.ClearError()
	forStr, errJson := json.Marshal(sFormula.Data)
	if errJson != nil {
		return nil, errJson
	}
	forMem := game_model.MemFormula{}
	errJson2 := json.Unmarshal(forStr, &forMem)
	if errJson2 != nil {
		return nil, errJson2
	}
	formula := game_model.Formula{GameKey: sFormula.GameKey, Level: sFormula.Level, Data: forMem}
	score := float64(0)
	for i := 0; i < len(formula.Data.Keys); i++ {
		_, ok := input.Data[formula.Data.Keys[i]]
		if !ok {
			return nil, errors.New("value key not found")
		}
	}
	for i := 0; i < len(formula.Data.Keys); i++ {
		key := formula.Data.Keys[i]
		weight := formula.Data.Weights[i]
		nonZero := formula.Data.NonZero[i]
		val, okV := input.Data[key]
		if !okV {
			return nil, errors.New("value key not found")
		}
		rule, isRule := formula.Data.Rules[key]
		value := float64(0)
		if isRule {
			valueStr, ok := val.(string)
			if !ok {
				return nil, errors.New("rule not valid")
			}
			valNum, ok3 := rule[valueStr]
			if !ok3 {
				return nil, errors.New("rule not found")
			} else {
				value = valNum
			}
		} else {
			valueNum, ok := val.(float64)
			if !ok {
				return nil, errors.New("invalid number value")
			}
			value = valueNum
		}
		if (value == 0) && nonZero {
			return nil, errors.New("0 for this key is not valid")
		}
		score += (value * weight)
		trx.ClearError()
		if formula.Data.Operations[i] == "sum" {
			oldValue := float64(0)
			err2 := trx.Db().Model(&model.User{}).Select(adapters.BuildJsonFetcher("metadata", input.GameKey + ".board." + input.Level + "." + key)).Where("id = ?", state.Info().UserId()).First(&oldValue).Error
			if err2 != nil {
				log.Println(err2)
			}
			trx.ClearError()
			newValue := oldValue + value
			log.Println(oldValue, value, newValue)
			err3 := adapters.UpdateJson(
				func() *gorm.DB {
					return trx.Db().Model(&model.User{}).Where("id = ?", state.Info().UserId())
				},
				&model.User{Id: state.Info().UserId()}, "metadata", input.GameKey+".board."+input.Level+"."+key, newValue,
			)
			if err3 != nil {
				log.Println(err3)
				return nil, err3
			}
		} else if formula.Data.Operations[i] == "replace_smaller" {
			oldValue := float64(0)
			err2 := trx.Db().Model(&model.User{}).Select(adapters.BuildJsonFetcher("metadata", input.GameKey + ".board." + input.Level + "." + key)).Where("id = ?", state.Info().UserId()).First(&oldValue).Error
			if err2 != nil {
				log.Println(err2)
			}
			trx.ClearError()
			if (err2 != nil) || (oldValue > value) {
				newValue := value
				err3 := adapters.UpdateJson(func() *gorm.DB { return trx.Db().Model(&model.User{}).
					Where("id = ?", state.Info().UserId()) }, &model.User{Id: state.Info().UserId()}, "metadata", input.GameKey+".board."+input.Level+"."+key, newValue)
				if err3 != nil {
					log.Println(err3)
					return nil, err3
				}
			}
		}
	}
	if formula.Data.TotalOp == "replace_smaller" {
		if score > 0 {
			oldValue := game_memory.FindSCore(tb.Cache(), state.Info().UserId(), input.GameKey, input.Level)
			if (oldValue == 0) || (score < oldValue) {
				game_memory.Replace(tb.Cache(), state.Info().UserId(), score, input.GameKey, input.Level)
			}
		}
	} else if formula.Data.TotalOp == "sum" {
		game_memory.IncrAndReturn(tb.Cache(), state.Info().UserId(), input.GameKey, input.Level, score)
	}
	return game_outputs_board.SubmitOutput{}, nil
}

// Get /board/get check [ true false false ] access [ true false false false POST ]
func (a *Actions) Get(s abstract.IState, input game_inputs_board.GetInput) (any, error) {
	var state = abstract.UseState[states.IStateL1](s)
	var tb = abstract.UseToolbox[toolbox.IToolboxL1](a.Layer.Tools())
	trx := state.Trx()

	level := input.Level
	key := input.GameKey + "_" + level
	shards, ok := lbShards[key]
	if ok {
		p := float64(0)
		trx.Db().Model(&model.User{}).Select(adapters.BuildJsonFetcher("metadata", input.GameKey + "." + shards.Param)).Where("id = ?", state.Info().UserId()).First(&p)
		trx.ClearError()
		for i := len(shards.Shards) - 1; i >= 0; i-- {
			if shards.Shards[i].Start <= p {
				level = shards.Shards[i].Level
				break
			}
		}
	}

	sFormula := game_model.StoredFormula{}
	gameErr := trx.Db().Model(&game_model.StoredFormula{}).
		Where("game_key = ?", input.GameKey).
		Where("level = ?", level).
		First(&sFormula).
		Error
	if gameErr != nil {
		return nil, errors.New("game or level not found")
	}
	trx.ClearError()
	forStr, errJson := json.Marshal(sFormula.Data)
	if errJson != nil {
		return nil, errJson
	}
	forMem := game_model.MemFormula{}
	errJson2 := json.Unmarshal(forStr, &forMem)
	if errJson2 != nil {
		return nil, errJson2
	}
	var topPlayers [100]game_memory.TopPlayer
	var count = 0
	if forMem.Order == "asc" {
		topPlayers, count = game_memory.TopPlayers(tb.Cache(), input.GameKey, level, true)
	} else {
		topPlayers, count = game_memory.TopPlayers(tb.Cache(), input.GameKey, level, false)
	}
	var ids = []string{}
	var result = []game_outputs_board.PreparedPlayer{}
	var tps = map[string]game_memory.TopPlayer{}
	for _, tp := range topPlayers {
		ids = append(ids, tp.UserId)
		tps[tp.UserId] = tp
	}
	type playerdata struct {
		UserId      string
		GameBoard   datatypes.JSON
		GameProfile datatypes.JSON
		GameLevel   datatypes.JSON
	}
	metas := []playerdata{}
	err := trx.Db().Model(&model.User{}).Select(
		"id as user_id, " + adapters.BuildJsonFetcher("metadata", input.GameKey + ".board") + " as game_board, " + adapters.BuildJsonFetcher("metadata", input.GameKey + ".profile") + " as game_profile, " + adapters.BuildJsonFetcher("metadata", input.GameKey + ".board." + level) + " as game_level").
		Where("id in ?", ids).
		Where(adapters.BuildJsonFetcher("metadata", input.GameKey + ".board." + level) + " IS NOT NULL and " + adapters.BuildJsonFetcher("metadata", input.GameKey + ".profile") + " IS NOT NULL and " + adapters.BuildJsonFetcher("metadata", input.GameKey + ".board") + " IS NOT NULL").
		Find(&metas).
		Error
	if err != nil {
		log.Println(err)
		return nil, err
	}
	var usersDict = map[string]playerdata{}
	for _, user := range metas {
		usersDict[user.UserId] = user
	}
	for _, tp := range topPlayers[0:count] {
		user, ok := usersDict[tp.UserId]
		if !ok {
			continue
		}
		profile := user.GameProfile
		levelData := user.GameLevel
		result = append(result, game_outputs_board.PreparedPlayer{UserId: tp.UserId, Profile: profile, Score: levelData})
	}
	return game_outputs_board.GetOutput{Players: result}, nil
}

// Winner /board/winner check [ true false false ] access [ true false false false POST ]
func (a *Actions) Winner(s abstract.IState, input game_inputs_board.WinnerInput) (any, error) {
	var state = abstract.UseState[states.IStateL1](s)
	trx := state.Trx()

	winnerData := ""
	err := trx.Db().Model(&model.User{}).Select(adapters.BuildJsonFetcher("metadata", input.GameKey + ".board." + input.Level + ".old")).Where("id = ?", state.Info().UserId()).First(&winnerData).Error
	if err != nil {
		log.Println(err)
		return game_outputs_board.RankOutput{}, nil
	}
	res := map[string]any{}
	err2 := json.Unmarshal([]byte(winnerData), &res)
	if err2 != nil {
		log.Println(err2)
		return game_outputs_board.RankOutput{}, nil
	}
	return res, nil
}

// Reward /board/reward check [ true false false ] access [ true false false false POST ]
func (a *Actions) Reward(s abstract.IState, input game_inputs_board.RewardInput) (any, error) {
	var state = abstract.UseState[states.IStateL1](s)
	trx := state.Trx()

	var user = model.User{Id: state.Info().UserId()}
	trx.Db().First(&user)
	err := adapters.UpdateJson(func() *gorm.DB { return trx.Db().Model(&model.User{}).Where("id = ?", state.Info().UserId()) }, &model.User{Id: state.Info().UserId()}, "metadata", input.GameKey+".board."+input.Level, map[string]interface{}{})
	if err != nil {
		log.Println(err)
		return game_outputs_board.RewardOutput{}, nil
	}
	return game_outputs_board.RewardOutput{}, nil
}

// Rank /board/rank check [ true false false ] access [ true false false false POST ]
func (a *Actions) Rank(s abstract.IState, input game_inputs_board.RankInput) (any, error) {
	var state = abstract.UseState[states.IStateL1](s)
	var tb = abstract.UseToolbox[toolbox.IToolboxL1](a.Layer.Tools())
	trx := state.Trx()

	level := input.Level
	key := input.GameKey + "_" + level
	shards, ok := lbShards[key]
	if ok {
		p := float64(0)
		trx.Db().Model(&model.User{}).Select(adapters.BuildJsonFetcher("metadata", input.GameKey + "." + shards.Param)).Where("id = ?", state.Info().UserId()).First(&p)
		trx.ClearError()
		for i := len(shards.Shards) - 1; i >= 0; i-- {
			if shards.Shards[i].Start <= p {
				level = shards.Shards[i].Level
				break
			}
		}
	}

	sFormula := game_model.StoredFormula{}
	err0 := trx.Db().Model(&game_model.StoredFormula{}).
		Where("game_key = ?", input.GameKey).
		Where("level = ?", level).
		First(&sFormula).
		Error
	if err0 != nil {
		return nil, err0
	}
	trx.ClearError()
	forStr, errJson := json.Marshal(sFormula.Data)
	if errJson != nil {
		return nil, errJson
	}
	forMem := game_model.MemFormula{}
	errJson2 := json.Unmarshal(forStr, &forMem)
	if errJson2 != nil {
		return nil, errJson2
	}
	var asc = (forMem.Order == "asc")

	var rank = game_memory.HumanRank(tb.Cache(), state.Info().UserId(), input.GameKey, level, asc)
	var levelDataStr string
	err := trx.Db().Model(&model.User{}).Select(adapters.BuildJsonFetcher("metadata", input.GameKey + ".board." + level)).Where("id = ?", state.Info().UserId()).First(&levelDataStr).Error
	if err != nil {
		return game_outputs_board.RankOutput{}, nil
	}
	result := map[string]any{}
	err2 := json.Unmarshal([]byte(levelDataStr), &result)
	if err2 != nil {
		log.Println(err2)
		return game_outputs_board.RankOutput{}, nil
	}
	delete(result, "old")
	return game_outputs_board.RankOutput{Rank: rank + 1, Score: result}, nil
}

// NextEnd /board/nextEnd check [ true false false ] access [ true false false false POST ]
func (a *Actions) NextEnd(s abstract.IState, input game_inputs_board.RankInput) (any, error) {
	diff := 5 - int(time.Now().Weekday())
	if diff < 0 {
		diff += 7
	}
	rawEnd := time.Now().Add(time.Duration(diff+1) * 24 * time.Hour)
	t := time.Date(rawEnd.Year(), rawEnd.Month(), rawEnd.Day(), 0, 0, 0, 0, rawEnd.Location())
	return map[string]any{"nextEndMillis": t.UnixMilli()}, nil
}
