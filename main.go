package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"appstats/internal/config"
	"appstats/internal/handlers"
	"appstats/internal/models"
)

func main() {
	cfg := config.Load()

	db, err := gorm.Open(mysql.Open(cfg.DSN), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect db: %v", err)
	}

	if err := db.AutoMigrate(&models.User{}, &models.UserEvent{}); err != nil {
		log.Fatalf("auto migrate failed: %v", err)
	}

	r := gin.Default()

	// Write-only API for event reporting.
	api := r.Group("/api")
	{
		api.POST("/events/report", handlers.ReportEventHandler(db))
	}

	// Admin dashboard: server-side query + chart rendering in browser.
	r.GET("/admin", handlers.AdminPageHandler(db))

	log.Println("server listening on :8080")
	if err := r.Run(":8080"); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}