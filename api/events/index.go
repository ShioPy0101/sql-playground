package handler

import (
	"encoding/json"
	"net/http"

	"github.com/ShioPy0101/sql-playground/pkg/httpapi"
	"github.com/ShioPy0101/sql-playground/pkg/service"
)

func Handler(w http.ResponseWriter, r *http.Request) {
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

	switch r.URL.Query().Get("action") {
	case "show":
		showEvent(w, r, tasks, userID)
	case "join":
		joinEvent(w, r, tasks, userID)
	case "task":
		showEventTask(w, r, tasks, userID)
	case "submit":
		submitEventTask(w, r, tasks, userID)
	case "history":
		eventTaskHistory(w, r, tasks, userID)
	default:
		httpapi.WriteJSON(w, http.StatusNotFound, httpapi.ErrorResponse{Message: "event endpoint was not found"})
	}
}

func showEvent(w http.ResponseWriter, r *http.Request, tasks *service.TaskService, userID string) {
	if !httpapi.RequireMethod(w, r, http.MethodGet) {
		return
	}
	page, err := tasks.EventPage(httpapi.EventSlugParam(r), userID)
	if err != nil {
		httpapi.WriteJSON(w, http.StatusNotFound, httpapi.ErrorResponse{Message: err.Error()})
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, page)
}

func joinEvent(w http.ResponseWriter, r *http.Request, tasks *service.TaskService, userID string) {
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
	participant, err := tasks.JoinEvent(httpapi.EventSlugParam(r), userID, request.Username)
	if err != nil {
		httpapi.WriteJSON(w, http.StatusBadRequest, httpapi.ErrorResponse{Message: err.Error()})
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, participant)
}

func showEventTask(w http.ResponseWriter, r *http.Request, tasks *service.TaskService, userID string) {
	if !httpapi.RequireMethod(w, r, http.MethodGet) {
		return
	}
	task, err := tasks.GetEventTask(httpapi.EventSlugParam(r), httpapi.NumberParam(r), userID)
	if err != nil {
		httpapi.WriteJSON(w, http.StatusNotFound, httpapi.ErrorResponse{Message: err.Error()})
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, task)
}

func submitEventTask(w http.ResponseWriter, r *http.Request, tasks *service.TaskService, userID string) {
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
	result, err := tasks.SubmitEventTask(httpapi.EventSlugParam(r), httpapi.NumberParam(r), request.Query, userID)
	if err != nil {
		httpapi.WriteJSON(w, http.StatusBadRequest, httpapi.ErrorResponse{Message: err.Error()})
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, result)
}

func eventTaskHistory(w http.ResponseWriter, r *http.Request, tasks *service.TaskService, userID string) {
	if !httpapi.RequireMethod(w, r, http.MethodGet) {
		return
	}
	submissions, err := tasks.EventTaskSubmissions(httpapi.EventSlugParam(r), httpapi.NumberParam(r), userID)
	if err != nil {
		httpapi.WriteJSON(w, http.StatusBadRequest, httpapi.ErrorResponse{Message: err.Error()})
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string][]service.TaskSubmission{"submissions": submissions})
}
