package errors

import "testing"

func TestAsAppErrorMappings(t *testing.T) {
	cases := []struct {
		name string
		err  error
		code Code
	}{
		{"validation-prefix", assertErr("VALIDATION:email required"), CodeValidationError},
		{"conflict-prefix", assertErr("CONFLICT:duplicate"), CodeConflict},
		{"unauthorized", assertErr("UNAUTHORIZED"), CodeUnauthorized},
		{"forbidden", assertErr("FORBIDDEN"), CodeForbidden},
		{"not-found", assertErr("NOT_FOUND"), CodeNotFound},
		{"unknown", assertErr("some random"), CodeInternalError},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := AsAppError(tc.err)
			if got.Code != tc.code {
				t.Fatalf("expected %s got %s", tc.code, got.Code)
			}
		})
	}
}

type staticErr string

func (e staticErr) Error() string { return string(e) }

func assertErr(msg string) error { return staticErr(msg) }
