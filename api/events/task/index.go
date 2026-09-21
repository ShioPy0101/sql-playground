package handler

import (
	"net/http"

	"github.com/ShioPy0101/sql-playground/pkg/httpapi"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	if !httpapi.RequireMethod(w, r, http.MethodGet) {
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
	task, err := tasks.GetEventTask(httpapi.EventSlugParam(r), httpapi.NumberParam(r), userID)
	if err != nil {
		httpapi.WriteJSON(w, http.StatusNotFound, httpapi.ErrorResponse{Message: err.Error()})
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, task)
}
