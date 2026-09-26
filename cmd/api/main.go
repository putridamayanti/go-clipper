package main

import (
	"context"
	analyzer2 "go-clipper/internal/analyzer"
	"go-clipper/internal/config"
	"go-clipper/internal/controllers"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	cfg := config.LoadConfig()
	ctx := context.Background()

	analyzer, captioner, err := analyzer2.NewFromConfig(ctx, cfg)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Using AI provider: %s", cfg.Provider)

	clipperController := controllers.NewClipperController(analyzer, captioner)

	api := r.Group("/api/v1")
	{
		api.POST("/clipper", clipperController.Create)
		api.POST("/download", clipperController.Download)
		api.POST("/download/instagram", clipperController.DownloadInstagram)
		api.POST("/analyze", clipperController.Analyze)
		api.POST("/generate-description", clipperController.GenerateDescription)
		api.POST("/clipper/captions", clipperController.GenerateCaption)
	}

	// Define a simple GET endpoint
	r.GET("/ping", func(c *gin.Context) {
		// Return JSON response
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	err = r.Run(":8000")
	if err != nil {
		return
	}
}
