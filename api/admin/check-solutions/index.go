package handler

import (
	"net/http"

	"github.com/ShioPy0101/sql-playground/pkg/httpapi"
	"github.com/ShioPy0101/sql-playground/pkg/service"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	if !httpapi.RequireMethod(w, r, http.MethodPost) {
		return
	}
	if !httpapi.RequireAdminBasicAuth(w, r) {
		return
	}

	taskService, err := httpapi.TaskService()
	if err != nil {
		httpapi.WriteJSON(w, http.StatusInternalServerError, httpapi.ErrorResponse{
			Message: "failed to initialize tasks",
		})
		return
	}

	checks, err := taskService.CheckAllSolutions()
	if err != nil {
		httpapi.WriteJSON(w, http.StatusInternalServerError, httpapi.ErrorResponse{
			Message: "failed to check solutions",
		})
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, map[string][]service.TaskSolutionCheck{
		"checks": checks,
	})
}
