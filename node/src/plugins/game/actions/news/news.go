package actions_news

import (
	"kasper/src/abstract"
	game_inputs_news "kasper/src/plugins/game/inputs/news"
	game_model "kasper/src/plugins/game/model"
	"kasper/src/shell/layer1/adapters"
	states "kasper/src/shell/layer1/module/state"
	"kasper/src/shell/utils/crypto"
	"time"
)

type Actions struct {
	Layer abstract.ILayer
}

func Install(s adapters.IStorage, a *Actions) error {
	s.AutoMigrate(&game_model.NewsSeen{})
	return s.AutoMigrate(&game_model.News{})
}

// Create /news/create check [ true false false ] access [ true false false false GET ]
func (a *Actions) Create(s abstract.IState, input game_inputs_news.CreateInput) (any, error) {
	var state = abstract.UseState[states.IStateL1](s)
	trx := state.Trx()

	news := game_model.News{Id: crypto.SecureUniqueId(a.Layer.Core().Id()), Data: input.Data, GameKey: input.GameKey, Time: time.Now().UnixMilli()}
	trx.Db().Create(&news)
	return map[string]any{"news": news}, nil
}

// Delete /news/delete check [ true false false ] access [ true false false false GET ]
func (a *Actions) Delete(s abstract.IState, input game_inputs_news.DeleteInput) (any, error) {
	var state = abstract.UseState[states.IStateL1](s)
	trx := state.Trx()

	news := game_model.News{Id: input.NewsId}
	trx.Db().Delete(&news)
	return map[string]any{}, nil
}

// Read /news/read check [ true false false ] access [ true false false false GET ]
func (a *Actions) Read(s abstract.IState, input game_inputs_news.ReadInput) (any, error) {
	var state = abstract.UseState[states.IStateL1](s)
	trx := state.Trx()

	newsList := []game_model.News{}
	trx.Db().Where("game_key = ?", input.GameKey).Order("time asc").Find(&newsList)
	return map[string]any{"newsList": newsList}, nil
}

// Last /news/last check [ true false false ] access [ true false false false GET ]
func (a *Actions) Last(s abstract.IState, input game_inputs_news.LastInput) (any, error) {
	var state = abstract.UseState[states.IStateL1](s)
	trx := state.Trx()

	news := game_model.News{}
	trx.Db().Where("game_key = ?", input.GameKey).Order("time desc").Last(&news)
	trx.ClearError()
	newsSeen := game_model.NewsSeen{Id: state.Info().UserId() + "|" + news.Id}
	trx.Db().First(&newsSeen)
	trx.ClearError()
	seen := true
	if newsSeen.Payload == "" {
		seen = false
	}
	return map[string]any{"news": news, "seen": seen}, nil
}

// See /news/see check [ true false false ] access [ true false false false GET ]
func (a *Actions) See(s abstract.IState, input game_inputs_news.SeeInput) (any, error) {
	var state = abstract.UseState[states.IStateL1](s)
	trx := state.Trx()

	newsSeen := game_model.NewsSeen{Id: state.Info().UserId() + "|" + input.NewsId, Payload: "dummy"}
	trx.Db().Create(&newsSeen)
	return map[string]any{"newsSeen": newsSeen}, nil
}
