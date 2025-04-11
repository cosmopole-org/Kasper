package actions_board

import (
	"encoding/json"
	"errors"
	"kasper/src/abstract"
	admin_inputs_board "kasper/src/plugins/admin/inputs/board"
	admin_model "kasper/src/plugins/admin/model"
	admin_outputs_board "kasper/src/plugins/admin/outputs/board"
	game_memory "kasper/src/plugins/game/tools/memory"
	"kasper/src/shell/api/model"
	"kasper/src/shell/layer1/adapters"
	states "kasper/src/shell/layer1/module/state"
	tb "kasper/src/shell/layer1/module/toolbox"
	"kasper/src/shell/utils/crypto"
	"log"

	"github.com/fatih/structs"
	"gorm.io/gorm"
)

type Actions struct {
	Layer abstract.ILayer
}

func Install(s adapters.IStorage, a *Actions) error {
	return s.AutoMigrate(&admin_model.StoredFormula{})
}

// Kickout /admin/board/kickout check [ true false false ] access [ true false false false POST ]
func (a *Actions) Kickout(s abstract.IState, input admin_inputs_board.KickoutInput) (any, error) {
	var toolbox = abstract.UseToolbox[tb.IToolboxL1](a.Layer.Tools())
	var state = abstract.UseState[states.IStateL1](s)
	if !state.Info().IsGod() {
		return nil, errors.New("access denied")
	}
	trx := state.Trx()
	err := adapters.UpdateJson(func() *gorm.DB { return trx.Db().Model(&model.User{}).Where("id = ?", input.UserId) }, &model.User{Id: input.UserId}, "metadata", input.GameKey+".board."+input.Level, map[string]any{})
	if err != nil {
		log.Println(err)
		return nil, err
	}
	game_memory.Kickout(toolbox.Cache(), input.GameKey, input.Level, input.UserId)
	return admin_outputs_board.KickoutOutput{}, nil
}

// SetFormula /admin/board/setFormula check [ true false false ] access [ true false false false POST ]
func (a *Actions) SetFormula(s abstract.IState, input admin_inputs_board.SetFormulaInput) (any, error) {
	var state = abstract.UseState[states.IStateL1](s)
	if !state.Info().IsGod() {
		return nil, errors.New("access denied")
	}
	trx := state.Trx()
	keysLength := len(input.Keys)
	if len(input.NonZero) != keysLength || len(input.Operations) != keysLength || len(input.Weights) != keysLength {
		return nil, errors.New("length not equal")
	}
	sFormula := admin_model.StoredFormula{}
	err := trx.Db().Model(&admin_model.StoredFormula{}).
		Where("game_key = ?", input.GameKey).
		Where("level = ?", input.Level).
		First(&sFormula).
		Error
	if err != nil {
		trx.ClearError()
		sFormula.Id = crypto.SecureUniqueId(a.Layer.Core().Id())
		sFormula.GameKey = input.GameKey
		sFormula.Level = input.Level
		trx.Db().Create(&sFormula)
	}
	trx.ClearError()
	err2 := adapters.UpdateJson(
		func() *gorm.DB {
			return trx.Db().Model(&admin_model.StoredFormula{}).
				Where("game_key = ?", input.GameKey).
				Where("level = ?", input.Level)
		},
		&sFormula,
		"data",
		"",
		structs.Map(admin_model.MemFormula{
			Keys:       input.Keys,
			Weights:    input.Weights,
			NonZero:    input.NonZero,
			Operations: input.Operations,
			TotalOp:    input.TotalOp,
			Rules:      input.Rules,
			Order:      input.Order,
		}),
	)
	if err2 != nil {
		log.Println(err2)
		return nil, err2
	}
	return admin_outputs_board.SetFormulaOutput{}, nil
}

// GetFormula /admin/board/getFormula check [ true false false ] access [ true false false false POST ]
func (a *Actions) GetFormula(s abstract.IState, input admin_inputs_board.GetFormulaInput) (any, error) {
	var state = abstract.UseState[states.IStateL1](s)
	if !state.Info().IsGod() {
		return nil, errors.New("access denied")
	}
	trx := state.Trx()
	sFormula := admin_model.StoredFormula{}
	err := trx.Db().Model(&admin_model.StoredFormula{}).
		Where("game_key = ?", input.GameKey).
		Where("level = ?", input.Level).
		First(&sFormula).
		Error
	if err != nil {
		log.Println(err)
		return nil, err
	}
	forStr, errJson := json.Marshal(sFormula.Data)
	if errJson != nil {
		return nil, errJson
	}
	forMem := admin_model.MemFormula{}
	errJson2 := json.Unmarshal(forStr, &forMem)
	if errJson2 != nil {
		return nil, errJson2
	}
	formula := admin_model.Formula{GameKey: sFormula.GameKey, Level: sFormula.Level, Data: forMem}
	return admin_outputs_board.GetFormulaOutput{Formula: formula}, nil
}

// ReadFormulas /admin/board/readFormulas check [ true false false ] access [ true false false false POST ]
func (a *Actions) ReadFormulas(s abstract.IState, input admin_inputs_board.GetFormulaInput) (any, error) {
	var state = abstract.UseState[states.IStateL1](s)
	if !state.Info().IsGod() {
		return nil, errors.New("access denied")
	}
	trx := state.Trx()
	formulas := []admin_model.Formula{}
	err := trx.Db().Model(&admin_model.Formula{}).
		Find(&formulas).
		Error
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return admin_outputs_board.ReadFormulasOutput{Formulas: formulas}, nil
}
