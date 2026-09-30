package main

import (
	"testing"

	"github.com/uadmin/uadmin"
)

func TestConfigureServer(t *testing.T) {
	originalPort := uadmin.Port
	originalRootURL := uadmin.RootURL

	defer func() {
		uadmin.Port = originalPort
		uadmin.RootURL = originalRootURL
	}()

	configureServer(AppConfig{
		Port: 8080,
	})

	if uadmin.Port != 8080 {
		t.Fatalf("expected uadmin.Port to be 8080, got %d", uadmin.Port)
	}

	if uadmin.RootURL != "/admin/" {
		t.Fatalf("expected uadmin.RootURL to be /admin/, got %q", uadmin.RootURL)
	}
}
