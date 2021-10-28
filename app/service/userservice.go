package service

import "auth/app/config"

type IUserStore interface {
	GetUser(username string, password string) (*User, error)
	GetUserById(userId int64) (*User, error)
}

func NewUserStore(conf *config.Config) (*UserService, error) {

	return &UserService{}, nil
}
