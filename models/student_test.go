package models

import (
	"testing"

	"github.com/uadmin/uadmin"
	"gorm.io/gorm"
)

func TestStudentUsername(t *testing.T) {
	setupStudentCreateTestDB(t)

	tests := []struct {
		name      string
		firstName string
		lastName  string
		want      string
		wantErr   bool
	}{
		{
			name:      "normal name",
			firstName: "Mark",
			lastName:  "Cruz",
			want:      "mcruz",
		},
		{
			name:      "first name with spaces",
			firstName: "Mary Ann",
			lastName:  "Cruz",
			want:      "mcruz",
		},
		{
			name:      "last name with spaces",
			firstName: "Mark",
			lastName:  "De La Cruz",
			want:      "mdelacruz",
		},
		{
			name:      "uppercase name",
			firstName: "MARK",
			lastName:  "CRUZ",
			want:      "mcruz",
		},
		{
			name:      "missing first name",
			firstName: "",
			lastName:  "Cruz",
			wantErr:   true,
		},
		{
			name:      "missing last name",
			firstName: "Mark",
			lastName:  "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := studentUsername(tt.firstName, tt.lastName)

			if (err != nil) != tt.wantErr {
				t.Fatalf(
					"studentUsername() error = %v, wantErr %v",
					err,
					tt.wantErr,
				)
			}

			if !tt.wantErr && got != tt.want {
				t.Fatalf(
					"studentUsername() = %q, want %q",
					got,
					tt.want,
				)
			}
		})
	}
}

func TestStudentUsernameDuplicate(t *testing.T) {
	setupStudentCreateTestDB(t)

	existing := uadmin.User{
		Username: "mcruz",
	}

	if err := uadmin.Save(&existing); err != nil {
		t.Fatalf("failed to create existing user: %v", err)
	}

	got, err := studentUsername("Mark", "Cruz")
	if err != nil {
		t.Fatalf("studentUsername() error = %v", err)
	}

	if got != "mcruz2" {
		t.Fatalf(
			"studentUsername() = %q, want %q",
			got,
			"mcruz2",
		)
	}
}

func TestStudentUsernameMultipleDuplicates(t *testing.T) {
	setupStudentCreateTestDB(t)

	usernames := []string{
		"mcruz",
		"mcruz2",
		"mcruz3",
	}

	for _, username := range usernames {
		user := uadmin.User{
			Username: username,
		}

		if err := uadmin.Save(&user); err != nil {
			t.Fatalf(
				"failed to create existing user %q: %v",
				username,
				err,
			)
		}
	}

	got, err := studentUsername("Mark", "Cruz")
	if err != nil {
		t.Fatalf("studentUsername() error = %v", err)
	}

	if got != "mcruz4" {
		t.Fatalf(
			"studentUsername() = %q, want %q",
			got,
			"mcruz4",
		)
	}
}

func setupStudentCreateTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	uadmin.ClearDB()

	uadmin.Database = &uadmin.DBSettings{
		Type: "sqlite",
		Name: t.TempDir() + "/student_test.db",
	}

	db := uadmin.GetDB()

	if err := db.AutoMigrate(&uadmin.User{}, &Student{}); err != nil {
		t.Fatalf("failed to migrate student test database: %v", err)
	}

	t.Cleanup(func() {
		uadmin.ClearDB()
		uadmin.Database = nil
	})

	return db
}

