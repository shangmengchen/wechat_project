package configs

import "testing"

func TestValidateForStartupRejectsWeakReleaseDefaults(t *testing.T) {
	cfg := defaultConfig()

	if err := ValidateForStartup(cfg); err == nil {
		t.Fatal("expected weak release config to be rejected")
	}
}

func TestValidateForStartupAllowsNonReleaseDefaults(t *testing.T) {
	cfg := defaultConfig()
	cfg.AppConfig.RunMode = "debug"

	if err := ValidateForStartup(cfg); err != nil {
		t.Fatalf("expected non-release config to pass, got %v", err)
	}
}

func TestValidateForStartupAllowsStrongReleaseConfig(t *testing.T) {
	cfg := defaultConfig()
	cfg.AuthConfig.TokenSecret = "strong-token-secret-for-release"
	cfg.AdminConfig.Password = "strong-admin-password"
	cfg.DbConfig.Password = "strong-db-password"

	if err := ValidateForStartup(cfg); err != nil {
		t.Fatalf("expected strong release config to pass, got %v", err)
	}
}

func TestValidateForStartupAllowsMySQLDSN(t *testing.T) {
	t.Setenv("MYSQL_DSN", "user:strong-db-password@tcp(localhost:3306)/couple_mini")
	cfg := defaultConfig()
	cfg.AuthConfig.TokenSecret = "strong-token-secret-for-release"
	cfg.AdminConfig.Password = "strong-admin-password"

	if err := ValidateForStartup(cfg); err != nil {
		t.Fatalf("expected MYSQL_DSN config to pass, got %v", err)
	}
}
