package views

import (
	"net/http"
	"time"
)

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	for _, c := range r.Cookies() {
		http.SetCookie(w, &http.Cookie{
			Name:    c.Name,
			Value:   "",
			Path:    "/",
			Expires: time.Unix(0, 0),
			MaxAge:  -1,
		})
	}
	http.Redirect(w, r, "/login/", http.StatusSeeOther)
}
