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
	router.POST("/projects/:id/duplicate", handlers.DuplicateProject)
	router.GET("/projects/:id/export", handlers.ExportProject)
	router.POST("/projects/import", handlers.ImportProject)
	router.PUT("/projects/:id", handlers.UpdateProject)
	router.DELETE("/projects/:id", handlers.DeleteProject)

	// Jobs
	router.GET("/projects/:id/jobs", handlers.ListJobs)
	router.POST("/projects/:id/jobs", handlers.AddJob)
	router.PUT("/projects/:id/jobs/:jobId", handlers.UpdateJob)
	router.DELETE("/projects/:id/jobs/:jobId", handlers.DeleteJob)
	router.POST("/projects/:id/jobs/:jobId/resume", handlers.ResumeJob)
	router.POST("/jobs/validate", handlers.ValidateJobHandler)

	// Variáveis
	router.GET("/projects/:id/variables", handlers.ListVariables)
	router.POST("/projects/:id/variables", handlers.CreateVariable)
	router.GET("/projects/:id/variables/:variableName", handlers.GetVariable)
	router.PUT("/projects/:id/variables/:variableName", handlers.UpdateVariable)
	router.DELETE("/projects/:id/variables/:variableName", handlers.DeleteVariable)

	// Elementos visuais
	router.GET("/projects/:id/visual-elements", handlers.ListVisualElements)
	router.POST("/projects/:id/visual-elements", handlers.CreateVisualElement)
	router.GET("/projects/:id/visual-elements/:elementId", handlers.GetVisualElement)
	router.PUT("/projects/:id/visual-elements/:elementId", handlers.UpdateVisualElement)
	router.DELETE("/projects/:id/visual-elements/:elementId", handlers.DeleteVisualElement)

	// Executar projeto
	router.POST("/projects/:id/run", handlers.RunProject)
	router.POST("/projects/:id/stop", handlers.StopProject)

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

	// Progresso de counts via WebSocket
	router.GET("/ws/counts", func(c *gin.Context) {
		status.CountStatusWS(c.Writer, c.Request)
	})

	// Uso de workers via WebSocket
	router.GET("/ws/workers", func(c *gin.Context) {
		status.WorkerStatusWS(c.Writer, c.Request)
	})

	api := router.Group("/api")
	{
		// Benchmarks
		api.POST("/projects/:id/benchmarks/run", handlers.RunBenchmark)
		api.GET("/projects/:id/benchmarks", handlers.ListBenchmarks)
		api.GET("/projects/:id/benchmarks/:runId", handlers.GetBenchmark)
		api.GET("/projects/:id/benchmarks/report", logger.BenchmarkHistoryReportHandler)
		api.GET("/projects/:id/benchmarks/:runId/report", logger.BenchmarkReportHandler)

		// Download do PDF
		api.GET("/pipeline/:pipelineId/report", logger.PipelineReportHandler)

		// Visualização inline do PDF
		api.GET("/pipeline/:pipelineId/report/preview", logger.PipelineReportInlineHandler)

		// Lista de pipelines disponíveis
		api.GET("/pipelines/reports", logger.ListPipelineReportsHandler)
		api.GET("/projects/:id/pipelines/reports", func(c *gin.Context) {
			q := c.Request.URL.Query()
			q.Set("projectId", c.Param("id"))
			c.Request.URL.RawQuery = q.Encode()
			logger.ListPipelineReportsHandler(c)
		})

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

		// Resumo de erros de um pipeline específico
		api.GET("/pipeline/:pipelineId/errors", func(c *gin.Context) {
			pipelineID := c.Param("pipelineId")
			errorSummary, err := logger.GetErrorSummary(pipelineID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{
					"error": fmt.Sprintf("Pipeline não encontrado: %v", err),
				})
				return
			}
			c.JSON(http.StatusOK, errorSummary)
		})

		// Log completo de um pipeline específico
		api.GET("/pipeline/:pipelineId/log", func(c *gin.Context) {
			pipelineID := c.Param("pipelineId")
			log, err := logger.LoadPipelineLog(pipelineID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{
					"error": fmt.Sprintf("Pipeline não encontrado: %v", err),
				})
				return
			}
			c.JSON(http.StatusOK, log)
		})
	}

	router.Run(":8080")
}
