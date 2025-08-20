package utils

import (
	"context"
	"os"
	"testing"
)

func TestSendValidation_NoRecipients(t *testing.T) {
	// Ensure env is present but it doesn't matter for this test
	os.Setenv("SMTP_EMAIL", "test@example.com")
	defer os.Unsetenv("SMTP_EMAIL")

	client := NewSMTPClient()
	err := client.Send(context.Background(), []string{}, "sub", "body")
	if err == nil {
		t.Fatalf("expected error for no recipients, got nil")
	}
}

func TestSendValidation_NoSender(t *testing.T) {
	// Clear sender env
	prev := os.Getenv("SMTP_EMAIL")
	os.Unsetenv("SMTP_EMAIL")
	defer os.Setenv("SMTP_EMAIL", prev)

	client := NewSMTPClient()
	err := client.Send(context.Background(), []string{"a@b.com"}, "s", "b")
	if err == nil {
		t.Fatalf("expected error for missing sender, got nil")
	}
}
