package main

import (
	"etl/handlers"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()

	// Projetos
	router.GET("/projects", handlers.ListProjects)
	router.POST("/projects", handlers.CreateProject)
	router.GET("/projects/:id", handlers.GetProjectByID)
	router.POST("/projects/:id/close", handlers.CloseProject)
	router.PUT("/projects/:id", handlers.UpdateProject)
	router.DELETE("/projects/:id", handlers.DeleteProject)

	// Jobs
	router.GET("/projects/:id/jobs", handlers.ListJobs)
	router.POST("/projects/:id/jobs", handlers.AddJob)
	router.PUT("/projects/:id/jobs/:jobId", handlers.UpdateJob)
	router.DELETE("/projects/:id/jobs/:jobId", handlers.DeleteJob)

	router.Run(":8080")
}
