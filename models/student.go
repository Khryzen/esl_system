package models

import (
	"fmt"
	"strings"

	"github.com/uadmin/uadmin"
)

type Student struct {
	uadmin.Model
	FirstName string `uadmin:"required"`
	LastName  string
	WeChatID  string `uadmin:"required"`
	Email     string

	// This is for the user access if the client also wants to have a user account for the student
	User   uadmin.User
	UserID uint
}

func (s Student) String() string {
	return s.FirstName + " " + s.LastName
}

// func (s *Student) Save() map[string]any {
// 	user := uadmin.User{}
// 	user.FirstName = s.FirstName
// 	user.LastName = s.LastName
// 	username := strings.ReplaceAll(s.FirstName, " ", "")[:1] + strings.ReplaceAll(s.LastName, " ", "")
// 	user.Username = username
// 	user.Password = strings.ReplaceAll(s.FirstName, " ", "")[:1] + strings.ReplaceAll(s.LastName, " ", "")
// 	user.Active = true
// 	user.RemoteAccess = true

// 	user.Save()

// 	uadmin.Get(&user, "username = ?", username)
// 	s.UserID = user.ID
// 	uadmin.Save(s)

// 	return map[string]any{
// 		"username": user.Username,
// 		"password": user.Username,
// 	}
// }

type StudentCredentials struct {
	Username string
	Password string
}

func (s *Student) Create() (StudentCredentials, error) {
	username, err := studentUsername(s.FirstName, s.LastName)
	if err != nil {
		return StudentCredentials{}, err
	}

	user := uadmin.User{
		FirstName:    s.FirstName,
		LastName:     s.LastName,
		Username:     username,
		Password:     username,
		Active:       true,
		RemoteAccess: true,
	}

	if err := uadmin.Save(&user); err != nil {
		return StudentCredentials{}, err
	}

	if err := uadmin.Get(&user, "username = ?", username); err != nil {
		return StudentCredentials{}, err
	}

	s.UserID = user.ID

	if err := uadmin.Save(s); err != nil {
		return StudentCredentials{}, err
	}

	return StudentCredentials{
		Username: user.Username,
		Password: username,
	}, nil
}

func studentUsername(firstName, lastName string) (string, error) {
	firstName = strings.ReplaceAll(strings.TrimSpace(firstName), " ", "")
	lastName = strings.ReplaceAll(strings.TrimSpace(lastName), " ", "")

	if firstName == "" || lastName == "" {
		return "", fmt.Errorf("first name and last name are required")
	}

	return firstName[:1] + lastName, nil
}
