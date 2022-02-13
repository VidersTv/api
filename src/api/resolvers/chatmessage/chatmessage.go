package chatmessage

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
	"github.com/viderstv/common/svc/mongo"
)

type Resolver struct {
	types.Resolver
}

func New(r types.Resolver) generated.ChatMessageResolver {
	return &Resolver{
		Resolver: r,
	}
}

func (r *Resolver) Channel(ctx context.Context, obj *model.ChatMessage) (*model.User, error) {
	user, err := loaders.For(ctx).UserLoader.Load(obj.ChannelID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}

		logrus.Error("failed to get stream: ", err)
		return nil, helpers.ErrInternalServerError
	}

	return modelstructures.User(user).ToModel(auth.For(ctx)), nil
}

func (r *Resolver) User(ctx context.Context, obj *model.ChatMessage) (*model.User, error) {
	user, err := loaders.For(ctx).UserLoader.Load(obj.UserID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}

		logrus.Error("failed to get stream: ", err)
		return nil, helpers.ErrInternalServerError
	}

	return modelstructures.User(user).ToModel(auth.For(ctx)), nil
}
