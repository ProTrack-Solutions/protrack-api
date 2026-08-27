package extractorcontext

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func ExtratorUserID(c *gin.Context) (uuid.UUID, error) {
	userIdAny, exists := c.Get("sub")
	if !exists {
		return uuid.Nil, errors.New("unauthorized: user_id not found in context")
	}

	userIdStr, ok := userIdAny.(string)
	if !ok {
		return uuid.Nil, errors.New("unauthorized: invalid user_id type in context")
	}

	userId, err := uuid.Parse(userIdStr)
	if err != nil {
		return uuid.Nil, errors.New("unauthorized: invalid user_id type in context")
	}

	return userId, nil
}