package main

import (
	"testing"

	"github.com/uadmin/uadmin"
)

func TestAppConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  AppConfig
		wantErr bool
	}{
		{
			name: "valid development config",
			config: AppConfig{
				Port:        1123,
				Environment: "DEVELOPMENT",
			},
			wantErr: false,
		},
		{
			name: "valid production config",
			config: AppConfig{
				Port:        1123,
				Environment: "PRODUCTION",
				Database: &uadmin.DBSettings{
					Host:     "localhost",
					Name:     "esl_system",
					User:     "postgres",
					Password: "password",
					Port:     5432,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid server port",
			config: AppConfig{
				Port:        0,
				Environment: "DEVELOPMENT",
			},
			wantErr: true,
		},
		{
			name: "production without database",
			config: AppConfig{
				Port:        1123,
				Environment: "PRODUCTION",
			},
			wantErr: true,
		},
		{
			name: "production with missing database host",
			config: AppConfig{
				Port:        1123,
				Environment: "PRODUCTION",
				Database: &uadmin.DBSettings{
					Name:     "esl_system",
					User:     "postgres",
					Password: "password",
					Port:     5432,
				},
			},
			wantErr: true,
		},
		{
			name: "production with missing database name",
			config: AppConfig{
				Port:        1123,
				Environment: "PRODUCTION",
				Database: &uadmin.DBSettings{
					Host:     "localhost",
					User:     "postgres",
					Password: "password",
					Port:     5432,
				},
			},
			wantErr: true,
		},
		{
			name: "production with missing database user",
			config: AppConfig{
				Port:        1123,
				Environment: "PRODUCTION",
				Database: &uadmin.DBSettings{
					Host:     "localhost",
					Name:     "esl_system",
					Password: "password",
					Port:     5432,
				},
			},
			wantErr: true,
		},
		{
			name: "production with missing database password",
			config: AppConfig{
				Port:        1123,
				Environment: "PRODUCTION",
				Database: &uadmin.DBSettings{
					Host: "localhost",
					Name: "esl_system",
					User: "postgres",
					Port: 5432,
				},
			},
			wantErr: true,
		},
		{
			name: "production with invalid database port",
			config: AppConfig{
				Port:        1123,
				Environment: "PRODUCTION",
				Database: &uadmin.DBSettings{
					Host:     "localhost",
					Name:     "esl_system",
					User:     "postgres",
					Password: "password",
					Port:     0,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAppConfigIsConfigured(t *testing.T) {
	t.Run("unconfigured when environment is empty", func(t *testing.T) {
		config := AppConfig{}

		if config.IsConfigured() {
			t.Fatal("expected configuration to be unconfigured")
		}
	})

	t.Run("configured in development without database", func(t *testing.T) {
		config := AppConfig{
			Environment: "DEVELOPMENT",
		}

		if !config.IsConfigured() {
			t.Fatal("expected development configuration to be configured")
		}
	})

	t.Run("unconfigured in production without database", func(t *testing.T) {
		config := AppConfig{
			Environment: "PRODUCTION",
		}

		if config.IsConfigured() {
			t.Fatal("expected production configuration without database to be unconfigured")
		}
	})

	t.Run("configured in production with database", func(t *testing.T) {
		config := AppConfig{
			Environment: "PRODUCTION",
			Database:    &uadmin.DBSettings{},
		}

		if !config.IsConfigured() {
			t.Fatal("expected production configuration with database to be configured")
		}
	})
}
func TestLoadAppConfig(t *testing.T) {
	t.Run("loads development configuration", func(t *testing.T) {
		t.Setenv("ENVIRONMENT", "DEVELOPMENT")
		t.Setenv("UADMIN_PORT", "8080")

		config := loadAppConfig()

		if config.Environment != "DEVELOPMENT" {
			t.Fatalf("expected environment DEVELOPMENT, got %q", config.Environment)
		}

		if config.Port != 8080 {
			t.Fatalf("expected port 8080, got %d", config.Port)
		}

		if config.Database != nil {
			t.Fatal("expected database configuration to be nil for development")
		}
	})

	t.Run("loads production database configuration", func(t *testing.T) {
		t.Setenv("ENVIRONMENT", "PRODUCTION")
		t.Setenv("UADMIN_PORT", "8080")
		t.Setenv("DB_HOST", "localhost")
		t.Setenv("DB_NAME", "esl")
		t.Setenv("DB_USER", "postgres")
		t.Setenv("DB_PASSWORD", "secret")
		t.Setenv("DB_PORT", "5433")

		config := loadAppConfig()

		if config.Environment != "PRODUCTION" {
			t.Fatalf("expected environment PRODUCTION, got %q", config.Environment)
		}

		if config.Database == nil {
			t.Fatal("expected database configuration to be present")
		}

		if config.Database.Host != "localhost" {
			t.Fatalf("expected database host localhost, got %q", config.Database.Host)
		}

		if config.Database.Name != "esl" {
			t.Fatalf("expected database name esl, got %q", config.Database.Name)
		}

		if config.Database.User != "postgres" {
			t.Fatalf("expected database user postgres, got %q", config.Database.User)
		}

		if config.Database.Password != "secret" {
			t.Fatalf("expected database password to be loaded")
		}

		if config.Database.Port != 5433 {
			t.Fatalf("expected database port 5433, got %d", config.Database.Port)
		}

		if config.Database.Type != "postgres" {
			t.Fatalf("expected database type postgres, got %q", config.Database.Type)
		}
	})

	t.Run("uses default ports when port environment variables are invalid", func(t *testing.T) {
		t.Setenv("ENVIRONMENT", "PRODUCTION")
		t.Setenv("UADMIN_PORT", "invalid")
		t.Setenv("DB_PORT", "invalid")

		config := loadAppConfig()

		if config.Port != 1123 {
			t.Fatalf("expected default application port 1123, got %d", config.Port)
		}

		if config.Database == nil {
			t.Fatal("expected database configuration to be present")
		}

		if config.Database.Port != 5432 {
			t.Fatalf("expected default database port 5432, got %d", config.Database.Port)
		}
	})
}
