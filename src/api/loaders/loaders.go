package loaders

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	jsoniter "github.com/json-iterator/go"
	"github.com/sirupsen/logrus"
	"github.com/viderstv/api/graph/loaders"
	"github.com/viderstv/api/graph/model"
	"github.com/viderstv/api/src/global"
	"github.com/viderstv/api/src/modelstructures"
	"github.com/viderstv/common/structures"
	"github.com/viderstv/common/svc/mongo"
	"github.com/viderstv/common/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const LoadersKey = utils.Key("dataloaders")

var json = jsoniter.ConfigCompatibleWithStandardLibrary

type Loaders struct {
	UserLoader           *loaders.UserLoader
	UserByLoginLoader    *loaders.UserByLoginLoader
	StreamByUserIDLoader *loaders.StreamByUserIDLoader
}

func New(gCtx global.Context) *Loaders {
	return &Loaders{
		UserLoader: loaders.NewUserLoader(loaders.UserLoaderConfig{
			Fetch: func(keys []primitive.ObjectID) ([]structures.User, []error) {
				ctx, cancel := context.WithTimeout(gCtx, time.Second*10)
				defer cancel()
				cur, err := gCtx.Inst().Mongo.Collection(mongo.CollectionNameUsers).Find(ctx, bson.M{
					"_id": bson.M{
						"$in": keys,
					},
				})

				dbUsers := []structures.User{}
				if err == nil {
					err = cur.All(ctx, &dbUsers)
				}
				users := make([]structures.User, len(keys))
				errs := make([]error, len(keys))
				if err != nil {
					logrus.Error("failed to fetch users: ", err)
					for i := range errs {
						errs[i] = err
					}
					return users, errs
				}

				mp := map[primitive.ObjectID]structures.User{}
				for _, v := range dbUsers {
					mp[v.ID] = v
				}

				for i, v := range keys {
					if user, ok := mp[v]; ok {
						users[i] = user
					} else {
						errs[i] = mongo.ErrNoDocuments
					}
				}

				return users, errs
			},
			Wait: time.Millisecond * 10,
		}),
		UserByLoginLoader: loaders.NewUserByLoginLoader(loaders.UserByLoginLoaderConfig{
			Fetch: func(keys []string) ([]structures.User, []error) {
				ctx, cancel := context.WithTimeout(gCtx, time.Second*10)
				defer cancel()
				cur, err := gCtx.Inst().Mongo.Collection(mongo.CollectionNameUsers).Find(ctx, bson.M{
					"login": bson.M{
						"$in": keys,
					},
				})
				dbUsers := []structures.User{}
				if err == nil {
					err = cur.All(ctx, &dbUsers)
				}
				users := make([]structures.User, len(keys))
				errs := make([]error, len(keys))
				if err != nil {
					logrus.Error("failed to fetch users: ", err)
					for i := range errs {
						errs[i] = err
					}
					return users, errs
				}

				mp := map[string]structures.User{}
				for _, v := range dbUsers {
					mp[v.Login] = v
				}

				for i, v := range keys {
					if user, ok := mp[v]; ok {
						users[i] = user
					} else {
						errs[i] = mongo.ErrNoDocuments
					}
				}

				return users, errs
			},
			Wait: time.Millisecond * 10,
		}),
		StreamByUserIDLoader: loaders.NewStreamByUserIDLoader(loaders.StreamByUserIDLoaderConfig{
			Fetch: func(keys []primitive.ObjectID) ([]*model.Stream, []error) {
				ctx, cancel := context.WithTimeout(gCtx, time.Second*10)
				defer cancel()
				cur, err := gCtx.Inst().Mongo.Collection(mongo.CollectionNameStreams).Find(ctx, bson.M{
					"user_id": bson.M{
						"$in": keys,
					},
					"ended_at": time.Time{},
				})
				dbStreams := []structures.Stream{}
				if err == nil {
					err = cur.All(ctx, &dbStreams)
				}
				streams := make([]*model.Stream, len(keys))
				errs := make([]error, len(keys))
				if err != nil {
					logrus.Error("failed to fetch users: ", err)
					for i := range errs {
						errs[i] = err
					}
					return streams, errs
				}

				mp := map[primitive.ObjectID]structures.Stream{}
				mpRedisCmds := map[primitive.ObjectID]*redis.StringCmd{}
				mpVariants := map[primitive.ObjectID][]*model.StreamVariant{}
				if len(dbStreams) != 0 {
					pipe := gCtx.Inst().Redis.Pipeline()
					for _, v := range dbStreams {
						mp[v.UserID] = v
						mpRedisCmds[v.UserID] = pipe.Get(ctx, fmt.Sprintf("stream:%s:variants", v.ID.Hex()))
					}
					_, _ = pipe.Exec(ctx)
					for k, v := range mp {
						res, err := mpRedisCmds[k].Result()
						if err != nil {
							delete(mp, k)
							logrus.Error("failed to get stream data from redis: ", err)
						}

						variants := []structures.JwtMuxerPayloadVariant{}
						if err := json.UnmarshalFromString(res, &variants); err != nil {
							delete(mp, k)
							logrus.Error("failed to get stream data from redis: ", err)
						}

						mVariants := make([]*model.StreamVariant, len(variants))
						for i, v := range variants {
							mVariants[i] = &model.StreamVariant{
								Name:    v.Name,
								Fps:     v.FPS,
								Bitrate: v.Bitrate,
								Width:   v.Width,
								Height:  v.Height,
							}
						}

						mpVariants[v.UserID] = mVariants
					}
				}

				n := 0

				for i, v := range keys {
					if stream, ok := mp[v]; ok {
						str := modelstructures.Stream(stream).ToModel()
						str.Variants = mpVariants[v]
						streams[i] = str
						n++
					} else {
						errs[i] = mongo.ErrNoDocuments
					}
				}

				return streams, errs
			},
			Wait: time.Millisecond * 10,
		}),
	}
}

func For(ctx context.Context) *Loaders {
	return ctx.Value(LoadersKey).(*Loaders)
}
