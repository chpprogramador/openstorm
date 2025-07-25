package main

import (
	"etl/handlers"
	"etl/logger"
	"etl/status"
	"fmt"
	"net/http"

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
	router.POST("/jobs/validate", handlers.ValidateJobHandler)

	// Executar projeto
	router.POST("/projects/:id/run", handlers.RunProject)

	// Status de jobs via WebSocket
	router.GET("/ws/status", func(c *gin.Context) {
		status.JobStatusWS(c.Writer, c.Request)
	})

	// Status de projeto via WebSocket
	router.GET("/ws/project-status", func(c *gin.Context) {
		status.ProjectStatusWS(c.Writer, c.Request)
	})

	// Logs de jobs via WebSocket
	router.GET("/ws/logs", func(c *gin.Context) {
		status.LogsWS(c.Writer, c.Request)
	})

	api := router.Group("/api")
	{
		// Download do PDF
		api.GET("/pipeline/:pipelineId/report", logger.PipelineReportHandler)

		// Visualização inline do PDF
		api.GET("/pipeline/:pipelineId/report/preview", logger.PipelineReportInlineHandler)

		// Lista de pipelines disponíveis
		api.GET("/pipelines/reports", logger.ListPipelineReportsHandler)

		// Estatísticas de um pipeline específico (JSON)
		api.GET("/pipeline/:pipelineId/stats", func(c *gin.Context) {
			pipelineID := c.Param("pipelineId")
			stats, err := logger.GetPipelineStats(pipelineID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{
					"error": fmt.Sprintf("Pipeline não encontrado: %v", err),
				})
				return
			}
			c.JSON(http.StatusOK, stats)
		})
	}

	router.Run(":8080")
}
