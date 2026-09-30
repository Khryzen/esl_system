package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/uadmin/uadmin"
)

func TestSetupApplication(t *testing.T) {
	originalDatabase := uadmin.Database
	originalPort := uadmin.Port
	originalRootURL := uadmin.RootURL
	originalMux := http.DefaultServeMux

	defer func() {
		uadmin.Database = originalDatabase
		uadmin.Port = originalPort
		uadmin.RootURL = originalRootURL
		http.DefaultServeMux = originalMux
	}()

	http.DefaultServeMux = http.NewServeMux()

	database := &uadmin.DBSettings{
		Host:     "localhost",
		Name:     "testdb",
		User:     "testuser",
		Password: "testpass",
		Port:     5432,
		Type:     "postgres",
	}

	setupApplication(AppConfig{
		Port:     8080,
		Database: database,
	})

	t.Run("configures database", func(t *testing.T) {
		if uadmin.Database != database {
			t.Fatal("expected configured database to be applied")
		}
	})

	t.Run("configures server", func(t *testing.T) {
		if uadmin.Port != 8080 {
			t.Fatalf("expected uadmin.Port to be 8080, got %d", uadmin.Port)
		}

		if uadmin.RootURL != "/admin/" {
			t.Fatalf(
				"expected uadmin.RootURL to be /admin/, got %q",
				uadmin.RootURL,
			)
		}
	})

	t.Run("registers models", func(t *testing.T) {
		models := []string{
			"assessment",
			"class",
			"course",
			"coursematerial",
			"enrollment",
			"homework",
			"invoice",
			"level",
			"material",
			"package",
			"student",
			"teacher",
		}

		for _, modelName := range models {
			if _, ok := uadmin.NewModel(modelName, false); !ok {
				t.Fatalf("expected %s to be registered", modelName)
			}
		}
	})

	t.Run("registers routes", func(t *testing.T) {
		tests := []struct {
			name    string
			path    string
			pattern string
		}{
			{
				name:    "login route",
				path:    "/login/",
				pattern: "/login/",
			},
			{
				name:    "logout route",
				path:    "/logout/",
				pattern: "/logout/",
			},
			{
				name:    "root route",
				path:    "/",
				pattern: "/",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				req := httptest.NewRequest(http.MethodGet, tt.path, nil)

				_, pattern := http.DefaultServeMux.Handler(req)

				if pattern != tt.pattern {
					t.Fatalf(
						"expected route pattern %q, got %q",
						tt.pattern,
						pattern,
					)
				}
			})
		}
	})
}
