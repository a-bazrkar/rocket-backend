package errcore

import "testing"

func TestShortFunc(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{
			in:   "github.com/a-bazrkar/rocket-backend/internal/repository.(*Repository).GetByID",
			want: "Repository.GetByID",
		},
		{
			in:   "github.com/a-bazrkar/rocket-backend/internal/repository.GetByID",
			want: "GetByID",
		},
	}
	for _, tc := range cases {
		if got := shortFunc(tc.in); got != tc.want {
			t.Fatalf("shortFunc(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
