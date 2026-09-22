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
	slug := httpapi.EventSlugParam(r)
	switch r.Method {
	case http.MethodGet:
		detail, err := tasks.AdminEvent(slug)
		if err != nil {
			httpapi.WriteJSON(w, http.StatusNotFound, httpapi.ErrorResponse{Message: err.Error()})
			return
		}
		httpapi.WriteJSON(w, http.StatusOK, detail)
	case http.MethodPut:
		var input service.UpdateEventTasksInput
		if json.NewDecoder(r.Body).Decode(&input) != nil {
			httpapi.WriteJSON(w, http.StatusBadRequest, httpapi.ErrorResponse{Message: "invalid request body"})
			return
		}
		detail, err := tasks.UpdateEventTasks(slug, input.TaskNumbers)
		if err != nil {
			httpapi.WriteJSON(w, http.StatusBadRequest, httpapi.ErrorResponse{Message: err.Error()})
			return
		}
		httpapi.WriteJSON(w, http.StatusOK, detail)
	default:
		w.Header().Set("Allow", "GET, PUT")
		httpapi.WriteJSON(w, http.StatusMethodNotAllowed, httpapi.ErrorResponse{Message: "method not allowed"})
	}
}
