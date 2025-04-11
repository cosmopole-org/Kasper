package admin_outputs_board

import admin_model "kasper/src/plugins/admin/model"

type GetFormulaOutput struct {
	Formula admin_model.Formula `json:"formula"`
}
