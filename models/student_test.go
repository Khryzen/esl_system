package models

import "testing"

func TestStudentUsername(t *testing.T) {
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
			want:      "MCruz",
		},
		{
			name:      "first name with spaces",
			firstName: "Mary Ann",
			lastName:  "Cruz",
			want:      "MCruz",
		},
		{
			name:      "last name with spaces",
			firstName: "Mark",
			lastName:  "De La Cruz",
			want:      "MDeLaCruz",
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
