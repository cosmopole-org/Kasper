package outputs_report

import "kasper/src/plugins/social/model"

type ReportOutput struct {
	Report model.Report `json:"report"`
}
