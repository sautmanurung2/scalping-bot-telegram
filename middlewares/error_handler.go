package middlewares

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ErrorHandlerMiddleware menangani kesalahan yang terjadi selama eksekusi Gin secara terpusat
func ErrorHandlerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) > 0 {
			err := c.Errors.Last()
			log.Printf("[ERROR INTERCEPTOR] Internal Error: %v", err.Err)

			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  "error",
				"code":    500,
				"message": "Terjadi kesalahan internal pada server bot.",
			})
		}
	}
}
