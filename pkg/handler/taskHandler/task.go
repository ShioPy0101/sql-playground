package taskHandler

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/ShioPy0101/sql-playground/pkg/httpapi"
	"github.com/ShioPy0101/sql-playground/pkg/service"
	"github.com/labstack/echo/v4"
)

type SubmitRequest struct {
	Query string `json:"query"`
}

type TaskHandler struct {
	service *service.TaskService
}

func NewTaskHandler(service *service.TaskService) *TaskHandler {
	return &TaskHandler{service: service}
}

func (h *TaskHandler) Get(c echo.Context) error {
	userID, err := ensureUserID(c)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"message": "failed to identify user",
		})
	}

	task, err := h.service.GetPublicTaskForUser(c.Param("number"), userID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"message": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, task)
}

func (h *TaskHandler) List(c echo.Context) error {
	userID, err := ensureUserID(c)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"message": "failed to identify user",
		})
	}

	tasks, err := h.service.ListPublicTasksForUser(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"message": "failed to load tasks",
		})
	}

	return c.JSON(http.StatusOK, map[string][]service.PublicTaskSummary{
		"tasks": tasks,
	})
}

func (h *TaskHandler) Submit(c echo.Context) error {
	var req SubmitRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"message": "invalid request body",
		})
	}

	userID, err := ensureUserID(c)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"message": "failed to identify user",
		})
	}

	result, err := h.service.SubmitForUser(c.Param("number"), req.Query, userID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"message": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, result)
}

func (h *TaskHandler) History(c echo.Context) error {
	userID, err := ensureUserID(c)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"message": "failed to identify user",
		})
	}

	submissions, err := h.service.SubmissionsForUserTask(c.Param("number"), userID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"message": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string][]service.TaskSubmission{
		"submissions": submissions,
	})
}

func (h *TaskHandler) CurrentUser(c echo.Context) error {
	userID, err := ensureUserID(c)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"message": "failed to identify user",
		})
	}

	return c.JSON(http.StatusOK, map[string]string{"userId": userID})
}

func (h *TaskHandler) AdminSubmissions(c echo.Context) error {
	if !httpapi.RequireAdminBasicAuth(c.Response().Writer, c.Request()) {
		return nil
	}

	submissions, err := h.service.Submissions()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"message": "failed to load submissions",
		})
	}

	return c.JSON(http.StatusOK, map[string][]service.TaskSubmission{
		"submissions": submissions,
	})
}

func (h *TaskHandler) AdminCheckSolutions(c echo.Context) error {
	if !httpapi.RequireAdminBasicAuth(c.Response().Writer, c.Request()) {
		return nil
	}

	checks, err := h.service.CheckAllSolutions()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"message": "failed to check solutions",
		})
	}

	return c.JSON(http.StatusOK, map[string][]service.TaskSolutionCheck{
		"checks": checks,
	})
}

const userCookieName = "sql_playground_user_id"

func ensureUserID(c echo.Context) (string, error) {
	if cookie, err := c.Cookie(userCookieName); err == nil && cookie.Value != "" {
		return cookie.Value, nil
	}

	userID, err := newUserID()
	if err != nil {
		return "", err
	}

	c.SetCookie(&http.Cookie{
		Name:     userCookieName,
		Value:    userID,
		Path:     "/",
		Expires:  time.Now().AddDate(1, 0, 0),
		MaxAge:   60 * 60 * 24 * 365,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	return userID, nil
}

func newUserID() (string, error) {
	bytes := make([]byte, 12)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "user_" + hex.EncodeToString(bytes), nil
}
