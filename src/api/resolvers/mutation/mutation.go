package mutation

import (
	"context"
	"fmt"
	"strings"
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
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

type Resolver struct {
	types.Resolver
}

func New(r types.Resolver) generated.MutationResolver {
	return &Resolver{
		Resolver: r,
	}
}

func (r *Resolver) SendMessage(ctx context.Context, channelID primitive.ObjectID, content string) (*model.ChatMessage, error) {
	if len(content) > 500 {
		return nil, helpers.ErrDontBeSilly
	}

	me := auth.For(ctx)
	if me == nil {
		return nil, helpers.ErrUnauthorized
	}

	user, err := loaders.For(ctx).UserLoader.Load(channelID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}

		logrus.Error("failed to get stream: ", err)
		return nil, helpers.ErrInternalServerError
	}

	if !user.Channel.Public && me.Role < structures.GlobalRoleStaff && me.MemberRole(user.ID) < structures.ChannelRoleViewer {
		return nil, helpers.ErrAccessDenied
	}

	pipe := r.Ctx.Inst().Redis.Pipeline()

	decr := func() {
		pipe := r.Ctx.Inst().Redis.Pipeline()
		pipe.Decr(ctx, fmt.Sprintf("chat-limits:%s:%s:1", channelID, me.ID))
		pipe.Decr(ctx, fmt.Sprintf("chat-limits:%s:%s:5", channelID, me.ID))
		_, _ = pipe.Exec(ctx)
	}

	chatLimits1SecondCmd := pipe.Incr(ctx, fmt.Sprintf("chat-limits:%s:%s:1", channelID, me.ID))
	chatLimits1SecondTtlCmd := pipe.TTL(ctx, fmt.Sprintf("chat-limits:%s:%s:1", channelID, me.ID))

	chatLimits5SecondCmd := pipe.Incr(ctx, fmt.Sprintf("chat-limits:%s:%s:5", channelID, me.ID))
	chatLimits5SecondTtlCmd := pipe.TTL(ctx, fmt.Sprintf("chat-limits:%s:%s:5", channelID, me.ID))

	bannedCmd := pipe.TTL(ctx, fmt.Sprintf("chat-bans:%s:%s", channelID, me.ID))
	_, err = pipe.Exec(ctx)
	if err != nil {
		logrus.Error("failed to get stream: ", err)
		return nil, helpers.ErrInternalServerError
	}

	if me.Role < structures.GlobalRoleStaff {
		switch bannedCmd.Val() {
		case -1:
			// permabanned
			decr()
			return nil, fmt.Errorf("%s: You are permanently banned", helpers.ErrAccessDenied.Error())
		case -2:
			// not banned
		default:
			// timedout
			decr()
			return nil, fmt.Errorf("%s: You are timedout try again in %s", helpers.ErrAccessDenied.Error(), bannedCmd.Val().String())
		}
	}

	if chatLimits1SecondTtlCmd.Val() == -1 || chatLimits5SecondTtlCmd.Val() == -1 {
		pipe := r.Ctx.Inst().Redis.Pipeline()
		if chatLimits1SecondTtlCmd.Val() == -1 {
			pipe.Expire(ctx, fmt.Sprintf("chat-limits:%s:%s:1", channelID, me.ID), time.Second)
			chatLimits1SecondTtlCmd.SetVal(time.Second)
		}
		if chatLimits5SecondTtlCmd.Val() == -1 {
			pipe.Expire(ctx, fmt.Sprintf("chat-limits:%s:%s:5", channelID, me.ID), time.Second*5)
			chatLimits5SecondTtlCmd.SetVal(time.Second * 5)
		}

		_, _ = pipe.Exec(ctx)
	}

	if me.Role < structures.GlobalRoleStaff && me.MemberRole(channelID) < structures.ChannelRoleVIP {
		if chatLimits1SecondCmd.Val() > 1 && chatLimits5SecondCmd.Val() > 3 {
			decr()
			lowest := chatLimits1SecondTtlCmd.Val()
			if lowest > chatLimits5SecondTtlCmd.Val() {
				lowest = chatLimits5SecondTtlCmd.Val()
			}
			return nil, fmt.Errorf("%s: You are sending messages too fast try again in %s", helpers.ErrAccessDenied.Error(), lowest)
		}
		// check std ratelimits
		if chatLimits1SecondCmd.Val() > 1 {
			decr()
			return nil, fmt.Errorf("%s: You are sending messages too fast try again in %s", helpers.ErrAccessDenied.Error(), chatLimits1SecondTtlCmd.Val())
		}
		if chatLimits5SecondCmd.Val() > 3 {
			decr()
			return nil, fmt.Errorf("%s: You are sending messages too fast try again in %s", helpers.ErrAccessDenied.Error(), chatLimits5SecondTtlCmd.Val())
		}
	} else {
		if chatLimits1SecondCmd.Val() > 5 && chatLimits5SecondCmd.Val() > 10 {
			decr()
			lowest := chatLimits1SecondTtlCmd.Val()
			if lowest > chatLimits5SecondTtlCmd.Val() {
				lowest = chatLimits5SecondTtlCmd.Val()
			}
			return nil, fmt.Errorf("%s: You are sending messages too fast try again in %s", helpers.ErrAccessDenied.Error(), lowest)
		}
		// check super ratelimits
		if chatLimits1SecondCmd.Val() > 5 {
			decr()
			return nil, fmt.Errorf("%s: You are sending messages too fast try again in %s", helpers.ErrAccessDenied.Error(), chatLimits1SecondTtlCmd.Val())
		}
		if chatLimits5SecondCmd.Val() > 10 {
			decr()
			return nil, fmt.Errorf("%s: You are sending messages too fast try again in %s", helpers.ErrAccessDenied.Error(), chatLimits5SecondTtlCmd.Val())
		}
	}

	mp := map[string]structures.Emote{}
	for _, v := range user.Channel.Emotes {
		mp[v.Tag] = v
	}

	emotes := []structures.MessageEmote{}
	splits := strings.Split(content, " ")
	for _, v := range splits {
		if emote, ok := mp[v]; ok {
			delete(mp, v)
			emotes = append(emotes, structures.MessageEmote{
				ID:  emote.ID,
				Tag: emote.Tag,
			})
		}
	}

	msg := structures.Message{
		ID:        primitive.NewObjectIDFromTimestamp(time.Now()),
		UserID:    me.ID,
		ChannelID: channelID,
		Content:   content,
		Emotes:    emotes,
	}

	if _, err = r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameMessages).InsertOne(ctx, msg); err != nil {
		logrus.Error("failed to insert chat message: ", err)
		return nil, helpers.ErrInternalServerError
	}

	msgText, _ := json.MarshalToString(msg)
	if err = r.Ctx.Inst().Redis.Publish(ctx, fmt.Sprintf("gql-subs:chat:%s", channelID.Hex()), msgText); err != nil {
		logrus.Error("failed to publish chat message: ", err)
		return nil, helpers.ErrInternalServerError
	}

	return modelstructures.Message(msg).ToModel(), nil
}
