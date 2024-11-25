package service

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/msteinert/pam"
)

// AuthService отвечает за аутентификацию пользователей и управление сессиями.
type AuthService struct {
	mu       sync.Mutex
	sessions map[string]UserSession
}

// UserSession представляет активную пользовательскую сессию.
type UserSession struct {
	Username string
	Expires  time.Time
}

// NewAuthService создает новый экземпляр AuthService.
func NewAuthService() *AuthService {
	return &AuthService{
		sessions: make(map[string]UserSession),
	}
}

// Authenticate выполняет аутентификацию пользователя с помощью PAM.
func (a *AuthService) Authenticate(username, password string) error {
	tx, err := pam.StartFunc("", username, func(s pam.Style, msg string) (string, error) {
		switch s {
		case pam.PromptEchoOff:
			return password, nil
		case pam.PromptEchoOn:
			return password, nil
		case pam.ErrorMsg:
			return "", errors.New(msg)
		case pam.TextInfo:
			return "", nil
		default:
			return "", errors.New("unknown PAM message style")
		}
	})
	if err != nil {
		return err
	}
	return tx.Authenticate(0)
}

// GenerateSessionToken генерирует уникальный токен для сессии.
func (a *AuthService) GenerateSessionToken() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// CreateSession создает новую сессию для указанного пользователя.
func (a *AuthService) CreateSession(username string) (string, time.Time) {
	a.mu.Lock()
	defer a.mu.Unlock()

	token := a.GenerateSessionToken()
	expires := time.Now().Add(24 * time.Hour) // Длительность сессии: 24 часа

	a.sessions[token] = UserSession{
		Username: username,
		Expires:  expires,
	}

	return token, expires
}

// IsValidSession проверяет, действителен ли указанный токен.
func (a *AuthService) IsValidSession(token string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	session, exists := a.sessions[token]
	if !exists || session.Expires.Before(time.Now()) {
		delete(a.sessions, token) // Удаляем истекшие сессии
		return false
	}

	return true
}

// GetSessionUsername возвращает имя пользователя для указанного токена.
func (a *AuthService) GetSessionUsername(token string) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	session, exists := a.sessions[token]
	if !exists || session.Expires.Before(time.Now()) {
		return "", errors.New("invalid or expired session")
	}

	return session.Username, nil
}

// InvalidateSession удаляет сессию для указанного токена.
func (a *AuthService) InvalidateSession(token string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	delete(a.sessions, token)
}
