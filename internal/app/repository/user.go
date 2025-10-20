package repository

import (
	"context"
	"errors"
	"loading_time/internal/app/ds"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
)

func (r *Repository) GetUserByLogin(login string) (*ds.User, error) {
	user := &ds.User{}
	err := r.db.Where("login = ?", login).First(user).Error
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *Repository) CreateUser(user *ds.User) error {
	return r.db.Create(user).Error
}

func (r *Repository) ComparePassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

func (r *Repository) SaveJWTToken(userID int, token string) error {
	expiration := 1 * time.Hour
	idStr := strconv.Itoa(userID)
	err := r.Redis().Set(context.Background(), idStr, token, expiration).Err()
	if err != nil {
		return err
	}
	return nil
}

func (r *Repository) RegisterUser(user ds.User) (ds.User, error) {
	candidate, err := r.GetUserByLogin(user.Login)
	if err == nil && candidate.Login == user.Login {
		return ds.User{}, errors.New("такой пользователь уже существует")
	}
	err = r.CreateUser(&user)
	if err != nil {
		return ds.User{}, err
	}
	return user, nil
}

func (r *Repository) LoginUser(user ds.User) (string, error) {
	candidate, err := r.GetUserByLogin(user.Login)
	if err != nil {
		return "", err
	}
	err = r.ComparePassword(candidate.Password, user.Password)
	if err != nil {
		return "", errors.New("пароли не совпали")
	}
	claims := jwt.MapClaims{
		"user_id":      candidate.UserID,
		"is_moderator": candidate.IsModerator,
		"exp":          time.Now().Add(time.Hour * 1).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(r.JWTKey()))
	if err != nil {
		return "", err
	}
	err = r.SaveJWTToken(candidate.UserID, tokenStr)
	if err != nil {
		return "", err
	}
	return tokenStr, nil
}
