package main

import (
	"auth/app/service"
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttputil"
)

func init() {
	us = service.NewUserService(&service.UserModel{})
}

func serve(handler fasthttp.RequestHandler, req *http.Request) (*http.Response, error) {
	ln := fasthttputil.NewInmemoryListener()
	defer ln.Close()

	go func() {
		err := fasthttp.Serve(ln, handler)
		if err != nil {
			panic(fmt.Errorf("failed to serve: %v", err))
		}
	}()

	client := http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return ln.Dial()
			},
		},
	}

	return client.Do(req)
}

func TestIndex(t *testing.T) {

	t.Run("Test page without app and redirect params return status 404", func(t *testing.T) {
		r, err := http.NewRequest("GET", "http://test/", nil)
		if err != nil {
			t.Error(err)
		}
		res, err := serve(Index(""), r)
		if err != nil {
			t.Error(err)
		}
		assert.Equal(t, res.StatusCode, 404)
	})

	t.Run("Test login form render", func(t *testing.T) {
		r, err := http.NewRequest("GET", "http://test?app=client&redirect=http://client/form", nil)
		if err != nil {
			t.Error(err)
		}
		res, err := serve(Index(""), r)
		if err != nil {
			t.Error(err)
		}

		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			t.Error(err)
		}
		if !strings.Contains(string(body), "loginForm") {
			t.Error("Login form not render on Index Page")
		}
	})

	t.Run("Test url prefix", func(t *testing.T) {
		r, err := http.NewRequest("GET", "http://test?app=client&redirect=http://client/form", nil)
		if err != nil {
			t.Error(err)
		}
		res, err := serve(Index("/fooprefix"), r)
		if err != nil {
			t.Error(err)
		}

		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			t.Error(err)
		}
		if !strings.Contains(string(body), "/fooprefix/login") {
			t.Error("Url prefix not apply in form action")
		}
	})

	t.Run("Test app and redirect params", func(t *testing.T) {
		r, err := http.NewRequest("GET", "http://test?app=client&redirect=http://client/form", nil)
		if err != nil {
			t.Error(err)
		}
		res, err := serve(Index("/fooprefix"), r)
		if err != nil {
			t.Error(err)
		}

		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			t.Error(err)
		}
		if !strings.Contains(string(body), "value=\"client\"") {
			t.Error("app param is not in login form")
		}
		if !strings.Contains(string(body), "value=\"http%3A%2F%2Fclient%2Fform\"") {
			t.Error("app redirect is not in login form")
		}
	})

	t.Run("Test error message", func(t *testing.T) {
		r, err := http.NewRequest("GET", "http://test?app=client&redirect=http://client/form&err=404", nil)
		if err != nil {
			t.Error(err)
		}
		res, err := serve(Index("/fooprefix"), r)
		if err != nil {
			t.Error(err)
		}

		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			t.Error(err)
		}
		if !strings.Contains(string(body), "loginForm__row--error") {
			t.Error("error message not found")
		}
	})

}

func TestLogin(t *testing.T) {
	t.Run("Test page without app, redirect, username and password form params return status 400", func(t *testing.T) {
		r, err := http.NewRequest("POST", "http://test/", nil)
		if err != nil {
			t.Error(err)
		}
		res, err := serve(Login(""), r)
		if err != nil {
			t.Error(err)
		}
		assert.Equal(t, res.StatusCode, 400)
	})

	t.Run("Test login with correct username and password", func(t *testing.T) {
		data := url.Values{}
		data.Set("app", "client")
		data.Set("redirect", "http://client/form")
		data.Set("username", "admin")
		data.Set("password", "password")
		r, err := http.NewRequest("POST", "http://test/login", strings.NewReader(data.Encode()))
		r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		r.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))
		if err != nil {
			t.Error(err)
		}
		res, err := serve(Login(""), r)
		if err != nil {
			t.Error(err)
		}
		fmt.Println(res.Request.URL)
		assert.Equal(t, "/form", res.Request.URL.Path)
	})

	t.Run("Test login with incorrect correct username and password", func(t *testing.T) {
		data := url.Values{}
		data.Set("app", "client")
		data.Set("redirect", "http://client/form")
		data.Set("username", "admin1")
		data.Set("password", "password")
		r, err := http.NewRequest("POST", "http://test/login", strings.NewReader(data.Encode()))
		r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		r.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))
		if err != nil {
			t.Error(err)
		}
		res, err := serve(Login(""), r)
		if err != nil {
			t.Error(err)
		}
		fmt.Println(res.Request.URL)
		assert.Equal(t, "/login", res.Request.URL.Path)
	})
}
