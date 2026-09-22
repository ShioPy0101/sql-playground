package sqliteHandler

import (
	"net/http"

	"github.com/ShioPy0101/sql-playground/pkg/service"
	"github.com/labstack/echo/v4"
)

type ExecuteRequest struct {
	CSV       string `json:"csv"`
	Input     string `json:"input"`
	InputType string `json:"inputType"`
	Query     string `json:"query"`
	CheckSQL  string `json:"checkSql"`
}

type ExecuteResponse struct {
	CSV     string                   `json:"csv"`
	Metrics service.ExecutionMetrics `json:"metrics"`
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

	input := req.Input
	inputType := req.InputType
	if inputType == "" {
		input = req.CSV
		inputType = "csv"
	}

	var result service.StatementExecutionResult
	var err error
	if req.CheckSQL != "" {
		result, err = h.service.ExecuteInputAndInspectWithStats(input, inputType, req.Query, req.CheckSQL)
	} else {
		result, err = h.service.ExecuteInputWithStats(input, inputType, req.Query)
	}
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"message": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, ExecuteResponse{
		CSV:     result.CSV,
		Metrics: result.Metrics,
	})
}
