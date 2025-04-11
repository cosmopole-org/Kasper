package outputs_spaces

import (
	models "kasper/src/shell/api/model"
)

type MemberUser struct {
	Member models.Member     `json:"member"`
	User   models.PublicUser `json:"user"`
}

type ReadMemberOutput struct {
	Members []MemberUser `json:"members"`
}
