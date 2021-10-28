//go:generate qtc -dir=templates

package main

import (
	"auth/app/config"
	"auth/app/service"
	"auth/app/templates"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/fasthttp/router"
	"github.com/getsentry/sentry-go"
	log "github.com/go-pkgz/lgr"
	"github.com/joho/godotenv"
	"github.com/sethvargo/go-signalcontext"
	"github.com/valyala/fasthttp"
)

var version = "dev"

var us *service.UserService

var node *snowflake.Node

var conf *config.Config

var (
	strContentType     = []byte("Content-Type")
	strApplicationJSON = []byte("application/json")
)

func main() {

	logId := time.Now().UnixNano()

	log.Printf("[INFO] Запуск сервера v%s %d", version, logId)

	// Загрузка конфигов
	err := godotenv.Load()
	if err != nil {
		log.Printf("[ERROR] Не могу прочитать конфигурации из .env файла, %+v", err)
		return
	}
	conf = config.NewConfig()

	// сервис сбора логов
	err = sentry.Init(sentry.ClientOptions{
		Dsn: conf.SentryDsn,
	})
	if err != nil {
		log.Printf("[ERROR] Не могу подключиться к sentry, %+v", err)
		return
	}

	// portArg := flag.String("port", "3003", "port")
	// prefixArg := flag.String("prefix", "", "url prefix")
	// flag.Parse()

	// arguments := os.Args
	// if len(arguments) == 1 {
	// 	fmt.Println("Укажите порт")
	// 	return
	// }

	us = service.NewUserService(&service.UserModel{})

	ctx, done := signalcontext.OnInterrupt()
	defer func() {
		if err := recover(); err != nil {
			log.Printf("[ERROR] Неожиданное завершение, %+v", err)
			done()
		}
	}()

	defer done()

	go func() {

		err := realMain(ctx, conf.Port, conf.UrlPrefix)
		if err != nil {
			log.Printf("[ERROR] Ошибка сервера, %+v", err)
			done()
		}
	}()
	<-ctx.Done()
	sentry.Flush(2 * time.Second)
	log.Printf("[INFO] Сервер остановлен %d", logId)
}

func Index(prefix string) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		// get app and redirect query params
		app := ctx.QueryArgs().Peek("app")
		redirect := ctx.QueryArgs().Peek("redirect")
		errCode := ctx.QueryArgs().Peek("err")
		if len(app) == 0 || len(redirect) == 0 {
			ctx.Response.SetStatusCode(404)
			return
		}

		userId, _ := GetCookie(ctx)

		if userId > 0 {
			code, err := us.GetCode(string(app), userId)

			// Ошибка БД
			if err != nil {
				ctx.Response.SetStatusCode(500)
				return
			}
			if code != "" {
				RedirectToApp(string(redirect), code, ctx)
				return
			}
		}

		errMsg := make([]byte, 0)
		if len(errCode) > 0 {
			errMsg = []byte("Такого пользователя не существует")
		}
		p := &templates.MainPage{
			Locale:   []byte("ru"),
			Prefix:   []byte(prefix),
			App:      []byte(app),
			Redirect: []byte(redirect),
			Error:    errMsg,
		}
		ctx.Response.Header.Set("Content-Type", "text/html;charset=UTF-8")
		templates.WritePageTemplate(ctx, p)
	}
}

func SetCookie(userId int64, ctx *fasthttp.RequestCtx) {
	var c fasthttp.Cookie
	c.SetKey("user")
	c.SetValue(strconv.FormatInt(userId, 10))
	c.SetMaxAge(24 * 60 * 60) // сутки
	c.SetHTTPOnly(true)
	c.SetSameSite(fasthttp.CookieSameSiteLaxMode)
	ctx.Response.Header.SetCookie(&c)
}

func RemoveCookie(ctx *fasthttp.RequestCtx) {
	ctx.Response.Header.DelCookie("user")
}

func GetCookie(ctx *fasthttp.RequestCtx) (int64, error) {
	user := ctx.Request.Header.Cookie("user")
	fmt.Println(user)
	if len(user) == 0 {
		return 0, errors.New("user cookie not found")
	}
	id, err := strconv.ParseInt(string(user), 10, 64)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func Login(prefix string) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		RemoveCookie(ctx)
		app := ctx.PostArgs().Peek("app")
		redirect := ctx.PostArgs().Peek("redirect")
		username := ctx.PostArgs().Peek("username")
		password := ctx.PostArgs().Peek("password")

		if len(app) == 0 || len(redirect) == 0 || len(username) == 0 || len(password) == 0 {
			fmt.Fprint(ctx, "Ошибка параметров")
			ctx.Response.SetStatusCode(400)
			return
		}
		userId, code, err := us.Login(string(app), string(username), string(password))
		// Ошибка БД
		if err != nil {
			ctx.Response.SetStatusCode(500)
			return
		}

		// Логин или пароль неправильные
		if code == "" {
			ctx.Redirect(fmt.Sprintf("%s?app=%s&redirect=%s&err=404", prefix, app, redirect), 303)
			return
		}
		SetCookie(userId, ctx)
		RedirectToApp(string(redirect), code, ctx)
	}
}

