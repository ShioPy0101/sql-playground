package main

import (
	"net/http"
	"os"

	// "github.com/ShioPy0101/sql-playground/pkg/pandasHandler"

	"github.com/ShioPy0101/sql-playground/pkg/handler/sqliteHandler"
	"github.com/ShioPy0101/sql-playground/pkg/handler/taskHandler"
	"github.com/ShioPy0101/sql-playground/pkg/service"
	"github.com/labstack/echo/v4"
	_ "modernc.org/sqlite"
)

func main() {
	e := echo.New()

	api := e.Group("/api")

	sqliteAPI := api.Group("/sqlite")
	taskAPI := api.Group("/tasks")
	adminAPI := api.Group("/admin")
	eventAPI := api.Group("/events")
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
	taskAPI.POST("/:number/benchmark", taskHandlerInstance.Benchmark)
	taskAPI.GET("/:number/history", taskHandlerInstance.History)
	adminAPI.GET("/submissions", taskHandlerInstance.AdminSubmissions)
	adminAPI.POST("/check-solutions", taskHandlerInstance.AdminCheckSolutions)
	adminAPI.GET("/events", taskHandlerInstance.AdminEvents)
	adminAPI.POST("/events", taskHandlerInstance.AdminCreateEvent)
	adminAPI.GET("/events/:slug", taskHandlerInstance.AdminEvent)
	adminAPI.PUT("/events/:slug/tasks", taskHandlerInstance.AdminUpdateEventTasks)
	eventAPI.GET("/:slug", taskHandlerInstance.EventPage)
	eventAPI.POST("/:slug/join", taskHandlerInstance.JoinEvent)
	eventAPI.GET("/:slug/tasks/:number", taskHandlerInstance.GetEventTask)
	eventAPI.POST("/:slug/tasks/:number/submit", taskHandlerInstance.SubmitEventTask)
	eventAPI.GET("/:slug/tasks/:number/history", taskHandlerInstance.EventTaskHistory)
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