func TestStudentCreate(t *testing.T) {
	t.Run("creates student user and returns credentials", func(t *testing.T) {
		setupStudentCreateTestDB(t)

		student := Student{
			FirstName: "Mark",
			LastName:  "Cruz",
			WeChatID:  "mark-wechat",
			Email:     "mark@example.com",
		}

		creds, err := student.Create()
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		if creds.Username != "mcruz" {
			t.Fatalf(
				"Create() username = %q, want %q",
				creds.Username,
				"mcruz",
			)
		}

		if creds.Password != "mcruz" {
			t.Fatalf(
				"Create() password = %q, want %q",
				creds.Password,
				"mcruz",
			)
		}

		if student.ID == 0 {
			t.Fatal("Create() did not persist the student")
		}

		if student.UserID == 0 {
			t.Fatal("Create() did not associate the user")
		}

		var user uadmin.User

		if err := uadmin.Get(
			&user,
			"id = ?",
			student.UserID,
		); err != nil {
			t.Fatalf(
				"failed to load associated user: %v",
				err,
			)
		}

		if user.Username != "mcruz" {
			t.Fatalf(
				"associated user username = %q, want %q",
				user.Username,
				"mcruz",
			)
		}

		if user.FirstName != "Mark" || user.LastName != "Cruz" {
			t.Fatalf(
				"associated user name = %q %q, want %q %q",
				user.FirstName,
				user.LastName,
				"Mark",
				"Cruz",
			)
		}
	})

	t.Run("rejects invalid names before persistence", func(t *testing.T) {
		setupStudentCreateTestDB(t)

		student := Student{
			FirstName: "Mark",
			LastName:  "",
			WeChatID:  "mark-wechat",
		}

		if _, err := student.Create(); err == nil {
			t.Fatal("Create() error = nil, want validation error")
		}

		var users []uadmin.User

		if err := uadmin.All(&users); err != nil {
			t.Fatalf(
				"failed to query users: %v",
				err,
			)
		}

		if len(users) != 0 {
			t.Fatalf(
				"created %d users, want 0",
				len(users),
			)
		}
	})

	t.Run("generates unique username for duplicate names", func(t *testing.T) {
		setupStudentCreateTestDB(t)

		first := Student{
			FirstName: "Mark",
			LastName:  "Cruz",
			WeChatID:  "mark-one",
		}

		firstCreds, err := first.Create()
		if err != nil {
			t.Fatalf(
				"first Create() error = %v",
				err,
			)
		}

		if firstCreds.Username != "mcruz" {
			t.Fatalf(
				"first username = %q, want %q",
				firstCreds.Username,
				"mcruz",
			)
		}

		second := Student{
			FirstName: "Mary",
			LastName:  "Cruz",
			WeChatID:  "mary-one",
		}

		secondCreds, err := second.Create()
		if err != nil {
			t.Fatalf(
				"second Create() error = %v",
				err,
			)
		}

		if secondCreds.Username != "mcruz2" {
			t.Fatalf(
				"second username = %q, want %q",
				secondCreds.Username,
				"mcruz2",
			)
		}
	})

	t.Run("avoids username used by existing uadmin user", func(t *testing.T) {
		setupStudentCreateTestDB(t)

		existing := uadmin.User{
			Username: "mcruz",
		}

		if err := uadmin.Save(&existing); err != nil {
			t.Fatalf(
				"failed to create existing user: %v",
				err,
			)
		}

		student := Student{
			FirstName: "Mark",
			LastName:  "Cruz",
			WeChatID:  "mark-wechat",
		}

		creds, err := student.Create()
		if err != nil {
			t.Fatalf(
				"Create() error = %v",
				err,
			)
		}

		if creds.Username != "mcruz2" {
			t.Fatalf(
				"Create() username = %q, want %q",
				creds.Username,
				"mcruz2",
			)
		}
	})

	t.Run("leaves user persisted when student save fails", func(t *testing.T) {
		db := setupStudentCreateTestDB(t)

		if err := db.Migrator().DropTable(&Student{}); err != nil {
			t.Fatalf(
				"failed to drop student table: %v",
				err,
			)
		}

		student := Student{
			FirstName: "Mark",
			LastName:  "Cruz",
			WeChatID:  "mark-wechat",
		}

		_, err := student.Create()

		if err == nil {
			t.Fatal(
				"Create() error = nil, want student save error",
			)
		}

		var user uadmin.User

		if err := uadmin.Get(
			&user,
			"username = ?",
			"mcruz",
		); err != nil {
			t.Fatalf(
				"expected user to remain persisted after student save failure: %v",
				err,
			)
		}

		if user.Username != "mcruz" {
			t.Fatalf(
				"persisted user username = %q, want %q",
				user.Username,
				"mcruz",
			)
		}
	})
}

