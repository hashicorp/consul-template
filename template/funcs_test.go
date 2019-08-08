package template

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/pkg/errors"
)

func TestFileSandbox(t *testing.T) {
	// while most of the function can be tested lexigraphically,
	// we need to be able to walk actual symlinks.
	_, filename, _, _ := runtime.Caller(0)
	sandboxDir := filepath.Join(filepath.Dir(filename), "testdata", "sandbox")
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
			sandboxDir,
			"/path/to/file",
			filepath.Join(sandboxDir, "path/to/file"),
			nil,
		},
		{
			"relative_path_in_sandbox",
			sandboxDir,
			"./path/to/file",
			filepath.Join(sandboxDir, "path/to/file"),
			nil,
		},
		{
			"symlink_path_in_sandbox",
			sandboxDir,
			"./path/to/ok-symlink",
			filepath.Join(sandboxDir, "path/to/ok-symlink"),
			nil,
		},
		{
			"relative_path_escaping_sandbox",
			sandboxDir,
			"/path/../../../funcs_test.go",
			"",
			errors.New("'/path/../../../funcs_test.go' is outside of sandbox"),
		},
		{
			"symlink_escaping_sandbox",
			sandboxDir,
			"/path/to/bad-symlink",
			"",
			errors.New("'/path/to/bad-symlink' is outside of sandbox"),
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
