package main

import (
	"etl/handlers"
	"etl/status"

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

	// Executar projeto
	router.POST("/projects/:id/run", handlers.RunProject)

	// Status de jobs via WebSocket
	router.GET("/ws/status", func(c *gin.Context) {
		status.JobStatusWS(c.Writer, c.Request)
	})

	router.Run(":8080")
}
