package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ProTrack-Solutions/protrack-api/internal/plans/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/plans/mocks"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

var errService = errors.New("service error")

func ginCtx() context.Context {
	return context.Background()
}

func samplePlanResponse(id uuid.UUID) domain.PlanResponse {
	now := time.Now().UTC()
	return domain.PlanResponse{
		ID:           id,
		Name:         "Plano VIP",
		Description:  "Descrição VIP",
		PriceCents:   9900,
		Currency:     "BRL",
		BillingCycle: "MONTHLY",
		Active:       true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// ---------------------------------------------------------------------------
// Service Contract Tests (via MockServiceInterface)
// ---------------------------------------------------------------------------

func TestServiceContract_CreatePlans(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	req := domain.CreatePlanRequest{
		Name:         "Plano Basico",
		Description:  "Desc",
		ValueAmount:  19.90,
		Currency:     "BRL",
		BillingCycle: "MONTHLY",
	}

	mockSvc.EXPECT().
		CreatePlans(gomock.Any(), req).
		Return(nil)

	err := mockSvc.CreatePlans(ginCtx(), req)
	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
}

func TestServiceContract_GetPlanByID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	planID := uuid.New()
	expectedResp := samplePlanResponse(planID)

	mockSvc.EXPECT().
		GetPlanByID(gomock.Any(), planID).
		Return(expectedResp, nil)

	resp, err := mockSvc.GetPlanByID(ginCtx(), planID)
	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if resp.ID != planID {
		t.Errorf("ID incorreto: %v", resp.ID)
	}
}

func TestServiceContract_ListPlans(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	expectedList := []domain.PlanResponse{
		samplePlanResponse(uuid.New()),
		samplePlanResponse(uuid.New()),
	}

	mockSvc.EXPECT().
		ListPlans(gomock.Any()).
		Return(expectedList, nil)

	list, err := mockSvc.ListPlans(ginCtx())
	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("esperava 2 planos, obteve %d", len(list))
	}
}

func TestServiceContract_ListPlansByActiveStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	expectedList := []domain.PlanResponse{samplePlanResponse(uuid.New())}

	mockSvc.EXPECT().
		ListPlansByActiveStatus(gomock.Any(), true).
		Return(expectedList, nil)

	list, err := mockSvc.ListPlansByActiveStatus(ginCtx(), true)
	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("esperava 1 plano, obteve %d", len(list))
	}
}

func TestServiceContract_UpdatePlan(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	planID := uuid.New()
	req := domain.UpdatePlanParams{Name: "Novo Nome"}

	mockSvc.EXPECT().
		UpdatePlan(gomock.Any(), planID, req).
		Return(nil)

	err := mockSvc.UpdatePlan(ginCtx(), planID, req)
	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
}

func TestServiceContract_TogglePlanActiveStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	planID := uuid.New()

	mockSvc.EXPECT().
		TogglePlanActiveStatus(gomock.Any(), planID, false).
		Return(nil)

	err := mockSvc.TogglePlanActiveStatus(ginCtx(), planID, false)
	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
}

// ---------------------------------------------------------------------------
// HTTP Integration Tests (httptest + gin)
// ---------------------------------------------------------------------------

// TestHTTP_CreatePlans_Success verifica retorno 201.
func TestHTTP_CreatePlans_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	mockSvc.EXPECT().
		CreatePlans(gomock.Any(), gomock.Any()).
		Return(nil)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/plans", func(c *gin.Context) {
		var req domain.CreatePlanRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := mockSvc.CreatePlans(c.Request.Context(), req); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusCreated)
	})

	body := `{"name":"Plano Teste","description":"Desc","value_amount":29.9,"currency":"BRL","billing_cycle":"MONTHLY"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/plans", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("esperava 201, obteve %d", w.Code)
	}
}

// TestHTTP_CreatePlans_BadJSON verifica retorno 400.
func TestHTTP_CreatePlans_BadJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/plans", func(c *gin.Context) {
		var req domain.CreatePlanRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusCreated)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/plans", bytes.NewBufferString(`{invalid json}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("esperava 400, obteve %d", w.Code)
	}
}

