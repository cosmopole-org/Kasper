package actions_report

import (
	"kasper/src/abstract"
	inputs_report "kasper/src/plugins/social/inputs/report"
	"kasper/src/plugins/social/model"
	outputs_report "kasper/src/plugins/social/outputs/report"
	"kasper/src/shell/layer1/adapters"
	module_state "kasper/src/shell/layer1/module/state"
	"kasper/src/shell/utils/crypto"
)

type Actions struct {
	Layer abstract.ILayer
}

func Install(s adapters.IStorage, a *Actions) error {
	return s.AutoMigrate(&model.Report{})
}

// Report /report/report check [ true false false ] access [ true false false false PUT ]
func (a *Actions) Report(s abstract.IState, input inputs_report.ReportInput) (any, error) {
	state := abstract.UseState[module_state.IStateL1](s)
	report := model.Report{GameKey: "hokm", Id: crypto.SecureUniqueId(a.Layer.Core().Id()), ReporterId: state.Info().UserId(), Data: input.Data}
	trx := state.Trx()
	err := trx.Db().Create(&report).Error
	if err != nil {
		return nil, err
	}
	return outputs_report.ReportOutput{Report: report}, nil
}
