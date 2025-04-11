package admin_outputs_player

import admin_model "kasper/src/plugins/admin/model"

type ListOutput struct {
	Players    []admin_model.PlayerMini `json:"players"`
	TotalCount int64                    `json:"totalCount"`
}
