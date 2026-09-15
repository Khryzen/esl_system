package views

import (
	"encoding/json"
	"net/http"

	"github.com/uadmin/uadmin"
)

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	context := map[string]interface{}{}

	session := uadmin.IsAuthenticated(r)
	if session != nil {
		http.Redirect(w, r, "/dashboard/", http.StatusSeeOther)
		return
	}

	uadmin.Trail(uadmin.DEBUG, "METHOD: %v", r.Method)

	if r.Method == http.MethodPost {
		uadmin.Trail(uadmin.INFO, "Login Post")

		username := r.FormValue("username")
		password := r.FormValue("password")

		user := uadmin.User{}

		uadmin.Get(&user, "username = ?", username)
		session = user.Login(password, "")

		if session == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)

			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Invalid username or password.",
			})

			return
		}

		http.SetCookie(w, &http.Cookie{
			Path:  "/",
			Name:  "session",
			Value: session.Key,
		})

		http.SetCookie(w, &http.Cookie{
			Path:  "/",
			Name:  "username",
			Value: user.Username,
		})

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":  true,
			"redirect": "/dashboard/",
		})

		return
	}

	uadmin.RenderHTML(w, r, "templates/auth/login.html", context)
}
