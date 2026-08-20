package extractorcontext

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func ExtratorCompanyID(c *gin.Context) (uuid.UUID, error) {
	companyIdAny, exists := c.Get("company_id")
	if !exists {
		return uuid.Nil, errors.New("unauthorized: invalid company_id type in context")
	}

	companyId := companyIdAny.(uuid.UUID)

	return companyId, nil
}
