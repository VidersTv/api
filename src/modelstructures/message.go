package modelstructures

import (
	"github.com/viderstv/api/graph/model"
	"github.com/viderstv/common/structures"
)

type Message structures.Message

func (m Message) ToModel() *model.ChatMessage {
	emotes := make([]*model.ChatMessageEmote, len(m.Emotes))
	for i, v := range m.Emotes {
		emotes[i] = MessageEmote(v).ToModel()
		emotes[i].ChannelID = m.ChannelID
	}

	return &model.ChatMessage{
		ID:        m.ID,
		UserID:    m.UserID,
		ChannelID: m.ChannelID,
		Content:   m.Content,
		Emotes:    emotes,
	}
}

type MessageEmote structures.MessageEmote

func (m MessageEmote) ToModel() *model.ChatMessageEmote {
	return &model.ChatMessageEmote{
		ID:  m.ID,
		Tag: m.Tag,
	}
}
