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
		Username string `json:"username"`
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
	participant, err := tasks.JoinEvent(httpapi.EventSlugParam(r), userID, request.Username)
	if err != nil {
		httpapi.WriteJSON(w, http.StatusBadRequest, httpapi.ErrorResponse{Message: err.Error()})
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, participant)
}
