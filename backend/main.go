package main

import (
	"etl/handlers"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()

	// Projetos
	router.POST("/projects", handlers.CreateProject)
	router.GET("/projects/:id", handlers.GetProjectByID)
	router.POST("/projects/:id/close", handlers.CloseProject)

	// Jobs
	router.GET("/projects/:id/jobs", handlers.ListJobs)
	router.POST("/projects/:id/jobs", handlers.AddJob)
	router.PUT("/projects/:id/jobs/:jobId", handlers.UpdateJob)
	router.DELETE("/projects/:id/jobs/:jobId", handlers.DeleteJob)

	router.Run(":8080")
}
