package usermembership

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

func New(r types.Resolver) generated.UserMembershipResolver {
	return &Resolver{
		Resolver: r,
	}
}

func (r *Resolver) Channel(ctx context.Context, obj *model.UserMembership) (*model.User, error) {
	user, err := loaders.For(ctx).UserLoader.Load(obj.ChannelID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}

		logrus.Error("failed to get user: ", err)
		return nil, helpers.ErrInternalServerError
	}

	return modelstructures.User(user).ToModel(auth.For(ctx)), nil
}

func (r *Resolver) AddedBy(ctx context.Context, obj *model.UserMembership) (*model.User, error) {
	user, err := loaders.For(ctx).UserLoader.Load(obj.AddedByID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}

		logrus.Error("failed to get user: ", err)
		return nil, helpers.ErrInternalServerError
	}

	return modelstructures.User(user).ToModel(auth.For(ctx)), nil
}
