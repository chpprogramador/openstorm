package main

import (
	"etl/handlers"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()

	// Projetos
	router.POST("/projects", handlers.CreateProject)
	router.GET("/projects/:name", handlers.OpenProject)
	router.POST("/projects/:name/close", handlers.CloseProject)

	// Jobs
	router.GET("/projects/:name/jobs", handlers.ListJobs)
	router.POST("/projects/:name/jobs", handlers.AddJob)
	router.PUT("/projects/:name/jobs/:jobName", handlers.UpdateJob)
	router.DELETE("/projects/:name/jobs/:jobName", handlers.DeleteJob)

	router.Run(":8080")
}
