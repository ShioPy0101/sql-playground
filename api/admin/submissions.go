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

	submissions, err := taskService.Submissions()
	if err != nil {
		httpapi.WriteJSON(w, http.StatusInternalServerError, httpapi.ErrorResponse{
			Message: "failed to load submissions",
		})
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, map[string][]service.TaskSubmission{
		"submissions": submissions,
	})
}
