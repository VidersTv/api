package resolvers

import (
	"github.com/viderstv/api/graph/generated"
	"github.com/viderstv/api/src/api/resolvers/chatmessage"
	"github.com/viderstv/api/src/api/resolvers/chatmessageemote"
	"github.com/viderstv/api/src/api/resolvers/mutation"
	"github.com/viderstv/api/src/api/resolvers/query"
	"github.com/viderstv/api/src/api/resolvers/stream"
	"github.com/viderstv/api/src/api/resolvers/subscription"
	"github.com/viderstv/api/src/api/resolvers/userchannel"
	"github.com/viderstv/api/src/api/resolvers/userchannelemote"
	"github.com/viderstv/api/src/api/resolvers/usermembership"
	"github.com/viderstv/api/src/api/types"
)

type Resolver struct {
	types.Resolver
	query        generated.QueryResolver
	subscription generated.SubscriptionResolver
	mutation     generated.MutationResolver

	stream           generated.StreamResolver
	userchannel      generated.UserChannelResolver
	userchannelemote generated.UserChannelEmoteResolver
	usermembership   generated.UserMembershipResolver
	chatmessage      generated.ChatMessageResolver
	chatmessageemote generated.ChatMessageEmoteResolver
}

func New(r types.Resolver) generated.ResolverRoot {
	return &Resolver{
		Resolver:         r,
		query:            query.New(r),
		stream:           stream.New(r),
		userchannel:      userchannel.New(r),
		userchannelemote: userchannelemote.New(r),
		usermembership:   usermembership.New(r),
		subscription:     subscription.New(r),
		chatmessage:      chatmessage.New(r),
		chatmessageemote: chatmessageemote.New(r),
		mutation:         mutation.New(r),
	}
}

func (r *Resolver) Query() generated.QueryResolver {
	return r.query
}

func (r *Resolver) Subscription() generated.SubscriptionResolver {
	return r.subscription
}

func (r *Resolver) Mutation() generated.MutationResolver {
	return r.mutation
}

func (r *Resolver) Stream() generated.StreamResolver {
	return r.stream
}

func (r *Resolver) UserChannel() generated.UserChannelResolver {
	return r.userchannel
}

func (r *Resolver) UserChannelEmote() generated.UserChannelEmoteResolver {
	return r.userchannelemote
}

func (r *Resolver) UserMembership() generated.UserMembershipResolver {
	return r.usermembership
}

func (r *Resolver) ChatMessage() generated.ChatMessageResolver {
	return r.chatmessage
}

func (r *Resolver) ChatMessageEmote() generated.ChatMessageEmoteResolver {
	return r.chatmessageemote
}
