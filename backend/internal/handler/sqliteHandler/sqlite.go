package sqliteHandler

import (
	"net/http"

	"github.com/ShioPy0101/sql-playground/internal/service"
	"github.com/labstack/echo/v4"
)

type ExecuteRequest struct {
	CSV   string `json:"csv"`
	Query string `json:"query"`
}

type ExecuteResponse struct {
	CSV string `json:"csv"`
}

type SQLiteHandler struct {
	service *service.SQLiteService
}

func NewSQLiteHandler(service *service.SQLiteService) *SQLiteHandler {
	return &SQLiteHandler{
		service: service,
	}
}

func (h *SQLiteHandler) Execute(c echo.Context) error {
	var req ExecuteRequest

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"message": "invalid request body",
		})
	}

	resultCSV, err := h.service.Execute(req.CSV, req.Query)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"message": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, ExecuteResponse{
		CSV: resultCSV,
	})
}
