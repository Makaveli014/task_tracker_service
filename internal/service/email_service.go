package service

import (
	"context"
	"fmt"
	"log"
	"math/rand"

	"TestTask_Bazis/internal/circuitbreaker"
)

type EmailService struct {
	breaker *circuitbreaker.Breaker
}

func NewEmailService(breaker *circuitbreaker.Breaker) *EmailService {
	return &EmailService{breaker: breaker}
}

func (s *EmailService) SendInvite(ctx context.Context, toEmail, teamName string) error {
	return s.breaker.Execute(func() error {
		return s.mockSend(ctx, toEmail, teamName)
	})
}

func (s *EmailService) mockSend(_ context.Context, toEmail, teamName string) error {
	// Mock: 10% шанс ошибки для демонстрации circuit breaker
	if rand.Float64() < 0.1 {
		return fmt.Errorf("email service unavailable")
	}
	log.Printf("[EMAIL MOCK] Invite to team %q sent to %s", teamName, toEmail)
	return nil
}
