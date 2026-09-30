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
	t.Run("unconfigured when database is nil", func(t *testing.T) {
		config := AppConfig{}

		if config.IsConfigured() {
			t.Fatal("expected configuration to be unconfigured")
		}
	})

	t.Run("configured when database settings are present", func(t *testing.T) {
		config := AppConfig{
			Database: &uadmin.DBSettings{},
		}

		if !config.IsConfigured() {
			t.Fatal("expected configuration to be configured")
		}
	})
}
