package twitch

import (
	"encoding/hex"
	"fmt"
	"math/rand"
	"net/url"
	"strings"
	"time"

	"github.com/fasthttp/router"
	"github.com/golang-jwt/jwt"
	jsoniter "github.com/json-iterator/go"
	"github.com/nicklaw5/helix"
	"github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"
	"github.com/viderstv/api/src/global"
	"github.com/viderstv/common/structures"
	"github.com/viderstv/common/svc/mongo"
	"github.com/viderstv/common/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

type TwitchState struct {
	Token    string    `json:"token"`
	ReturnTo string    `json:"return_to"`
	Expiry   time.Time `json:"expiry"`
}

type TwitchOtpRequest struct {
	Token string `json:"token"`
}

type TwitchOtpResponse struct {
	AccessToken string    `json:"access_token,omitempty"`
	ExpiresAt   time.Time `json:"expires_at,omitempty"`
	Error       string    `json:"error,omitempty"`
}

func Handle(gCtx global.Context, authWrapper func(handler fasthttp.RequestHandler) fasthttp.RequestHandler, r *router.Group) {
	r.GET("/login", func(ctx *fasthttp.RequestCtx) {
		csrf, _ := utils.GenerateRandomBytes(32)
		state, _ := json.Marshal(TwitchState{
			Token:    hex.EncodeToString(csrf),
			ReturnTo: utils.B2S(ctx.QueryArgs().Peek("return_to")),
			Expiry:   time.Now().Add(time.Minute * 5),
		})

		cookie := &fasthttp.Cookie{}
		cookie.SetDomain(gCtx.Config().Frontend.Cookie.Domain)
		cookie.SetSecure(gCtx.Config().Frontend.Cookie.Secure)
		cookie.SetHTTPOnly(true)
		cookie.SetExpire(time.Now().Add(time.Minute * 5))
		cookie.SetKey("twitch_csrf")
		cookie.SetValue(hex.EncodeToString(csrf))
		ctx.Response.Header.AddBytesV("Set-Cookie", cookie.Cookie())

		query := url.Values{}
		query.Set("client_id", gCtx.Config().Twitch.ClientID)
		query.Set("redirect_uri", gCtx.Config().Twitch.LoginRedirectURI)
		query.Set("response_type", "code")
		query.Set("state", hex.EncodeToString(state))

		ctx.Redirect(fmt.Sprintf("https://id.twitch.tv/oauth2/authorize?%s", query.Encode()), fasthttp.StatusTemporaryRedirect)
	})

	r.GET("/login/callback", func(ctx *fasthttp.RequestCtx) {
		code := utils.B2S(ctx.QueryArgs().Peek("code"))

		state, _ := hex.DecodeString(utils.B2S(ctx.QueryArgs().Peek("state")))
		pl := TwitchState{}
		_ = json.Unmarshal(state, &pl)

		stateCookie := utils.B2S(ctx.Request.Header.Cookie("twitch_csrf"))

		if len(code) == 0 || len(state) == 0 || pl.Token != stateCookie || pl.Expiry.Before(time.Now()) {
			ctx.Redirect(fmt.Sprintf("%s?error=twitch_login_error", gCtx.Config().Frontend.ErrorUrl), fasthttp.StatusTemporaryRedirect)
			return
		}

		client, err := helix.NewClient(&helix.Options{
			ClientID:     gCtx.Config().Twitch.ClientID,
			ClientSecret: gCtx.Config().Twitch.ClientSecret,
			RedirectURI:  gCtx.Config().Twitch.LoginRedirectURI,
		})
		if err != nil {
			logrus.Error("failed to create helix client: ", err)
			ctx.Redirect(fmt.Sprintf("%s?error=twitch_login_error&internal=true", gCtx.Config().Frontend.ErrorUrl), fasthttp.StatusTemporaryRedirect)
			return
		}

		tokenResp, err := client.RequestUserAccessToken(code)
		if err != nil {
			logrus.Error("failed to create helix client: ", err)
			ctx.Redirect(fmt.Sprintf("%s?error=twitch_login_error&internal=true", gCtx.Config().Frontend.ErrorUrl), fasthttp.StatusTemporaryRedirect)
			return
		}

		if tokenResp.Data.AccessToken == "" {
			ctx.Redirect(fmt.Sprintf("%s?error=twitch_login_error", gCtx.Config().Frontend.ErrorUrl), fasthttp.StatusTemporaryRedirect)
			return
		}

		client.SetUserAccessToken(tokenResp.Data.AccessToken)
		userResp, err := client.GetUsers(&helix.UsersParams{})
		if err != nil {
			logrus.Error("failed to create helix client: ", err)
			ctx.Redirect(fmt.Sprintf("%s?error=twitch_login_error&internal=true", gCtx.Config().Frontend.ErrorUrl), fasthttp.StatusTemporaryRedirect)
			return
		}

		if len(userResp.Data.Users) != 1 {
			ctx.Redirect(fmt.Sprintf("%s?error=twitch_login_error", gCtx.Config().Frontend.ErrorUrl), fasthttp.StatusTemporaryRedirect)
			return
		}

		user := userResp.Data.Users[0]

		color := structures.NewColor(byte(rand.Intn(255)), byte(rand.Intn(255)), byte(rand.Intn(255)), 255)

		dbUser := structures.User{}
		streamKey, _ := utils.GenerateRandomBytes(16)
		res := gCtx.Inst().Mongo.Collection(mongo.CollectionNameUsers).FindOneAndUpdate(ctx, bson.M{
			"twitch_account.id": user.ID,
		}, bson.M{
			"$set": bson.M{
				"login":        user.Login,
				"display_name": user.DisplayName,
				"twitch_account": structures.TwitchAccount{
					ID:             user.ID,
					Login:          user.Login,
					DisplayName:    user.DisplayName,
					ProfilePicture: user.ProfileImageURL,
				},
			},
			"$setOnInsert": bson.M{
				"color": color,
				"role":  structures.GlobalRoleUser,
				"channel": structures.Channel{
					Public:    true,
					StreamKey: hex.EncodeToString(streamKey),
					Emotes:    []structures.Emote{},
				},
				"memberships": []structures.Member{},
			},
		}, options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After))
		err = res.Err()
		if err == nil {
			err = res.Decode(&dbUser)
		}
		if err != nil {
			logrus.Error("mongo upsert failed on twitch login: ", err)
			ctx.Redirect(fmt.Sprintf("%s?error=twitch_login_error&internal=true", gCtx.Config().Frontend.ErrorUrl), fasthttp.StatusTemporaryRedirect)
			return
		}

		_key, _ := utils.GenerateRandomBytes(64)

		key := hex.EncodeToString(_key)
		if err := gCtx.Inst().Redis.SetEX(ctx, fmt.Sprintf("otp:twitch_login:%s", key), dbUser.ID.Hex(), time.Second*90); err != nil {
			logrus.Error("redis otp failed on twitch login: ", err)
			ctx.Redirect(fmt.Sprintf("%s?error=twitch_login_error&internal=true", gCtx.Config().Frontend.ErrorUrl), fasthttp.StatusTemporaryRedirect)
			return
		}

		query := url.Values{}
		query.Set("otp", key)
		query.Set("return_to", pl.ReturnTo)

		ctx.Redirect(fmt.Sprintf("%s?%s", gCtx.Config().Frontend.OtpUrl, query.Encode()), fasthttp.StatusTemporaryRedirect)
	})

	r.POST("/login/otp", func(ctx *fasthttp.RequestCtx) {
		origin := utils.B2S(ctx.Request.Header.Peek("Origin"))
		if origin != "" {
			for _, v := range gCtx.Config().Frontend.CORS.Origins {
				if v == origin {
					goto next
				}
			}

			data, _ := json.Marshal(TwitchOtpResponse{
				Error: "bad origin",
			})
			ctx.SetBody(data)
			ctx.SetContentType("application/json")
			ctx.SetStatusCode(400)
			return
		}

	next:
		if origin != "" {
			ctx.Response.Header.Set("Access-Control-Allow-Headers", "Content-Type")
			ctx.Response.Header.Set("Access-Control-Allow-Origin", origin)
			ctx.Response.Header.Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		}

		req := TwitchOtpRequest{}
		switch strings.ToLower(utils.B2S(ctx.Request.Header.ContentType())) {
		case "application/json":
			_ = json.Unmarshal(ctx.Request.Body(), &req)
		}

		if req.Token == "" {
			data, _ := json.Marshal(TwitchOtpResponse{
				Error: "invalid otp",
			})
			ctx.SetBody(data)
			ctx.SetContentType("application/json")
			ctx.SetStatusCode(400)
			return
		}

		pipe := gCtx.Inst().Redis.Pipeline()
		getCmd := pipe.Get(ctx, fmt.Sprintf("otp:twitch_login:%s", req.Token))
		pipe.Del(ctx, fmt.Sprintf("otp:twitch_login:%s", req.Token))
		_, err := pipe.Exec(ctx)
		val := getCmd.Val()
		if err != nil || val == "" {
			data, _ := json.Marshal(TwitchOtpResponse{
				Error: "invalid otp",
			})
			ctx.SetBody(data)
			ctx.SetContentType("application/json")
			ctx.SetStatusCode(400)
			return
		}

		uID, err := primitive.ObjectIDFromHex(val)
		if err != nil {
			data, _ := json.Marshal(TwitchOtpResponse{
				Error: "invalid otp",
			})
			ctx.SetBody(data)
			ctx.SetContentType("application/json")
			ctx.SetStatusCode(400)
			return
		}

		now := time.Now()
		expiresAt := now.Add(time.Hour * 24 * 14)
		login := structures.JwtLogin{
			UserID: uID,
			StandardClaims: jwt.StandardClaims{
				ExpiresAt: expiresAt.Unix(),
				IssuedAt:  now.Unix(),
				Issuer:    "api:twitch_login",
			},
		}

		tkn, err := structures.EncodeJwt(login, gCtx.Config().Auth.JwtToken)
		if err != nil {
			logrus.Error("failed to sign jwt: ", err)
			data, _ := json.Marshal(TwitchOtpResponse{
				Error: "internal server error",
			})
			ctx.SetBody(data)
			ctx.SetContentType("application/json")
			ctx.SetStatusCode(500)
			return
		}

		data, _ := json.Marshal(TwitchOtpResponse{
			AccessToken: tkn,
			ExpiresAt:   expiresAt,
		})
		ctx.SetBody(data)
		ctx.SetContentType("application/json")
		ctx.SetStatusCode(200)
	})
}