func TestStudentUpdate(t *testing.T) {
	t.Run("updates student and synchronizes associated user", func(t *testing.T) {
		setupStudentCreateTestDB(t)

		student := Student{
			FirstName: "Mark",
			LastName:  "Cruz",
			WeChatID:  "mark-wechat",
			Email:     "mark@example.com",
		}

		if _, err := student.Create(); err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		student.FirstName = "Mary"
		student.LastName = "Santos"
		student.WeChatID = "mary-wechat"
		student.Email = "mary@example.com"

		if err := student.Update(); err != nil {
			t.Fatalf("Update() error = %v", err)
		}

		var gotStudent Student

		if err := uadmin.Get(
			&gotStudent,
			"id = ?",
			student.ID,
		); err != nil {
			t.Fatalf("failed to load updated student: %v", err)
		}

		if gotStudent.FirstName != "Mary" ||
			gotStudent.LastName != "Santos" {
			t.Fatalf(
				"updated student name = %q %q, want %q %q",
				gotStudent.FirstName,
				gotStudent.LastName,
				"Mary",
				"Santos",
			)
		}

		if gotStudent.WeChatID != "mary-wechat" ||
			gotStudent.Email != "mary@example.com" {
			t.Fatalf(
				"updated student contact = %q %q, want %q %q",
				gotStudent.WeChatID,
				gotStudent.Email,
				"mary-wechat",
				"mary@example.com",
			)
		}

		var user uadmin.User

		if err := uadmin.Get(
			&user,
			"id = ?",
			student.UserID,
		); err != nil {
			t.Fatalf("failed to load updated user: %v", err)
		}

		if user.FirstName != "Mary" ||
			user.LastName != "Santos" {
			t.Fatalf(
				"updated user name = %q %q, want %q %q",
				user.FirstName,
				user.LastName,
				"Mary",
				"Santos",
			)
		}

		// The login username must remain unchanged.
		if user.Username != "mcruz" {
			t.Fatalf(
				"updated user username = %q, want %q",
				user.Username,
				"mcruz",
			)
		}
	})

	t.Run("rejects invalid names before persistence", func(t *testing.T) {
		setupStudentCreateTestDB(t)

		student := Student{
			FirstName: "Mark",
			LastName:  "Cruz",
			WeChatID:  "mark-wechat",
			Email:     "mark@example.com",
		}

		if _, err := student.Create(); err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		student.FirstName = "   "
		student.LastName = "Santos"

		if err := student.Update(); err == nil {
			t.Fatal("Update() error = nil, want validation error")
		}

		var gotStudent Student

		if err := uadmin.Get(
			&gotStudent,
			"id = ?",
			student.ID,
		); err != nil {
			t.Fatalf("failed to load student: %v", err)
		}

		if gotStudent.FirstName != "Mark" ||
			gotStudent.LastName != "Cruz" {
			t.Fatalf(
				"student changed after rejected update: %q %q",
				gotStudent.FirstName,
				gotStudent.LastName,
			)
		}

		var user uadmin.User

		if err := uadmin.Get(
			&user,
			"id = ?",
			student.UserID,
		); err != nil {
			t.Fatalf("failed to load user: %v", err)
		}

		if user.FirstName != "Mark" ||
			user.LastName != "Cruz" {
			t.Fatalf(
				"user changed after rejected update: %q %q",
				user.FirstName,
				user.LastName,
			)
		}
	})

	t.Run("rejects missing associated user", func(t *testing.T) {
		setupStudentCreateTestDB(t)

		student := Student{
			FirstName: "Mark",
			LastName:  "Cruz",
			WeChatID:  "mark-wechat",
		}

		if _, err := student.Create(); err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		var user uadmin.User

		if err := uadmin.Get(
			&user,
			"id = ?",
			student.UserID,
		); err != nil {
			t.Fatalf("failed to load associated user: %v", err)
		}

		if err := uadmin.Delete(&user); err != nil {
			t.Fatalf("failed to delete associated user: %v", err)
		}

		student.FirstName = "Mary"
		student.LastName = "Santos"

		if err := student.Update(); err == nil {
			t.Fatal("Update() error = nil, want missing-user error")
		}
	})
}
