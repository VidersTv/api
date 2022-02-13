package stream

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
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Resolver struct {
	types.Resolver
}

func New(r types.Resolver) generated.StreamResolver {
	return &Resolver{
		Resolver: r,
	}
}

func (r *Resolver) User(ctx context.Context, obj *model.Stream) (*model.User, error) {
	user, err := loaders.For(ctx).UserLoader.Load(obj.UserID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}

		logrus.Error("failed to get user: ", err)
		return nil, helpers.ErrInternalServerError
	}

	return modelstructures.User(user).ToModel(auth.For(ctx)), nil
}

func (r *Resolver) AccessToken(ctx context.Context, obj *model.Stream) (*string, error) {
	if obj.User == nil {
		user, err := loaders.For(ctx).UserLoader.Load(obj.UserID)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				return nil, nil
			}

			logrus.Error("failed to get user: ", err)
			return nil, helpers.ErrInternalServerError
		}
		obj.User = modelstructures.User(user).ToModel(auth.For(ctx))
	}

	uid := primitive.NilObjectID
	if obj.User.Channel.Public {
		if user := auth.For(ctx); user != nil {
			uid = user.ID
		}

		goto authed
	} else if user := auth.For(ctx); user != nil {
		if user.Role >= structures.GlobalRoleStaff {
			uid = user.ID
			goto authed
		}
		for _, v := range user.Memberships {
			if v.ChannelID == obj.UserID {
				if v.Role >= structures.ChannelRoleViewer {
					uid = user.ID
					goto authed
				}
			}
		}
	}

	return nil, nil

authed:
	tkn, err := structures.EncodeJwt(structures.JwtWatchStream{
		ChannelID: obj.UserID,
		StreamID:  obj.ID,
		UserID:    uid,
	}, r.Ctx.Config().Auth.EdgeJwtToken)
	if err != nil {
		logrus.Error("failed to encode jwt: ", err)
		return nil, helpers.ErrInternalServerError
	}

	return &tkn, nil
}
