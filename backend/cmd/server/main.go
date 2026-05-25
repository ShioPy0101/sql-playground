package main

import (
	"net/http"

	// "github.com/ShioPy0101/sql-playground/internal/pandasHandler"

	"github.com/ShioPy0101/sql-playground/internal/handler/sqliteHandler"
	"github.com/ShioPy0101/sql-playground/internal/service"
	"github.com/labstack/echo/v4"
)

func main() {
	e := echo.New()

	api := e.Group("/api")

	sqliteAPI := api.Group("/sqlite")
	// pandasAPI := api.Group("/pandas")

	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Hello, Echo!")
	})

	sqliteService := service.NewSQLiteService()

	sqliteHandlerInstance := sqliteHandler.NewSQLiteHandler(sqliteService)

	sqliteAPI.POST("/execute", sqliteHandlerInstance.Execute)
	// pandasAPI.GET("/examples", pandasHandler.ListExamples)

	e.Logger.Fatal(e.Start(":8080"))
}
