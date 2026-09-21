package handler

import (
	"net/http"

	"github.com/ShioPy0101/sql-playground/pkg/httpapi"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	if !httpapi.RequireMethod(w, r, http.MethodGet) || !httpapi.RequireAdminBasicAuth(w, r) {
		return
	}
	tasks, err := httpapi.TaskService()
	if err != nil {
		httpapi.WriteJSON(w, http.StatusInternalServerError, httpapi.ErrorResponse{Message: "failed to initialize tasks"})
		return
	}
	detail, err := tasks.AdminEvent(httpapi.EventSlugParam(r))
	if err != nil {
		httpapi.WriteJSON(w, http.StatusNotFound, httpapi.ErrorResponse{Message: err.Error()})
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, detail)
}
