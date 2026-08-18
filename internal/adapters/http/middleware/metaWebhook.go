package middleware

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

func MetaWhatsappMiddleware(appSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		payload, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			c.Abort()
			return
		}

		signature := c.GetHeader("X-Hub-Signature-256")
		if signature == "" {
			c.JSON(http.StatusForbidden, gin.H{"error": "missing signature"})
			c.Abort()
			return
		}

		mac := hmac.New(sha256.New, []byte(appSecret))
		mac.Write(payload)
		expectedMAC := mac.Sum(nil)
		expectedSignature := "sha256=" + hex.EncodeToString(expectedMAC)

		if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
			c.JSON(http.StatusForbidden, gin.H{"error": "invalid signature"})
			c.Abort()
			return
		}

		c.Request.Body = io.NopCloser(bytes.NewBuffer(payload))
		c.Next()
	}
}
