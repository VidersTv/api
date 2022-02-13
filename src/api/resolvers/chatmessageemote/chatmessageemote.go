package chatmessageemote

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/viderstv/api/graph/generated"
	"github.com/viderstv/api/graph/model"
	"github.com/viderstv/api/src/api/auth"
	"github.com/viderstv/api/src/api/helpers"
	"github.com/viderstv/api/src/api/loaders"
	"github.com/viderstv/api/src/api/types"
	"github.com/viderstv/api/src/modelstructures"
	"github.com/viderstv/common/structures"
	"github.com/viderstv/common/svc/mongo"
)

type Resolver struct {
	types.Resolver
}

func New(r types.Resolver) generated.ChatMessageEmoteResolver {
	return &Resolver{
		Resolver: r,
	}
}

func (r *Resolver) Emote(ctx context.Context, obj *model.ChatMessageEmote) (*model.UserChannelEmote, error) {
	user, err := loaders.For(ctx).UserLoader.Load(obj.ChannelID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}

		logrus.Error("failed to get stream: ", err)
		return nil, helpers.ErrInternalServerError
	}

	me := auth.For(ctx)

	if !user.Channel.Public && (me == nil || (me.Role < structures.GlobalRoleStaff && me.MemberRole(user.ID) < structures.ChannelRoleViewer)) {
		return nil, helpers.ErrAccessDenied
	}

	for _, v := range user.Channel.Emotes {
		if v.ID == obj.ID {
			return modelstructures.Emote(v).ToModel(), nil
		}
	}

	return nil, nil
}
