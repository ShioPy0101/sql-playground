package main

import (
	"net/http"
	"os"

	// "github.com/ShioPy0101/sql-playground/internal/pandasHandler"

	"github.com/ShioPy0101/sql-playground/internal/handler/sqliteHandler"
	"github.com/ShioPy0101/sql-playground/internal/handler/taskHandler"
	"github.com/ShioPy0101/sql-playground/internal/service"
	"github.com/labstack/echo/v4"
	_ "modernc.org/sqlite"
)

func main() {
	e := echo.New()

	api := e.Group("/api")

	sqliteAPI := api.Group("/sqlite")
	taskAPI := api.Group("/tasks")
	adminAPI := api.Group("/admin")
	// pandasAPI := api.Group("/pandas")

	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Hello, Echo!")
	})

	sqliteService := service.NewSQLiteService()
	taskService, err := service.NewTaskService(sqliteService)
	if err != nil {
		e.Logger.Fatal(err)
	}

	sqliteHandlerInstance := sqliteHandler.NewSQLiteHandler(sqliteService)
	taskHandlerInstance := taskHandler.NewTaskHandler(taskService)

	sqliteAPI.POST("/execute", sqliteHandlerInstance.Execute)
	api.GET("/me", taskHandlerInstance.CurrentUser)
	taskAPI.GET("", taskHandlerInstance.List)
	taskAPI.GET("/:number", taskHandlerInstance.Get)
	taskAPI.POST("/:number/submit", taskHandlerInstance.Submit)
	adminAPI.GET("/submissions", taskHandlerInstance.AdminSubmissions)
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
