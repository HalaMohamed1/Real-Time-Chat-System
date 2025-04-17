package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {

	InitDB()

	router := gin.Default()

	// Enable CORS
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	router.POST("/signup", SignUp)
	router.POST("/login", Login)

	// API routes
	router.GET("/api/hello", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Hello from API Gateway!",
		})
	})

	router.GET("/auth/google", handleGoogleLogin)
	router.GET("/oauth2/callback", handleGoogleCallback)
	router.GET("/auth/decode", func(c *gin.Context) {
		tokenString := c.Query("token")

		claims, err := ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"claims": claims,
		})
	})

	// Start server
	log.Println("Server starting on :8080")
	router.Run(":8080")

}
