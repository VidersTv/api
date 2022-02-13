package api

import (
	"strings"
	"time"

	"github.com/fasthttp/router"
	"github.com/viderstv/api/src/api/loaders"
	"github.com/viderstv/api/src/api/twitch"
	"github.com/viderstv/api/src/global"
	"github.com/viderstv/common/structures"
	"github.com/viderstv/common/utils"

	"github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"
)

func New(gCtx global.Context) <-chan struct{} {
	done := make(chan struct{})
	loader := loaders.New(gCtx)

	gql := GqlHandler(gCtx, loader)
	authWrapper := func(handler fasthttp.RequestHandler) fasthttp.RequestHandler {
		return func(ctx *fasthttp.RequestCtx) {
			auth := utils.B2S(ctx.Request.Header.Peek("Authorization"))

			if strings.HasPrefix(auth, "Bearer ") {
				auth := strings.TrimPrefix(auth, "Bearer ")
				login := structures.JwtLogin{}
				if err := structures.DecodeJwt(&login, gCtx.Config().Auth.JwtToken, auth); err != nil {
					logrus.Error("err: ", err)
					goto handler
				}

				if time.Unix(login.ExpiresAt, 0).Before(time.Now()) {
					logrus.Error("expired")
					goto handler
				}

				ctx.SetUserValue("user", &login.UserID)
			}

		handler:
			handler(ctx)
		}
	}

	router := router.New()

	router.GET("/gql", authWrapper(gql))
	router.POST("/gql", authWrapper(gql))

	router.HandleOPTIONS = true
	router.GlobalOPTIONS = func(ctx *fasthttp.RequestCtx) {
		origin := utils.B2S(ctx.Request.Header.Peek("Origin"))
		if origin != "" {
			for _, v := range gCtx.Config().Frontend.CORS.Origins {
				if v == origin {
					goto next
				}
			}

			ctx.SetStatusCode(fasthttp.StatusForbidden)
			return
		}

	next:
		ctx.Response.Header.Set("Vary", "Origin")
		if origin != "" {
			ctx.Response.Header.Set("Access-Control-Allow-Headers", "Content-Type,Authorization")
			ctx.Response.Header.Set("Access-Control-Allow-Origin", origin)
			ctx.Response.Header.Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			ctx.Response.Header.Set("Access-Control-Allow-Credentials", "true")
		}

		ctx.SetStatusCode(fasthttp.StatusNoContent)
	}

	twitch.Handle(gCtx, authWrapper, router.Group("/twitch"))

	server := fasthttp.Server{
		Handler: func(ctx *fasthttp.RequestCtx) {
			start := time.Now()
			defer func() {
				var err interface{}
				// err := recover()
				if err != nil {
					ctx.Response.SetStatusCode(fasthttp.StatusInternalServerError)
				}
				// gCtx.Inst().Prometheus.ResponseTimeMilliseconds().Observe(float64(time.Since(start)/time.Microsecond) / 1000)
				l := logrus.WithFields(logrus.Fields{
					"status":     ctx.Response.StatusCode(),
					"duration":   int64(time.Since(start) / time.Millisecond),
					"entrypoint": "api",
					"path":       utils.B2S(ctx.Path()),
				})
				if err != nil {
					l.Error("panic in handler: ", err)
				} else {
					l.Info("")
				}
			}()
			router.Handler(ctx)
		},
		ReadTimeout:     time.Second * 10,
		WriteTimeout:    time.Second * 10,
		CloseOnShutdown: true,
		Name:            "Viders",
	}

	go func() {
		if err := server.ListenAndServe(gCtx.Config().API.Bind); err != nil {
			logrus.Fatal("failed to start api server: ", err)
		}
		close(done)
	}()

	go func() {
		<-gCtx.Done()
		_ = server.Shutdown()
	}()

	return done
}
