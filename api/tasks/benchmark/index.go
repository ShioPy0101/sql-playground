package handler

import (
	"encoding/json"
	"net/http"

	"github.com/ShioPy0101/sql-playground/pkg/httpapi"
	"github.com/ShioPy0101/sql-playground/pkg/service"
)

type benchmarkRequest struct {
	Query string `json:"query"`
}

func Handler(w http.ResponseWriter, r *http.Request) {
	if !httpapi.RequireMethod(w, r, http.MethodPost) {
		return
	}
	var req benchmarkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpapi.WriteJSON(w, http.StatusBadRequest, httpapi.ErrorResponse{Message: "invalid request body"})
		return
	}
	task, err := service.LoadTaskDefinition(httpapi.NumberParam(r))
	if err != nil {
		httpapi.WriteJSON(w, http.StatusNotFound, httpapi.ErrorResponse{Message: err.Error()})
		return
	}
	if task.Benchmark == nil || !task.Benchmark.Enabled {
		httpapi.WriteJSON(w, http.StatusBadRequest, httpapi.ErrorResponse{Message: "この課題では性能計測が設定されていません"})
		return
	}
	report, err := service.NewBenchmarkService().RunContext(r.Context(), task.Mode, *task.Benchmark, req.Query)
	if err != nil {
		httpapi.WriteJSON(w, http.StatusBadRequest, httpapi.ErrorResponse{Message: err.Error()})
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, report)
}
