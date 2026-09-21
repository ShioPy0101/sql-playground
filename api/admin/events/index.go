package handler

import (
	"encoding/json"
	"net/http"

	"github.com/ShioPy0101/sql-playground/pkg/httpapi"
	"github.com/ShioPy0101/sql-playground/pkg/service"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	if !httpapi.RequireAdminBasicAuth(w, r) {
		return
	}
	tasks, err := httpapi.TaskService()
	if err != nil {
		httpapi.WriteJSON(w, http.StatusInternalServerError, httpapi.ErrorResponse{Message: "failed to initialize tasks"})
		return
	}
	switch r.Method {
	case http.MethodGet:
		events, err := tasks.Events()
		if err != nil {
			httpapi.WriteJSON(w, http.StatusInternalServerError, httpapi.ErrorResponse{Message: "failed to load events"})
			return
		}
		httpapi.WriteJSON(w, http.StatusOK, map[string][]service.EventSummary{"events": events})
	case http.MethodPost:
		var input service.CreateEventInput
		if json.NewDecoder(r.Body).Decode(&input) != nil {
			httpapi.WriteJSON(w, http.StatusBadRequest, httpapi.ErrorResponse{Message: "invalid request body"})
			return
		}
		event, err := tasks.CreateEvent(input)
		if err != nil {
			httpapi.WriteJSON(w, http.StatusBadRequest, httpapi.ErrorResponse{Message: err.Error()})
			return
		}
		httpapi.WriteJSON(w, http.StatusCreated, event)
	default:
		w.Header().Set("Allow", "GET, POST")
		httpapi.WriteJSON(w, http.StatusMethodNotAllowed, httpapi.ErrorResponse{Message: "method not allowed"})
	}
}
