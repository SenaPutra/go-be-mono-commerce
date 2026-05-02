package auth

import (
	"testing"
	"time"
)

func TestSignAndParse(t *testing.T) {
	tok, err := SignWithTTL("secret", "user-1", RoleCustomer, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := Parse("secret", tok)
	if err != nil {
		t.Fatal(err)
	}
	if claims.UserID != "user-1" || claims.Role != RoleCustomer {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestParseInvalidSecret(t *testing.T) {
	tok, _ := SignWithTTL("secret", "user-1", RoleCustomer, time.Hour)
	if _, err := Parse("other", tok); err == nil {
		t.Fatal("expected error")
	}
}
