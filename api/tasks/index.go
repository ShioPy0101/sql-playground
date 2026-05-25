package handler

import (
	"net/http"

	"github.com/ShioPy0101/sql-playground/internal/httpapi"
	"github.com/ShioPy0101/sql-playground/internal/service"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	if !httpapi.RequireMethod(w, r, http.MethodGet) {
		return
	}

	taskService, err := httpapi.TaskService()
	if err != nil {
		httpapi.WriteJSON(w, http.StatusInternalServerError, httpapi.ErrorResponse{
			Message: "failed to initialize tasks",
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

	tasks, err := taskService.ListPublicTasksForUser(userID)
	if err != nil {
		httpapi.WriteJSON(w, http.StatusInternalServerError, httpapi.ErrorResponse{
			Message: "failed to load tasks",
		})
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, map[string][]service.PublicTaskSummary{
		"tasks": tasks,
	})
}
