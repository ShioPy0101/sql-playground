package httpapi

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ShioPy0101/sql-playground/pkg/service"
)

const UserCookieName = "sql_playground_user_id"
const AdminBasicAuthRealm = "SQL Playground Admin"

type ErrorResponse struct {
	Message string `json:"message"`
}

var (
	taskService     *service.TaskService
	taskServiceErr  error
	taskServiceOnce sync.Once
)

func TaskService() (*service.TaskService, error) {
	taskServiceOnce.Do(func() {
		taskService, taskServiceErr = service.NewTaskService(service.NewSQLiteService())
	})
	return taskService, taskServiceErr
}

func EnsureUserID(w http.ResponseWriter, r *http.Request) (string, error) {
	if cookie, err := r.Cookie(UserCookieName); err == nil && cookie.Value != "" {
		return cookie.Value, nil
	}

	userID, err := newUserID()
	if err != nil {
		return "", err
	}

	http.SetCookie(w, &http.Cookie{
		Name:     UserCookieName,
		Value:    userID,
		Path:     "/",
		Expires:  time.Now().AddDate(1, 0, 0),
		MaxAge:   60 * 60 * 24 * 365,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	return userID, nil
}

func NumberParam(r *http.Request) string {
	if number := r.URL.Query().Get("number"); number != "" {
		return number
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/tasks/")
	path = strings.TrimSuffix(path, "/submit")
	return strings.Trim(path, "/")
}

func RequireMethod(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method == method {
		return true
	}

	w.Header().Set("Allow", method)
	WriteJSON(w, http.StatusMethodNotAllowed, ErrorResponse{
		Message: "method not allowed",
	})
	return false
}

func RequireAdminBasicAuth(w http.ResponseWriter, r *http.Request) bool {
	expectedUsername := os.Getenv("ADMIN_USERNAME")
	expectedPassword := os.Getenv("ADMIN_PASSWORD")
	if expectedUsername == "" || expectedPassword == "" {
		WriteJSON(w, http.StatusInternalServerError, ErrorResponse{
			Message: "admin credentials are not configured",
		})
		return false
	}

	username, password, ok := r.BasicAuth()
	if !ok || subtle.ConstantTimeCompare([]byte(username), []byte(expectedUsername)) != 1 || subtle.ConstantTimeCompare([]byte(password), []byte(expectedPassword)) != 1 {
		w.Header().Set("WWW-Authenticate", `Basic realm="`+AdminBasicAuthRealm+`", charset="UTF-8"`)
		WriteJSON(w, http.StatusUnauthorized, ErrorResponse{
			Message: "authentication required",
		})
		return false
	}

	return true
}

func WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func newUserID() (string, error) {
	bytes := make([]byte, 12)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "user_" + hex.EncodeToString(bytes), nil
}
