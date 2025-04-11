package admin_outputs_board

import admin_model "kasper/cmd/babble/sigma/plugins/admin/model"

type ReadFormulasOutput struct {
	Formulas []admin_model.Formula `json:"formulas"`
}
