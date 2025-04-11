package actions_report

import (
	"kasper/cmd/babble/sigma/abstract"
	"kasper/cmd/babble/sigma/layer1/adapters"
	"kasper/cmd/babble/sigma/layer1/module/state"
	inputs_report "kasper/cmd/babble/sigma/plugins/social/inputs/report"
	"kasper/cmd/babble/sigma/plugins/social/model"
	outputs_report "kasper/cmd/babble/sigma/plugins/social/outputs/report"
	"kasper/cmd/babble/sigma/utils/crypto"
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
