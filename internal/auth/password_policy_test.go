package auth

import "testing"

func TestValidatePasswordAcceptsStrongPassword(t *testing.T) {
	if err := ValidatePassword("MyStr0ng!Pass"); err != nil {
		t.Fatalf("expected valid password, got %v", err)
	}
}

func TestValidatePasswordRejectsTooShort(t *testing.T) {
	err := ValidatePassword("Ab1!")
	if err == nil {
		t.Fatal("expected error for short password")
	}
	if err.Code != "password_too_short" {
		t.Fatalf("expected password_too_short, got %q", err.Code)
	}
}

func TestValidatePasswordRejectsMissingCharacterClasses(t *testing.T) {
	err := ValidatePassword("alllowercase1!")
	if err == nil || err.Code != "password_requirements_not_met" {
		t.Fatalf("expected requirements error, got %v", err)
	}
	if err.Message == "" {
		t.Fatal("expected user-facing message")
	}
}

func TestValidatePasswordRejectsCommonPassword(t *testing.T) {
	err := ValidatePassword("Password1!")
	if err == nil || err.Code != "password_too_common" {
		t.Fatalf("expected common password error, got %v", err)
	}
}
