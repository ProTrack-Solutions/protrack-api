package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/ProTrack-Solutions/protrack-api/internal/logger/discord"
	"github.com/ProTrack-Solutions/protrack-api/internal/logger/discord/domain"
	"github.com/gin-gonic/gin"
)

func GinDiscordErrorMiddleware(discordLog *discord.DiscordLogger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				stackTrace := string(debug.Stack())

				msg := fmt.Sprintf("**Rota:** %s %s\n**Erro:** %v\n\n```%s```",
					c.Request.Method, c.Request.URL.Path, err, truncateStack(stackTrace))

				discordLog.Send(domain.LevelError, "CRASH / PANIC no Gin", msg)

				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": "Internal Server Error",
				})
			}
		}()

		// Processa os próximos handlers
		c.Next()

		// Intercepta se o contexto tiver erros acumulados ou status 500+
		if c.Writer.Status() >= 500 {
			errMsgs := c.Errors.String()
			if errMsgs == "" {
				errMsgs = "Nenhuma mensagem de erro detalhada foi anexada ao contexto."
			}

			msg := fmt.Sprintf("**Rota:** %s %s\n**Status:** %d\n**Erros:** %s",
				c.Request.Method, c.Request.URL.Path, c.Writer.Status(), errMsgs)

			discordLog.Send(domain.LevelError, "Erro 5XX no Gin", msg)
		}
	}
}

func truncateStack(stack string) string {
	if len(stack) > 1500 {
		return stack[:1500] + "\n... [stack truncado]"
	}
	return stack
}
