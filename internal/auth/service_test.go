package auth

import "testing"

func TestValidateEmailPassword(t *testing.T) {
	if errs := validateEmailPassword("invalid", "123"); len(errs) != 2 {
		t.Fatalf("expected 2 errs, got %v", errs)
	}
	if errs := validateEmailPassword("user@example.com", "12345678"); len(errs) != 0 {
		t.Fatalf("expected no errs, got %v", errs)
	}
}
