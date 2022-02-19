package subscription

import (
	"context"
	"fmt"
	"time"

	jsoniter "github.com/json-iterator/go"
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
	"go.mongodb.org/mongo-driver/mongo/options"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

type Resolver struct {
	types.Resolver
}

func New(r types.Resolver) generated.SubscriptionResolver {
	return &Resolver{
		Resolver: r,
	}
}

func (r *Resolver) Me(ctx context.Context) (<-chan *model.User, error) {
	user := auth.For(ctx)
	if user == nil {
		return nil, nil
	}

	usr, err := loaders.For(ctx).UserLoader.Load(user.ID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}

		logrus.Error("failed to get stream: ", err)
		return nil, helpers.ErrInternalServerError
	}

	ch := make(chan *model.User, 1)
	ch <- modelstructures.User(usr).ToModel(auth.For(ctx))

	ctx, cancel := context.WithCancel(ctx)

	subCh := make(chan string, 1)
	r.Ctx.Inst().Redis.Subscribe(ctx, subCh, fmt.Sprintf("gql-subs:users:%s", user.ID.Hex()))

	go func() {
		<-ctx.Done()

		close(ch)
		close(subCh)
	}()
	go func() {
		defer cancel()

		for range subCh {
			usr, err := loaders.For(ctx).UserLoader.Load(user.ID)
			if err != nil {
				if err == mongo.ErrNoDocuments {
					return
				}

				logrus.Error("failed to get stream: ", err)
				return
			}

			ch <- modelstructures.User(usr).ToModel(&usr)
		}
	}()

	return ch, nil
}

func (r *Resolver) User(ctx context.Context, id primitive.ObjectID) (<-chan *model.User, error) {
	authed := auth.For(ctx)
	ids := []primitive.ObjectID{id}
	if authed != nil {
		ids = append(ids, authed.ID)
	}

	usrs, errs := loaders.For(ctx).UserLoader.LoadAll(ids)
	if errs[0] != nil {
		if errs[0] == mongo.ErrNoDocuments {
			return nil, nil
		}

		logrus.Error("failed to get stream: ", errs[0])
		return nil, helpers.ErrInternalServerError
	}

	var me *structures.User
	if len(usrs) == 2 {
		me = &usrs[1]
	}

	ch := make(chan *model.User, 1)
	ch <- modelstructures.User(usrs[0]).ToModel(me)

	subCh := make(chan string, 1)
	r.Ctx.Inst().Redis.Subscribe(ctx, subCh, fmt.Sprintf("gql-subs:users:%s", id.Hex()))

	ctx, cancel := context.WithCancel(ctx)
	go func() {
		<-ctx.Done()

		close(ch)
		close(subCh)
	}()

	go func() {
		defer func() {
			cancel()
			if err := recover(); err != nil {
				logrus.Error("panic recovered: ", err)
			}
		}()

		for range subCh {
			usrs, errs := loaders.For(ctx).UserLoader.LoadAll(ids)
			if errs[0] != nil {
				if errs[0] == mongo.ErrNoDocuments {
					return
				}

				logrus.Error("failed to get stream: ", errs[0])
				return
			}

			var me *structures.User
			if len(usrs) == 2 {
				me = &usrs[1]
			}

			select {
			case <-ctx.Done():
				return
			default:
			}

			ch <- modelstructures.User(usrs[0]).ToModel(me)
		}
	}()

	return ch, nil
}

func (r *Resolver) Messages(ctx context.Context, channelID primitive.ObjectID) (<-chan *model.ChatMessage, error) {
	me := auth.For(ctx)
	channel, err := loaders.For(ctx).UserLoader.Load(channelID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}

		logrus.Error("failed to get stream: ", err)
		return nil, helpers.ErrInternalServerError
	}

	if !channel.Channel.Public && (me == nil || (channel.Role < structures.GlobalRoleStaff && me.MemberRole(channel.ID) < structures.ChannelRoleViewer)) {
		return nil, helpers.ErrAccessDenied
	}

	ch := make(chan *model.ChatMessage, 1)
	ch <- &model.ChatMessage{
		ID:        primitive.NewObjectIDFromTimestamp(time.Now()),
		Content:   "Welcome to the chat room",
		UserID:    primitive.NilObjectID,
		ChannelID: channelID,
	}

	subCh := make(chan string, 1)
	r.Ctx.Inst().Redis.Subscribe(ctx, subCh, fmt.Sprintf("gql-subs:chat:%s", channelID.Hex()))
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		<-ctx.Done()

		close(ch)
		close(subCh)
	}()

	if me != nil {
		go func() {
			tick := time.NewTicker(time.Second * 5)
			defer tick.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-tick.C:
				}
				if _, err := r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameCountDocuments).UpdateOne(ctx, bson.M{
					"key":   me.ID,
					"group": channelID,
					"type":  structures.CountDocumentTypeChatter,
				}, bson.M{
					"$set": bson.M{
						"expiry": time.Now().Add(time.Second * 15),
					},
					"$setOnInsert": bson.M{
						"key":   me.ID,
						"group": channelID,
						"type":  structures.CountDocumentTypeChatter,
					},
				}, options.Update().SetUpsert(true)); err != nil {
					logrus.Error("could not upsert: ", err)
				}
			}
		}()
	}

	go func() {
		defer func() {
			cancel()
			if err := recover(); err != nil {
				logrus.Error("panic recovered: ", err)
			}
		}()

		for msg := range subCh {
			channel, err := loaders.For(ctx).UserLoader.Load(channelID)
			if err != nil {
				if err == mongo.ErrNoDocuments {
					return
				}

				logrus.Error("failed to get stream: ", err)
				return
			}

			me := auth.For(ctx)
			if !channel.Channel.Public && (me == nil || (me.Role < structures.GlobalRoleStaff && me.MemberRole(channel.ID) < structures.ChannelRoleViewer)) {
				return
			}

			dbMsg := structures.Message{}

			if err := json.UnmarshalFromString(msg, &dbMsg); err != nil {
				logrus.Error("failed to decode msg: ", err)
				continue
			}

			select {
			case <-ctx.Done():
				return
			default:
			}

			ch <- modelstructures.Message(dbMsg).ToModel()
		}
	}()

	return ch, nil
}
