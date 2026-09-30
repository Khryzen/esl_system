package models

import (
	"strconv"
	"strings"

	"github.com/uadmin/uadmin"
)

func generateUsername(firstName, lastName string) string {
	firstName = strings.TrimSpace(firstName)
	lastName = strings.TrimSpace(lastName)

	if firstName == "" || lastName == "" {
		return ""
	}

	firstInitial := []rune(firstName)[0]

	base := strings.ToLower(string(firstInitial) + lastName)
	base = strings.ReplaceAll(base, " ", "")

	user := uadmin.User{}

	candidate := base
	if uadmin.Count(&user, "username = ?", candidate) == 0 {
		return candidate
	}

	for suffix := 2; ; suffix++ {
		candidate = base + strconv.Itoa(suffix)

		if uadmin.Count(&user, "username = ?", candidate) == 0 {
			return candidate
		}
	}
}
