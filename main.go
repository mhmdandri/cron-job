package main

import (
	"log"
	"net/http"

	"cron-job/config"
	"cron-job/handlers"
	"cron-job/services"

	"github.com/gin-gonic/gin"
)

func main() {
	// Load environment configuration
	cfg := config.LoadConfig()

	log.Printf("==================================================")
	log.Printf("Starting Shipment Cron Monitor Service")
	log.Printf("Target API : %s", cfg.TargetURL)
	log.Printf("Schedule   : %s", cfg.CronSchedule)
	log.Printf("Port       : %s", cfg.Port)
	log.Printf("==================================================")

	// Initialize Services
	cronService := services.NewCronService(cfg)

	// Initialize Scheduler
	scheduler, err := services.NewSchedulerManager(cronService, cfg.CronSchedule)
	if err != nil {
		log.Fatalf("Failed to initialize scheduler: %v", err)
	}

	// Trigger initial hit on startup asynchronously
	go func() {
		log.Println("[STARTUP] Executing initial health check hit...")
		cronService.Execute("startup")
	}()

	// Setup Gin Router
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// Serve Static UI
	r.StaticFile("/", "./static/index.html")
	r.Static("/static", "./static")

	// API Routes
	apiHandler := handlers.NewAPIHandler(cfg, cronService, scheduler)

	r.GET("/health", apiHandler.HealthCheck)
	r.GET("/api/status", apiHandler.GetStatus)
	r.GET("/api/logs", apiHandler.GetLogs)
	r.POST("/api/trigger", apiHandler.TriggerManual)
	r.POST("/api/schedule", apiHandler.UpdateSchedule)

	// Fallback route to UI
	r.NoRoute(func(c *gin.Context) {
		c.File("./static/index.html")
	})

	serverAddr := ":" + cfg.Port
	log.Printf("Server listening on http://localhost%s", serverAddr)

	if err := r.Run(serverAddr); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Failed to start Gin server: %v", err)
	}
}
