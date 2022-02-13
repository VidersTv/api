package api

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/executor"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/fasthttp/websocket"
	"github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
	"github.com/viderstv/api/graph/generated"
	"github.com/viderstv/api/src/api/cache"
	"github.com/viderstv/api/src/api/complexity"
	"github.com/viderstv/api/src/api/helpers"
	"github.com/viderstv/api/src/api/loaders"
	"github.com/viderstv/api/src/api/middleware"
	"github.com/viderstv/api/src/api/resolvers"
	"github.com/viderstv/api/src/api/types"
	wsTransport "github.com/viderstv/api/src/api/websocket"
	"github.com/viderstv/api/src/global"
	"github.com/viderstv/common/structures"
	"github.com/viderstv/common/utils"
)

func GqlHandler(gCtx global.Context, loader *loaders.Loaders) func(ctx *fasthttp.RequestCtx) {
	schema := generated.NewExecutableSchema(generated.Config{
		Resolvers:  resolvers.New(types.Resolver{Ctx: gCtx}),
		Directives: middleware.New(gCtx),
		Complexity: complexity.New(gCtx),
	})
	srv := handler.New(schema)

	exec := executor.New(schema)

	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})
	srv.Use(extension.Introspection{})

	srv.Use(&extension.ComplexityLimit{
		Func: func(ctx context.Context, rc *graphql.OperationContext) int {
			return 75
		},
	})

	srv.Use(extension.Introspection{})
	srv.Use(extension.AutomaticPersistedQuery{
		Cache: cache.NewRedisCache(gCtx, "", time.Hour*6),
	})

	srv.SetRecoverFunc(func(ctx context.Context, err interface{}) (userMessage error) {
		logrus.Error("panic in handler: ", err)
		return helpers.ErrInternalServerError
	})

	wsTransport := wsTransport.Websocket{
		KeepAlivePingInterval: 10 * time.Second,
		InitFunc: func(ctx context.Context, initPayload wsTransport.InitPayload) (context.Context, error) {
			auth := initPayload.Authorization()

			if strings.HasPrefix(auth, "Bearer ") {
				auth := strings.TrimPrefix(auth, "Bearer ")
				login := structures.JwtLogin{}
				if err := structures.DecodeJwt(&login, gCtx.Config().Auth.JwtToken, auth); err != nil {
					logrus.Error("err: ", err)
					goto handler
				}

				if time.Unix(login.ExpiresAt, 0).Before(time.Now()) {
					goto handler
				}

				ctx = context.WithValue(ctx, helpers.UserKey, &login.UserID)
			}

		handler:
			return ctx, nil
		},
		Upgrader: websocket.FastHTTPUpgrader{
			CheckOrigin: func(ctx *fasthttp.RequestCtx) bool {
				return true
			},
		},
	}

	return func(ctx *fasthttp.RequestCtx) {
		origin := utils.B2S(ctx.Request.Header.Peek("origin"))
		if origin == "" {
			origin = "*"
		}

		ctx.Response.Header.Set("Access-Control-Allow-Origin", origin)
		ctx.Response.Header.Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		ctx.Response.Header.Set("Access-Control-Allow-Headers", "Content-Type")
		ctx.Response.Header.Set("Access-Control-Max-Age", "86400")
		ctx.Response.Header.Set("Access-Control-Allow-Credentials", "true")
		ctx.Response.Header.Set("Vary", "Origin")

		lCtx := context.WithValue(context.WithValue(gCtx, loaders.LoadersKey, loader), helpers.UserKey, ctx.UserValue("user"))
		if wsTransport.Supports(ctx) {
			wsTransport.Do(ctx, lCtx, exec)
		} else {
			fasthttpadaptor.NewFastHTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				srv.ServeHTTP(w, r.WithContext(lCtx))
			}))(ctx)
		}

	}
}
