package query

import (
	"context"
	"sort"

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
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Resolver struct {
	types.Resolver
}

func New(r types.Resolver) generated.QueryResolver {
	return &Resolver{
		Resolver: r,
	}
}

func (r *Resolver) User(ctx context.Context, uID primitive.ObjectID) (*model.User, error) {
	user, err := loaders.For(ctx).UserLoader.Load(uID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}

		logrus.Error("failed to get user: ", err)
		return nil, helpers.ErrInternalServerError
	}

	return modelstructures.User(user).ToModel(auth.For(ctx)), nil
}

func (r *Resolver) Me(ctx context.Context) (*model.User, error) {
	user := auth.For(ctx)
	if user != nil {
		return modelstructures.User(*user).ToModel(user), nil
	}

	return nil, nil
}

func (r *Resolver) UserByLogin(ctx context.Context, login string) (*model.User, error) {
	user, err := loaders.For(ctx).UserByLoginLoader.Load(login)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}

		logrus.Error("failed to get user: ", err)
		return nil, helpers.ErrInternalServerError
	}

	return modelstructures.User(user).ToModel(auth.For(ctx)), nil
}

func (r *Resolver) LiveChannels(ctx context.Context, page int, limit int) ([]*model.User, error) {
	return nil, nil
}

func (r *Resolver) Chatters(ctx context.Context, channelID primitive.ObjectID, page int, limit int) ([]*model.User, error) {
	me := auth.For(ctx)

	channel, err := loaders.For(ctx).UserLoader.Load(channelID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}

		logrus.Error("failed to query users: ", err)
		return nil, helpers.ErrInternalServerError
	}

	if !channel.Channel.Public && (me == nil || (me.Role < structures.GlobalRoleStaff && me.MemberRole(channel.ID) < structures.ChannelRoleViewer)) {
		return nil, helpers.ErrAccessDenied
	}

	cur, err := r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameCountDocuments).Find(ctx, bson.M{
		"group": channelID,
		"type":  structures.CountDocumentTypeChatter,
	})
	if err != nil {
		logrus.Error("failed to query users: ", err)
		return nil, helpers.ErrInternalServerError
	}

	countDocs := []structures.CountDocument{}
	if err := cur.All(ctx, &countDocs); err != nil {
		logrus.Error("failed to query users: ", err)
		return nil, helpers.ErrInternalServerError
	}

	ids := make([]primitive.ObjectID, len(countDocs))
	for i, v := range countDocs {
		ids[i] = v.Key.(primitive.ObjectID)
	}

	users, errs := loaders.For(ctx).UserLoader.LoadAll(ids)
	fileredUsers := []structures.User{}
	for i, v := range users {
		if errs[i] == nil {
			fileredUsers = append(fileredUsers, v)
		}
	}

	sort.Slice(fileredUsers, func(i, j int) bool {
		return fileredUsers[i].Role > fileredUsers[j].Role || fileredUsers[i].MemberRole(channelID) > fileredUsers[j].MemberRole(channelID)
	})

	if page*limit > len(fileredUsers) {
		return nil, nil
	}

	fileredUsers = fileredUsers[page*limit:]
	models := make([]*model.User, len(fileredUsers))
	for i, v := range fileredUsers {
		models[i] = modelstructures.User(v).ToModel(me)
	}

	return models, nil
}

func (r *Resolver) ViewerCount(ctx context.Context, channelID primitive.ObjectID) (*int, error) {
	me := auth.For(ctx)

	channel, err := loaders.For(ctx).UserLoader.Load(channelID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}

		logrus.Error("failed to query users: ", err)
		return nil, helpers.ErrInternalServerError
	}

	if !channel.Channel.Public && (me == nil || (me.Role < structures.GlobalRoleStaff && me.MemberRole(channel.ID) < structures.ChannelRoleViewer)) {
		return nil, helpers.ErrAccessDenied
	}

	count, err := r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameCountDocuments).CountDocuments(ctx, bson.M{
		"group": channelID,
		"type":  structures.CountDocumentTypeViewer,
	})
	if err != nil {
		logrus.Error("failed to query users: ", err)
		return nil, helpers.ErrInternalServerError
	}

	i := int(count)
	return &i, nil
}
