package admin_outputs_board

import admin_model "kasper/cmd/babble/sigma/plugins/admin/model"

type GetFormulaOutput struct {
	Formula admin_model.Formula `json:"formula"`
}
