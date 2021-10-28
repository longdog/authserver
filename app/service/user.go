package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/dgrijalva/jwt-go"
	"github.com/muesli/cache2go"
)

type User struct {
	Id          int64
	Username    string
	Permissions []int
}

type LoginData struct {
	App,
	Username,
	Password string
}

type TokenData struct {
	App          string `json:"app"`
	RefreshToken string `json:"refreshToken"`
	JwtToken     string `json:"jwtToken"`
}

type RefreshData struct {
	App          string `json:"app"`
	RefreshToken string `json:"refreshToken"`
}

type IUserModel interface {
	GetUser(username string, password string) (*User, error)
	GetUserById(userId int64) (*User, error)
}

type UserService struct {
	Model IUserModel
	db    *cache2go.CacheTable
	node  *snowflake.Node
}

func NewUserService(userModel IUserModel) *UserService {
	db := cache2go.Cache("user")
	node, _ := snowflake.NewNode(1)
	return &UserService{
		Model: userModel,
		db:    db,
		node:  node,
	}
}

func (u *UserService) Login(app, username, password string) (int64, string, error) {
	user, err := u.Model.GetUser(username, password)
	if err != nil {
		return 0, "", err
	}
	if user == nil {
		return 0, "", errors.New("user not found")
	}
	code, err := u.GetCode(app, user.Id)
	return user.Id, code, err
}

func (u *UserService) GetCode(app string, userId int64) (string, error) {

	code := u.node.Generate().String()

	u.db.Delete(fmt.Sprintf("logout.%d", userId))
	u.db.Add("code."+app+"."+code, 10*time.Second, userId)

	return code, nil
}

func (u *UserService) GetUserById(userId int64) (*User, error) {
	user, err := u.Model.GetUserById(userId)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("user not found")
	}
	return user, nil
}

func (u *UserService) GetNewToken(app, refreshToken string) (string, string, error) {

	res, err := u.db.Value("code." + app + "." + refreshToken)
	if err != nil {
		return "", "", err
	}
	userId := res.Data().(int64)

	_, err = u.db.Value(fmt.Sprintf("logout.%d", userId))
	if err == nil {
		return "", "", err
	}

	user, err := u.Model.GetUserById(userId)
	if err != nil {
		return "", "", err
	}
	if user == nil {
		return "", "", errors.New("user not found")
	}
	u.db.Delete("code." + app + "." + refreshToken)
	code := u.node.Generate().String()
	u.db.Add("code."+app+"."+code, 60*time.Minute, user.Id)

	token, err := u.GenerateJwt(user.Id, app, user.Username, user.Permissions)
	if err != nil {
		return "", "", err
	}

	return code, token, nil
}

func (u *UserService) LoginApi(app, username, password string) (string, string, error) {
	user, err := u.Model.GetUser(username, password)
	if err != nil {
		return "", "", err
	}
	if user == nil {
		return "", "", errors.New("user not found")
	}
	code := u.node.Generate().String()
	u.db.Delete(fmt.Sprintf("logout.%d", user.Id))
	u.db.Add("code."+app+"."+code, 60*time.Minute, user.Id)
	token, err := u.GenerateJwt(user.Id, app, user.Username, user.Permissions)
	if err != nil {
		return "", "", err
	}
	return code, token, nil
}

func (u *UserService) GenerateJwt(userId int64, app, username string, permissions []int) (string, error) {
	atClaims := jwt.MapClaims{}
	atClaims["userId"] = userId
	atClaims["app"] = app
	atClaims["username"] = username
	atClaims["permissions"] = permissions
	atClaims["exp"] = time.Now().Add(time.Minute * 15).Unix()
	at := jwt.NewWithClaims(jwt.SigningMethodHS256, atClaims)
	token, err := at.SignedString([]byte("ACCESS_SECRET"))
	if err != nil {
		return "", err
	}
	return token, nil
}

func (u *UserService) Logout(userId int64) error {
	u.db.Add(fmt.Sprintf("logout.%d", userId), 61*time.Minute, true)
	return nil
}
