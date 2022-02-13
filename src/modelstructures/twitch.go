package modelstructures

import (
	"github.com/viderstv/api/graph/model"
	"github.com/viderstv/common/structures"
)

type TwitchAccount structures.TwitchAccount

func (t TwitchAccount) ToModel() *model.UserTwitchAccount {
	return &model.UserTwitchAccount{
		ID:             t.ID,
		Login:          t.Login,
		DisplayName:    t.DisplayName,
		ProfilePicture: t.ProfilePicture,
	}
}
