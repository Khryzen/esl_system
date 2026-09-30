package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRegisterRoutes(t *testing.T) {
	originalMux := http.DefaultServeMux
	defer func() {
		http.DefaultServeMux = originalMux
	}()

	http.DefaultServeMux = http.NewServeMux()

	registerRoutes()

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
}
