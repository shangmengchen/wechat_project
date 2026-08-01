package auth

import (
	"strings"
	"testing"
	"time"
)

func TestParseTokenRejectsTamperedSignature(t *testing.T) {
	token, err := SignToken("secret", time.Hour, "u1", "openid-1")
	if err != nil {
		t.Fatalf("SignToken: %v", err)
	}
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		t.Fatalf("unexpected token format: %q", token)
	}

	_, err = ParseToken("secret", parts[0]+".tampered")
	if err != ErrInvalidToken {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}
