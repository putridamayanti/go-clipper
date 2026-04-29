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

	analyzer, err := analyzer2.NewAnalyzer(ctx, cfg.GeminiAPIKey)
	if err != nil {
		log.Fatal(err)
	}

	captioner, err := analyzer2.NewCaptioner(ctx, cfg.GeminiAPIKeyForCaption)
	if err != nil {
		log.Fatalf("Error initializing captioner: %v", err)
	}

	clipperController := controllers.NewClipperController(*analyzer, *captioner)

	api := r.Group("/api/v1")
	{
		api.POST("/clipper", clipperController.Create)
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
