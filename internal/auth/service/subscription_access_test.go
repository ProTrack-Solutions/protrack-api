package service

import (
	"testing"
	"time"

	subscriptionDomain "github.com/ProTrack-Solutions/protrack-api/internal/subscriptions/domain"
)

func TestCheckSubscriptionAccess(t *testing.T) {
	future := time.Now().Add(24 * time.Hour)
	past := time.Now().Add(-24 * time.Hour)

	tests := []struct {
		status    string
		periodEnd time.Time
		wantErr   error
	}{
		{"active", future, nil},
		{"authorized", future, nil},
		{"trialing", future, nil},
		{"trialing", past, ErrSubscriptionExpired},
		{"canceled", future, ErrSubscriptionCanceled},
		{"paused", future, ErrSubscriptionPaused},
		{"unpaid", future, ErrSubscriptionPaused},
		{"incomplete", future, ErrSubscriptionIncomplete},
		{"incomplete_expired", future, ErrSubscriptionIncomplete},
		{"active", past, ErrSubscriptionExpired},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			err := checkSubscriptionAccess(subscriptionDomain.SubscriptionResponse{
				Status:           tt.status,
				CurrentPeriodEnd: tt.periodEnd,
			})
			if err != tt.wantErr {
				t.Errorf("status %q: esperava %v, obteve %v", tt.status, tt.wantErr, err)
			}
		})
	}
}
