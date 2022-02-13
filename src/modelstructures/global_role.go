package modelstructures

import (
	"github.com/viderstv/api/graph/model"
	"github.com/viderstv/common/structures"
)

type GlobalRole structures.GlobalRole

func (g GlobalRole) ToModel() model.GlobalRole {
	switch structures.GlobalRole(g) {
	case structures.GlobalRoleOwner:
		return model.GlobalRoleOwner
	case structures.GlobalRoleStaff:
		return model.GlobalRoleStaff
	case structures.GlobalRoleStreamer:
		return model.GlobalRoleStreamer
	case structures.GlobalRoleUser:
		return model.GlobalRoleUser
	}

	return ""
}
