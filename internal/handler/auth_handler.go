package handler

import (
	"encoding/json"
	"fileStation/internal/service"
	"fileStation/pkg/logger"
	"html/template"
	"net/http"
	"time"
)

// AuthHandler обрабатывает запросы, связанные с авторизацией.
type AuthHandler struct {
	authService *service.AuthService
	templates   *template.Template
	version     string
}

// NewAuthHandler создаёт новый экземпляр AuthHandler.
func NewAuthHandler(authService *service.AuthService, templates *template.Template, version string) *AuthHandler {
    return &AuthHandler{
        authService: authService,
        templates:   templates,
        version:     version,
    }
}


// LoginHandler обрабатывает запросы на вход пользователя.
func (h *AuthHandler) LoginHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method == http.MethodPost {
        // Получение данных формы
        username := r.FormValue("username")
        password := r.FormValue("password")

        // Аутентификация пользователя
        err := h.authService.Authenticate(username, password)
        if (err != nil) {
            logger.Infof("Login failed for user: %s", username)
            w.WriteHeader(http.StatusUnauthorized)
            w.Write([]byte("Invalid username or password"))
            return
        }

        // Создание сессии
        token, expires := h.authService.CreateSession(username)

        // Установка cookie с токеном сессии
        http.SetCookie(w, &http.Cookie{
            Name:     "session_token",
            Value:    token,
            Path:     "/",
            Expires:  expires,
            HttpOnly: true,
            SameSite: http.SameSiteLaxMode, // Set the SameSite attribute
        })

        logger.Infof("Login successful for user: %s", username)
        w.WriteHeader(http.StatusOK)
        return
    }

    http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// LogoutHandler обрабатывает запросы на выход пользователя.
func (h *AuthHandler) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	// Получение токена из cookie
	cookie, err := r.Cookie("session_token")
	if (err != nil) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Инвалидация сессии
	h.authService.InvalidateSession(cookie.Value)

	// Удаление cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Path:     "/",
		Expires:  time.Now().Add(-time.Hour),
		HttpOnly: true,
	})

	// Перенаправление на главную страницу
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// Middleware проверяет, авторизован ли пользователь, и добавляет имя пользователя в запрос.
func (h *AuthHandler) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_token")
		if err != nil || !h.authService.IsValidSession(cookie.Value) {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
			return
		}

		// Добавление имени пользователя в заголовки запроса
		username, err := h.authService.GetSessionUsername(cookie.Value)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
			return
		}
		r.Header.Set("X-User", username)

		next.ServeHTTP(w, r)
	})
}

// CheckSessionHandler проверяет, авторизован ли пользователь.
func (h *AuthHandler) CheckSessionHandler(w http.ResponseWriter, r *http.Request) {
    cookie, err := r.Cookie("session_token")
    if err != nil {
        w.WriteHeader(http.StatusUnauthorized)
        json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
        return
    }
    username, err := h.authService.GetSessionUsername(cookie.Value)
    if err != nil {
        w.WriteHeader(http.StatusUnauthorized)
        json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
        return
    }
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{"username": username})
}

// renderTemplate - вспомогательная функция для рендеринга шаблонов.
func (h *AuthHandler) renderTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	err := h.templates.ExecuteTemplate(w, tmpl, data)
	if err != nil {
		logger.Errorf("Error rendering template index.html: %v", err)
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
	}
}
