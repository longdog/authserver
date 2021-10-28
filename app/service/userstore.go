// +build !mock

package service

import (
	"auth/app/config"
	"context"
	"os"

	"github.com/getsentry/sentry-go"
	log "github.com/go-pkgz/lgr"
	"github.com/jackc/pgx/v4/pgxpool"
)

type UserStore struct {
	db *pgxpool.Pool
}

func NewUserStore(ctx context.Context, conf *config.Config) (*UserStore, error) {
	dbpool, err := pgxpool.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		sentry.CaptureException(err)
		log.Printf("[ERROR] Ошибка подключения к БД %+v", err)
		return nil, err
	}
	go func() {
		<-ctx.Done()
		dbpool.Close()
		log.Printf("[INFO] ПОдключение к БД закрыто")
		return
	}()
	return &UserStore{
		db: dbpool,
	}, nil
}

func (u *UserStore) GetUser(username string, password string) (*User, error) {

	_, err := u.db.Query(context.Background(), "SELECT e_user_login($1, $2)", username, password)
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

func (u *UserStore) GetUserById(userId int64) (*User, error) {

	return &User{
		Id:       1,
		Username: "admin",
	}, nil

}
