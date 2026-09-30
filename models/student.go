package models

import (
	"errors"
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

func (s *Student) Update() error {
	if err := validateStudentNames(s.FirstName, s.LastName); err != nil {
		return err
	}

	if s.UserID == 0 {
		return errors.New("student user is required")
	}

	user := uadmin.User{}

	if err := uadmin.Get(&user, "id = ?", s.UserID); err != nil {
		return err
	}

	user.FirstName = strings.TrimSpace(s.FirstName)
	user.LastName = strings.TrimSpace(s.LastName)

	if err := uadmin.Save(&user); err != nil {
		return err
	}

	s.FirstName = user.FirstName
	s.LastName = user.LastName

	return uadmin.Save(s)
}

func studentUsername(firstName, lastName string) (string, error) {
	firstName = strings.TrimSpace(firstName)
	lastName = strings.TrimSpace(lastName)

	if firstName == "" {
		return "", errors.New("first name is required")
	}

	if lastName == "" {
		return "", errors.New("last name is required")
	}

	return generateUsername(firstName, lastName), nil
}

func validateStudentNames(firstName, lastName string) error {
	if strings.TrimSpace(firstName) == "" {
		return errors.New("first name is required")
	}

	if strings.TrimSpace(lastName) == "" {
		return errors.New("last name is required")
	}

	return nil
}
