package modelstructures

import (
	"github.com/viderstv/api/graph/model"
	"github.com/viderstv/common/structures"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User structures.User

func (u User) ToModel(authed *structures.User) *model.User {
	memberships := make([]*model.UserMembership, len(u.Memberships))
	for i, v := range u.Memberships {
		memberships[i] = Member(v).ToModel()
	}

	channel := Channel(u.Channel).ToModel()
	channel.ID = u.ID

	if authed == nil || (authed.Role < structures.GlobalRoleStaff && authed.MemberRole(u.ID) < structures.ChannelRoleAdmin) {
		channel.StreamKey = nil
		memberships = nil
	}
	var pfp *primitive.ObjectID
	if !u.ProfilePicture.IsZero() {
		pfp = &u.ProfilePicture
	}

	return &model.User{
		ID:             u.ID,
		Login:          u.Login,
		DisplayName:    u.DisplayName,
		ProfilePicture: pfp,
		Color:          Color(u.Color).ToModel(),
		Role:           GlobalRole(u.Role).ToModel(),
		Channel:        channel,
		TwitchAccount:  TwitchAccount(u.TwitchAccount).ToModel(),
		Memberships:    memberships,
	}
}
