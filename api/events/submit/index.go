package handler

import (
	"encoding/json"
	"net/http"

	"github.com/ShioPy0101/sql-playground/pkg/httpapi"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	if !httpapi.RequireMethod(w, r, http.MethodPost) {
		return
	}
	var request struct {
		Query string `json:"query"`
	}
	if json.NewDecoder(r.Body).Decode(&request) != nil {
		httpapi.WriteJSON(w, http.StatusBadRequest, httpapi.ErrorResponse{Message: "invalid request body"})
		return
	}
	userID, err := httpapi.EnsureUserID(w, r)
	if err != nil {
		httpapi.WriteJSON(w, http.StatusInternalServerError, httpapi.ErrorResponse{Message: "failed to identify user"})
		return
	}
	tasks, err := httpapi.TaskService()
	if err != nil {
		httpapi.WriteJSON(w, http.StatusInternalServerError, httpapi.ErrorResponse{Message: "failed to initialize tasks"})
		return
	}
	result, err := tasks.SubmitEventTask(httpapi.EventSlugParam(r), httpapi.NumberParam(r), request.Query, userID)
	if err != nil {
		httpapi.WriteJSON(w, http.StatusBadRequest, httpapi.ErrorResponse{Message: err.Error()})
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, result)
}
