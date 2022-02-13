package userchannel

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/viderstv/api/graph/generated"
	"github.com/viderstv/api/graph/model"
	"github.com/viderstv/api/src/api/helpers"
	"github.com/viderstv/api/src/api/loaders"
	"github.com/viderstv/api/src/api/types"
	"github.com/viderstv/common/svc/mongo"
)

type Resolver struct {
	types.Resolver
}

func New(r types.Resolver) generated.UserChannelResolver {
	return &Resolver{
		Resolver: r,
	}
}

func (r *Resolver) CurrentStream(ctx context.Context, obj *model.UserChannel) (*model.Stream, error) {
	stream, err := loaders.For(ctx).StreamByUserIDLoader.Load(obj.ID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}

		logrus.Error("failed to get stream: ", err)
		return nil, helpers.ErrInternalServerError
	}

	return stream, nil
}
