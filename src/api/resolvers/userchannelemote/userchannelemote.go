package userchannelemote

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

func New(r types.Resolver) generated.UserChannelEmoteResolver {
	return &Resolver{
		Resolver: r,
	}
}

func (r *Resolver) Uploader(ctx context.Context, obj *model.UserChannelEmote) (*model.User, error) {
	user, err := loaders.For(ctx).UserLoader.Load(obj.UploaderID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}

		logrus.Error("failed to get user: ", err)
		return nil, helpers.ErrInternalServerError
	}

	return modelstructures.User(user).ToModel(auth.For(ctx)), nil
}
