package admin_outputs_report

import admin_model "kasper/src/plugins/admin/model"

type ReadReportsOutput struct {
	Reports []admin_model.ResultReport `json:"reports"`
}
