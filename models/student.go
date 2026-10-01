package models

import (
	"errors"
	"strings"

	"github.com/uadmin/uadmin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
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

	password := username
	passwordHash, err := hashStudentPassword(password)
	if err != nil {
		return StudentCredentials{}, err
	}

	user := uadmin.User{
		FirstName:    strings.TrimSpace(s.FirstName),
		LastName:     strings.TrimSpace(s.LastName),
		Username:     strings.ToLower(username),
		Password:     passwordHash,
		Active:       true,
		RemoteAccess: true,
	}

	db := uadmin.GetDB()

	err = db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&user).Error; err != nil {
			return err
		}

		s.UserID = user.ID

		if err := tx.Save(s).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return StudentCredentials{}, err
	}

	return StudentCredentials{
		Username: user.Username,
		Password: password,
	}, nil
}

func (s *Student) Update() error {
	if err := validateStudentNames(s.FirstName, s.LastName); err != nil {
		return err
	}

	if s.UserID == 0 {
		return errors.New("student user is required")
	}

	db := uadmin.GetDB()

	return db.Transaction(func(tx *gorm.DB) error {
		var user uadmin.User

		if err := tx.Where("id = ?", s.UserID).First(&user).Error; err != nil {
			return err
		}

		user.FirstName = strings.TrimSpace(s.FirstName)
		user.LastName = strings.TrimSpace(s.LastName)

		if err := tx.Save(&user).Error; err != nil {
			return err
		}

		s.FirstName = user.FirstName
		s.LastName = user.LastName

		return tx.Save(s).Error
	})
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

func hashStudentPassword(password string) (string, error) {
	passwordBytes := []byte(password + uadmin.Salt)

	if len(passwordBytes) > 72 {
		passwordBytes = passwordBytes[:72]
	}

	hash, err := bcrypt.GenerateFromPassword(passwordBytes, 5)
	if err != nil {
		return "", err
	}

	return string(hash), nil
}
