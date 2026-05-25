package handler

import (
	"encoding/json"
	"net/http"

	"github.com/ShioPy0101/sql-playground/internal/httpapi"
)

type submitRequest struct {
	Query string `json:"query"`
}

func Handler(w http.ResponseWriter, r *http.Request) {
	if !httpapi.RequireMethod(w, r, http.MethodPost) {
		return
	}

	taskService, err := httpapi.TaskService()
	if err != nil {
		httpapi.WriteJSON(w, http.StatusInternalServerError, httpapi.ErrorResponse{
			Message: "failed to initialize tasks",
		})
		return
	}

	var req submitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpapi.WriteJSON(w, http.StatusBadRequest, httpapi.ErrorResponse{
			Message: "invalid request body",
		})
		return
	}

	userID, err := httpapi.EnsureUserID(w, r)
	if err != nil {
		httpapi.WriteJSON(w, http.StatusInternalServerError, httpapi.ErrorResponse{
			Message: "failed to identify user",
		})
		return
	}

	result, err := taskService.SubmitForUser(httpapi.NumberParam(r), req.Query, userID)
	if err != nil {
		httpapi.WriteJSON(w, http.StatusNotFound, httpapi.ErrorResponse{
			Message: err.Error(),
		})
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, result)
}
