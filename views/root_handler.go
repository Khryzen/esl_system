package views

import (
	"net/http"
	"strings"

	"github.com/uadmin/uadmin"
)

func RootHandler(w http.ResponseWriter, r *http.Request) {
	session := uadmin.IsAuthenticated(r)
	if session == nil {
		http.Redirect(w, r, "/login/", http.StatusSeeOther)
		return
	}

	page := strings.TrimPrefix(r.URL.Path, "/")
	page = strings.TrimSuffix(page, "/")

	context := map[string]interface{}{}

	switch page {
	case "":
		page = "dashboard"
	default:
		page = "dashboard"
	}

	Render(w, r, page, context)
}
