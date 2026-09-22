package handler

import (
	"encoding/json"
	"net/http"

	"github.com/ShioPy0101/sql-playground/pkg/service"
)

type executeRequest struct {
	CSV       string `json:"csv"`
	Input     string `json:"input"`
	InputType string `json:"inputType"`
	Query     string `json:"query"`
	CheckSQL  string `json:"checkSql"`
}

type executeResponse struct {
	CSV string `json:"csv"`
}

type errorResponse struct {
	Message string `json:"message"`
}

var sqliteService = service.NewSQLiteService()

func Handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{
			Message: "method not allowed",
		})
		return
	}

	var req executeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Message: "invalid request body",
		})
		return
	}

	var resultCSV string
	var err error
	input := req.Input
	inputType := req.InputType
	if inputType == "" {
		input = req.CSV
		inputType = "csv"
	}
	if req.CheckSQL != "" {
		resultCSV, err = sqliteService.ExecuteInputAndInspect(input, inputType, req.Query, req.CheckSQL)
	} else {
		resultCSV, err = sqliteService.ExecuteInput(input, inputType, req.Query)
	}
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Message: err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, executeResponse{
		CSV: resultCSV,
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