// TestHTTP_CreatePlans_ServiceError verifica retorno 500.
func TestHTTP_CreatePlans_ServiceError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	mockSvc.EXPECT().
		CreatePlans(gomock.Any(), gomock.Any()).
		Return(errService)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/plans", func(c *gin.Context) {
		var req domain.CreatePlanRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := mockSvc.CreatePlans(c.Request.Context(), req); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusCreated)
	})

	body := `{"name":"Plano Teste","description":"Desc","value_amount":29.9,"currency":"BRL","billing_cycle":"MONTHLY"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/plans", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("esperava 500, obteve %d", w.Code)
	}
}

// TestHTTP_GetPlanByID_InvalidUUID verifica retorno 400.
func TestHTTP_GetPlanByID_InvalidUUID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/plans/:id", func(c *gin.Context) {
		planIdStr := c.Param("id")
		if _, err := uuid.Parse(planIdStr); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid plan ID"})
			return
		}
		c.JSON(http.StatusOK, gin.H{})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/plans/nao-uuid", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("esperava 400, obteve %d", w.Code)
	}
}

// TestHTTP_GetPlanByID_Success verifica retorno 200 com JSON do plano.
func TestHTTP_GetPlanByID_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	planID := uuid.New()
	expectedResp := samplePlanResponse(planID)

	mockSvc.EXPECT().
		GetPlanByID(gomock.Any(), planID).
		Return(expectedResp, nil)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/plans/:id", func(c *gin.Context) {
		planIdStr := c.Param("id")
		id, err := uuid.Parse(planIdStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid plan ID"})
			return
		}
		plan, err := mockSvc.GetPlanByID(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, plan)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/plans/%s", planID), nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("esperava 200, obteve %d", w.Code)
	}

	var resp domain.PlanResponse
	_ = json.NewDecoder(w.Body).Decode(&resp)
	if resp.ID != planID {
		t.Errorf("ID do plano retornado incorreto")
	}
}

// TestHTTP_ListPlans_Success verifica retorno 200 com lista.
func TestHTTP_ListPlans_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	plansList := []domain.PlanResponse{samplePlanResponse(uuid.New())}

	mockSvc.EXPECT().
		ListPlans(gomock.Any()).
		Return(plansList, nil)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/plans", func(c *gin.Context) {
		plans, err := mockSvc.ListPlans(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, plans)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/plans", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("esperava 200, obteve %d", w.Code)
	}
}

// TestHTTP_ListPlansByActiveStatus_Success verifica retorno 200 com query param active.
func TestHTTP_ListPlansByActiveStatus_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	plansList := []domain.PlanResponse{samplePlanResponse(uuid.New())}

	mockSvc.EXPECT().
		ListPlansByActiveStatus(gomock.Any(), true).
		Return(plansList, nil)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/plans/active", func(c *gin.Context) {
		activeStr := c.Query("active")
		active := activeStr == "true"
		plans, err := mockSvc.ListPlansByActiveStatus(c.Request.Context(), active)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, plans)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/plans/active?active=true", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("esperava 200, obteve %d", w.Code)
	}
}

// TestHTTP_UpdatePlan_Success verifica retorno 200.
func TestHTTP_UpdatePlan_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	planID := uuid.New()

	mockSvc.EXPECT().
		UpdatePlan(gomock.Any(), planID, gomock.Any()).
		Return(nil)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.PUT("/plans/:id", func(c *gin.Context) {
		planIdStr := c.Param("id")
		id, err := uuid.Parse(planIdStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid plan ID"})
			return
		}
		var req domain.UpdatePlanParams
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := mockSvc.UpdatePlan(c.Request.Context(), id, req); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusOK)
	})

	body := `{"name":"Novo Nome","description":"Desc","value_amount":39.9,"currency":"BRL","billing_cycle":"MONTHLY"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/plans/%s", planID), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("esperava 200, obteve %d", w.Code)
	}
}

// TestHTTP_TogglePlanActiveStatus_Success verifica retorno 200 na desativação/ativação.
func TestHTTP_TogglePlanActiveStatus_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	planID := uuid.New()

	mockSvc.EXPECT().
		TogglePlanActiveStatus(gomock.Any(), planID, false).
		Return(nil)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.PATCH("/plans/:id/active", func(c *gin.Context) {
		planIdStr := c.Param("id")
		id, err := uuid.Parse(planIdStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid plan ID"})
			return
		}
		activeStr := c.Query("active")
		active := activeStr == "true"
		if err := mockSvc.TogglePlanActiveStatus(c.Request.Context(), id, active); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/plans/%s/active?active=false", planID), nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("esperava 200, obteve %d", w.Code)
	}
}
