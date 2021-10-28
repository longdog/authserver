// +build !mock

package service

import "auth/app/config"

type UserModel struct {
	conf *config.Config
}

func (u *UserModel) GetUser(username string, password string) (*User, error) {
	var mockUsername = "admin"
	var mockPassword = "password"
	if username == mockUsername && password == mockPassword {
		return &User{
			Id:       1,
			Username: "admin",
		}, nil
	}
	return nil, nil
}

func (u *UserModel) GetUserById(userId int64) (*User, error) {

	return &User{
		Id:       1,
		Username: "admin",
	}, nil

}
