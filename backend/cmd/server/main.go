package main

import (
	"net/http"
	"os"

	// "github.com/ShioPy0101/sql-playground/internal/pandasHandler"

	"github.com/ShioPy0101/sql-playground/internal/handler/sqliteHandler"
	"github.com/ShioPy0101/sql-playground/internal/handler/taskHandler"
	"github.com/ShioPy0101/sql-playground/internal/service"
	"github.com/labstack/echo/v4"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	e := echo.New()

	api := e.Group("/api")

	sqliteAPI := api.Group("/sqlite")
	taskAPI := api.Group("/tasks")
	// pandasAPI := api.Group("/pandas")

	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Hello, Echo!")
	})

	sqliteService := service.NewSQLiteService()
	taskService := service.NewTaskService(sqliteService)

	sqliteHandlerInstance := sqliteHandler.NewSQLiteHandler(sqliteService)
	taskHandlerInstance := taskHandler.NewTaskHandler(taskService)

	sqliteAPI.POST("/execute", sqliteHandlerInstance.Execute)
	taskAPI.GET("/:number", taskHandlerInstance.Get)
	taskAPI.POST("/:number/submit", taskHandlerInstance.Submit)
	// pandasAPI.GET("/examples", pandasHandler.ListExamples)

	e.Logger.Fatal(e.Start(serverAddress()))
}

func serverAddress() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	return ":" + port
}
