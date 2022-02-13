package auth

import (
	"context"

	"github.com/viderstv/api/src/api/helpers"
	"github.com/viderstv/api/src/api/loaders"
	"github.com/viderstv/common/structures"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func For(ctx context.Context) *structures.User {
	raw, _ := ctx.Value(helpers.UserKey).(*primitive.ObjectID)
	if raw != nil {
		usr, err := loaders.For(ctx).UserLoader.Load(*raw)
		if err == nil {
			return &usr
		}

		return nil
	}

	return nil
}
