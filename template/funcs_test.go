package template

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/pkg/errors"
)

func TestFileSandbox(t *testing.T) {
	cwd, _ := os.Getwd()
	cases := []struct {
		name         string
		sandbox      string
		path         string
		expectedPath string
		expectedErr  error
	}{
		{
			"absolute_path_no_sandbox",
			"",
			"/path/to/file",
			"/path/to/file",
			nil,
		},
		{
			"relative_path_no_sandbox",
			"",
			"./path/to/file",
			filepath.Join(cwd, "path/to/file"),
			nil,
		},
		{
			"absolute_path_with_sandbox",
			"/sandbox",
			"/path/to/file",
			"/sandbox/path/to/file",
			nil,
		},
		{
			"relative_path_in_sandbox",
			"/sandbox",
			"./path/to/file",
			"/sandbox/path/to/file",
			nil,
		},
		{
			"relative_path_escaping_sandbox",
			"/sandbox",
			"/path/../../../to/file",
			"",
			errors.New("'/path/../../../to/file' is outside of sandbox"),
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			result, err := sandboxedPath(tc.sandbox, tc.path)
			if tc.expectedErr != nil {
				if err == nil {
					t.Fatalf("expected error %s got nil", tc.expectedErr)
				}
				if err.Error() != tc.expectedErr.Error() {
					t.Fatalf("expected %s got %s", tc.expectedErr, err)
				}
			}
			if result != tc.expectedPath {
				t.Fatalf("expected %s got %s", tc.expectedPath, result)
			}
		})
	}
}
