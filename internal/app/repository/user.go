package repository

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"loading_time/internal/app/ds"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
)

// NOTE: этот файл реализует: CreateUser, GetUserByLogin, RegisterUser,
// Authenticate, LoginUser, SaveJWTToken, SaveSession и работу с Redis.
// Он ориентирован на структуру Repository, у которой должны быть поля:
// db *gorm.DB, redisClient *redis.Client, jwtKey string
// (см. инструкцию внизу, если нужно инициализировать redisClient/jwtKey).

// GetUserByLogin returns user by login
func (r *Repository) GetUserByLogin(login string) (*ds.User, error) {
	user := &ds.User{}
	err := r.db.Where("login = ?", login).First(user).Error
	if err != nil {
		return nil, err
	}
	return user, nil
}

// CreateUser hashes password and saves new user
func (r *Repository) CreateUser(user *ds.User) error {
	if user.Password == "" {
		return fmt.Errorf("password is empty")
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("bcrypt generate error: %w", err)
	}
	fmt.Printf("DEBUG: CreateUser: generated hash for login=%s hash=%s\n", user.Login, string(hashed))
	user.Password = string(hashed)
	return r.db.Create(user).Error
}

// RegisterUser checks uniqueness and creates user
func (r *Repository) RegisterUser(user ds.User) (ds.User, error) {
	// проверка существует ли уже
	exist, err := r.GetUserByLogin(user.Login)
	if err == nil && exist != nil {
		return ds.User{}, fmt.Errorf("user already exists")
	}
	if user.Role == "" {
		user.Role = "creator"
	}
	if err := r.CreateUser(&user); err != nil {
		return ds.User{}, err
	}
	created, err := r.GetUserByLogin(user.Login)
	if err != nil {
		return ds.User{}, err
	}
	// не отдаём пароль наружу
	created.Password = ""
	return *created, nil
}

// Authenticate: возвращает пользователя, если логин+пароль верны
func (r *Repository) Authenticate(login, password string) (*ds.User, error) {
	fmt.Printf("DEBUG: Authenticate called for login=%s\n", login)

	user, err := r.GetUserByLogin(login)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	// сравниваем хэш
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		fmt.Printf("DEBUG: bcrypt compare failed for login=%s err=%v\n", login, err)
		return nil, fmt.Errorf("invalid password")
	}
	// не отдаём пароль дальше
	user.Password = ""
	return user, nil
}

// LoginUser: полноценный flow — проверка, генерация JWT, сохранение токена и сессии
func (r *Repository) LoginUser(login, password string) (jwtToken string, sessionID string, err error) {
	fmt.Printf("DEBUG: LoginUser called. db=%v redis=%v jwtKeyLen=%d\n", r.db, r.redisClient, len(r.jwtKey))

	// базовые проверки инициализации
	if r.db == nil {
		return "", "", fmt.Errorf("db is nil")
	}
	if r.redisClient == nil {
		return "", "", fmt.Errorf("redis client is nil")
	}
	if r.jwtKey == "" {
		return "", "", fmt.Errorf("jwt key is empty")
	}

	// проверка учётных данных
	user, err := r.GetUserByLogin(login)
	if err != nil {
		return "", "", fmt.Errorf("user not found: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		// ПЕЧАТАЕМ ХЭШ и введённый пароль (только временно для отладки!)
		fmt.Printf("DEBUG: password mismatch. dbHash=%s provided=%s err=%v\n", user.Password, password, err)
		return "", "", fmt.Errorf("неверный пароль")
	}

	// claims
	claims := jwt.MapClaims{
		"user_id": user.UserID,
		"role":    user.Role,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	}
	tokenObj := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := tokenObj.SignedString([]byte(r.jwtKey))
	if err != nil {
		return "", "", fmt.Errorf("jwt sign error: %w", err)
	}

	// Сохранить JWT в Redis: ключ "jwt:<userID>" -> token
	if err := r.SaveJWTToken(user.UserID, tokenStr); err != nil {
		return "", "", fmt.Errorf("save jwt token error: %w", err)
	}

	// создать session id
	sid := make([]byte, 16)
	if _, err := rand.Read(sid); err != nil {
		return "", "", fmt.Errorf("rand read session id error: %w", err)
	}
	sessionID = hex.EncodeToString(sid)

	// сохранить сессию: key "sess:<sessionID>" -> fields user_id, role
	if err := r.SaveSession(sessionID, user.UserID, user.Role, 24*time.Hour); err != nil {
		return "", "", fmt.Errorf("save session error: %w", err)
	}

	fmt.Printf("DEBUG: LoginUser success login=%s userID=%d session=%s\n", login, user.UserID, sessionID)
	return tokenStr, sessionID, nil
}

// SaveJWTToken stores token in redis with TTL
func (r *Repository) SaveJWTToken(userID int, token string) error {
	key := "jwt:" + strconv.Itoa(userID)
	err := r.redisClient.Set(context.Background(), key, token, 24*time.Hour).Err()
	if err != nil {
		fmt.Printf("DEBUG: SaveJWTToken error: %v\n", err)
		return err
	}
	fmt.Printf("DEBUG: SaveJWTToken saved key=%s\n", key)
	return nil
}

// SaveSession stores session map in redis
func (r *Repository) SaveSession(sessionID string, userID int, role string, ttl time.Duration) error {
	key := "sess:" + sessionID
	data := map[string]interface{}{
		"user_id": strconv.Itoa(userID),
		"role":    role,
	}
	if err := r.redisClient.HSet(context.Background(), key, data).Err(); err != nil {
		fmt.Printf("DEBUG: SaveSession HSet error: %v\n", err)
		return err
	}
	if err := r.redisClient.Expire(context.Background(), key, ttl).Err(); err != nil {
		fmt.Printf("DEBUG: SaveSession Expire error: %v\n", err)
		return err
	}
	fmt.Printf("DEBUG: SaveSession saved key=%s ttl=%v\n", key, ttl)
	return nil
}

//__________________________________________________________________________________________

// GetUserByID — получить пользователя по ID
func (r *Repository) GetUserByID(userID int) (*ds.User, error) {
	user := &ds.User{}
	if err := r.db.First(user, userID).Error; err != nil {
		return nil, err
	}
	return user, nil
}

// UpdateUser — обновить данные пользователя
func (r *Repository) UpdateUser(user ds.User) error {
	return r.db.Model(&ds.User{}).Where("user_id = ?", user.UserID).Updates(user).Error
}
