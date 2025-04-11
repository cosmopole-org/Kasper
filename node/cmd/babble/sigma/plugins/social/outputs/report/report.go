package outputs_report

import "kasper/cmd/babble/sigma/plugins/social/model"

type ReportOutput struct {
	Report model.Report `json:"report"`
}
