package modelstructures

import (
	"github.com/viderstv/api/graph/model"
	"github.com/viderstv/common/structures"
)

type Channel structures.Channel

func (c Channel) ToModel() *model.UserChannel {
	emotes := make([]*model.UserChannelEmote, len(c.Emotes))
	for i, v := range c.Emotes {
		emotes[i] = Emote(v).ToModel()
	}
	return &model.UserChannel{
		Title:            c.Title,
		StreamKey:        &c.StreamKey,
		Public:           c.Public,
		LastLive:         &c.LastLive,
		TwitchRoleMirror: c.TwitchRoleMirror,
		Emotes:           emotes,
	}
}

type Member structures.Member

func (m Member) ToModel() *model.UserMembership {
	return &model.UserMembership{
		ChannelID: m.ChannelID,
		AddedByID: m.AddedByID,
		Role:      ChannelRole(m.Role).ToModel(),
	}
}

type ChannelRole structures.ChannelRole

func (c ChannelRole) ToModel() model.ChannelRole {
	switch structures.ChannelRole(c) {
	case structures.ChannelRoleAdmin:
		return model.ChannelRoleAdmin
	case structures.ChannelRoleModerator:
		return model.ChannelRoleModerator
	case structures.ChannelRoleEditor:
		return model.ChannelRoleEditor
	case structures.ChannelRoleVIP:
		return model.ChannelRoleVip
	case structures.ChannelRoleViewer:
		return model.ChannelRoleViewer
	case structures.ChannelRoleUser:
		return model.ChannelRoleUser
	}

	return ""
}

type Emote structures.Emote

func (e Emote) ToModel() *model.UserChannelEmote {
	return &model.UserChannelEmote{
		ID:         e.ID,
		Tag:        e.Tag,
		UploaderID: e.UploaderID,
	}
}
