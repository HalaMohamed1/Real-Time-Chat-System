package main

import (
	"log"
	"net/http"
	"time" //for cache

	"github.com/gin-gonic/gin"
)

func main() {
	InitDB()
	InitCache() // initialize Redis cache

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
		if tokenString == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Token missing"})
			return
		}

		// Check cache first
		var cachedClaims map[string]interface{}
		cacheKey := "jwt:" + tokenString
		if err := GetCache(cacheKey, &cachedClaims); err == nil {
			c.JSON(http.StatusOK, gin.H{
				"claims": cachedClaims,
				"cached": true,
			})
			return
		}

		// Not in cache - validate properly
		claims, err := ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		// Cache valid results (for 5 minutes)
		if err := SetCache(cacheKey, claims, 5*time.Minute); err != nil {
			log.Printf("Failed to cache JWT: %v", err)
		}

		c.JSON(http.StatusOK, gin.H{
			"claims": claims,
			"cached": false,
		})
	})

	// Add logout endpoint
	router.POST("/logout", func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")
		if tokenString == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Authorization header missing"})
			return
		}

		// Remove token from cache
		cacheKey := "jwt:" + tokenString
		if err := DeleteCache(cacheKey); err != nil {
			log.Printf("Failed to delete cache: %v", err)
		}

		c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
	})

	// Start server
	log.Println("Server starting on :8080")
	router.Run(":8080")
}
