package repository

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"loading_time/internal/app/ds"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
)

func (r *Repository) GetUserByID(userID int) (*ds.User, error) {
	user := &ds.User{}
	err := r.db.First(user, userID).Error
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *Repository) UpdateUser(user ds.User) error {
	return r.db.Model(&ds.User{}).Where("user_id = ?", user.UserID).Updates(user).Error
}

func (r *Repository) GetUserByLogin(login string) (*ds.User, error) {
	user := &ds.User{}
	err := r.db.Where("login = ?", login).First(user).Error
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *Repository) CreateUser(user *ds.User) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.Password = string(hashedPassword)
	return r.db.Create(user).Error
}

func (r *Repository) SaveJWTToken(userID int, token string) error {
	expiration := 24 * time.Hour // синхронизируем с LoginUser
	idStr := strconv.Itoa(userID)
	err := r.Redis().Set(context.Background(), "jwt:"+idStr, token, expiration).Err()
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
	user.Role = "creator"
	err = r.CreateUser(&user)
	if err != nil {
		return ds.User{}, err
	}

	createdUser, err := r.GetUserByLogin(user.Login)
	if err != nil {
		return ds.User{}, err
	}
	return *createdUser, nil
}

func (r *Repository) LoginUser(login, password string) (jwtToken string, sessionID string, err error) {
	fmt.Printf("DEBUG: r.db = %v\n", r.db)
	if r.db == nil {
		return "", "", fmt.Errorf("db is nil")
	}

	candidate, err := r.GetUserByLogin(login)
	if err != nil {
		return "", "", fmt.Errorf("пользователь не найден: %v", err)
	}

	// ПРЯМОЕ СРАВНЕНИЕ ХЭША И ПАРОЛЯ
	if err := bcrypt.CompareHashAndPassword([]byte(candidate.Password), []byte(password)); err != nil {
		return "", "", fmt.Errorf("неверный пароль")
	}

	// JWT
	claims := jwt.MapClaims{
		"user_id": candidate.UserID,
		"role":    candidate.Role,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(r.JWTKey()))
	if err != nil {
		return "", "", err
	}

	if err := r.SaveJWTToken(candidate.UserID, tokenStr); err != nil {
		return "", "", err
	}

	// Сессия
	sid := make([]byte, 16)
	if _, err := rand.Read(sid); err != nil {
		return "", "", err
	}
	sessionID = hex.EncodeToString(sid)
	if err := r.SaveSession(sessionID, candidate.UserID, candidate.Role, 24*time.Hour); err != nil {
		return "", "", err
	}

	return tokenStr, sessionID, nil
}
func (r *Repository) SaveSession(sessionID string, userID int, role string, ttl time.Duration) error {
	key := "sess:" + sessionID
	data := map[string]interface{}{
		"user_id": strconv.Itoa(userID),
		"role":    role,
	}
	err := r.Redis().HMSet(context.Background(), key, data).Err()
	if err != nil {
		return err
	}
	return r.Redis().Expire(context.Background(), key, ttl).Err()
}
