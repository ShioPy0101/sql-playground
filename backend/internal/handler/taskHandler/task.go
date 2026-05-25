package taskHandler

import (
	"net/http"

	"github.com/ShioPy0101/sql-playground/internal/service"
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
	task, err := h.service.GetPublicTask(c.Param("number"))
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"message": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, task)
}

func (h *TaskHandler) Submit(c echo.Context) error {
	var req SubmitRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"message": "invalid request body",
		})
	}

	result, err := h.service.Submit(c.Param("number"), req.Query)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"message": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, result)
}