func RedirectToApp(redirect string, code string, ctx *fasthttp.RequestCtx) {
	decodedUrl, err := url.QueryUnescape(redirect)
	if err != nil {
		log.Printf("[WARN] Ошибка редиректа, %s, %+v", redirect, err)
		ctx.Response.SetStatusCode(400)
		fmt.Fprint(ctx, "Ошибка параметров: "+err.Error())
		return
	}
	urlCode := decodedUrl
	if strings.Contains(decodedUrl, "?") {
		urlCode = urlCode + "&code=" + code
	} else {
		urlCode = urlCode + "?code=" + code
	}
	ctx.Redirect(urlCode, 303)
}

func Logout(prefix string) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		userId, err := GetCookie(ctx)
		if err == nil {
			us.Logout(userId)
		}
		RemoveCookie(ctx)
		ctx.Redirect(prefix+"/", 303)
	}
}

func doJSONWrite(ctx *fasthttp.RequestCtx, code int, obj interface{}) {
	ctx.Response.Header.SetCanonical(strContentType, strApplicationJSON)
	ctx.Response.SetStatusCode(code)
	if err := json.NewEncoder(ctx).Encode(obj); err != nil {
		log.Printf("[ERROR] Ошибка отправки json, %+v, %+v", obj, err)
		sentry.CaptureException(err)
		ctx.Error(err.Error(), fasthttp.StatusInternalServerError)
	}
}

func LoginApi(ctx *fasthttp.RequestCtx) {
	var u service.LoginData
	err := json.Unmarshal(ctx.PostBody(), &u)
	if err != nil {
		log.Printf("[WARN] Ошибка парсинга json, %s, %+v", string(ctx.PostBody()), err)
		ctx.Response.SetStatusCode(400)
		return
	}

	code, token, err := us.LoginApi(u.App, u.Username, u.Password)
	fmt.Printf("user %+v", err)
	if err != nil {
		ctx.Response.SetStatusCode(404)
		return
	}
	ret := service.TokenData{
		App:          u.App,
		RefreshToken: code,
		JwtToken:     token,
	}
	doJSONWrite(ctx, 200, ret)
}

func RefreshTokenApi(ctx *fasthttp.RequestCtx) {
	var u service.RefreshData
	err := json.Unmarshal(ctx.PostBody(), &u)
	if err != nil {
		log.Printf("[WARN] Ошибка парсинга json, %s, %+v", string(ctx.PostBody()), err)
		ctx.Response.SetStatusCode(400)
		return
	}
	code, token, err := us.GetNewToken(u.App, u.RefreshToken)

	if err != nil {
		ctx.Response.SetStatusCode(404)
		return
	}
	ret := service.TokenData{
		App:          u.App,
		RefreshToken: code,
		JwtToken:     token,
	}
	doJSONWrite(ctx, 200, ret)
}

func ValidateRpcData(messageMAC, message []byte) bool {
	//h := blake3.New(256, nil)
	// out := make([]byte, 0)
	// h.Write(out)
	// hash2 := hex.EncodeToString(h.Sum(body))
	// fmt.Println(hash2)
	// return true, nil
	mac := hmac.New(sha256.New, []byte("TEST"))
	mac.Write(message)
	expectedMAC := mac.Sum(nil)
	return hmac.Equal(messageMAC, expectedMAC)
}

func ValidateRpc(ctx *fasthttp.RequestCtx) bool {
	reqHash := ctx.Request.Header.Peek("hash")
	body := ctx.PostBody()
	return ValidateRpcData(reqHash[:], body[:])
}

func LogoutProc(ctx *fasthttp.RequestCtx) {
	ValidateRpc(ctx)
	// TODO:
	// isValid, err := ValidateRpc(ctx)
	// if err != nil {
	// 	ctx.SetStatusCode(500)
	// 	return
	// }
	// if !isValid {
	// 	ctx.SetStatusCode(400)
	// 	return
	// }

	ctx.SetStatusCode(200)
}

func realMain(ctx context.Context, port string, prefix string) error {

	r := router.New()
	r.GET(prefix+"/", Index(prefix))
	r.GET(prefix+"/logout", Logout(prefix))
	r.POST(prefix+"/login", Login(prefix))
	// external rest api
	r.POST("/api/login", LoginApi)
	// r.GET("/api/current", GetCurrentUserApi)
	// r.POST("/api/token", GetTokenApi)
	r.POST("/api/refresh", RefreshTokenApi)
	// rpc
	r.GET("/rpc/logout", LogoutProc)

	return fasthttp.ListenAndServe(":"+port, r.Handler)
}
