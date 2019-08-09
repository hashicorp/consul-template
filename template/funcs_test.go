package template

import (
	"fmt"
	"path/filepath"
	"runtime"
	"testing"
)

func TestFileSandbox(t *testing.T) {
	// while most of the function can be tested lexigraphically,
	// we need to be able to walk actual symlinks.
	_, filename, _, _ := runtime.Caller(0)
	sandboxDir := filepath.Join(filepath.Dir(filename), "testdata", "sandbox")
	cases := []struct {
		name     string
		sandbox  string
		path     string
		expected error
	}{
		{
			"absolute_path_no_sandbox",
			"",
			"/path/to/file",
			nil,
		},
		{
			"relative_path_no_sandbox",
			"",
			"./path/to/file",
			nil,
		},
		{
			"absolute_path_with_sandbox",
			sandboxDir,
			filepath.Join(sandboxDir, "path/to/file"),
			nil,
		},
		{
			"relative_path_in_sandbox",
			sandboxDir,
			filepath.Join(sandboxDir, "path/to/../to/file"),
			nil,
		},
		{
			"symlink_path_in_sandbox",
			sandboxDir,
			filepath.Join(sandboxDir, "path/to/ok-symlink"),
			nil,
		},
		{
			"relative_path_escaping_sandbox",
			sandboxDir,
			filepath.Join(sandboxDir, "path/../../../funcs_test.go"),
			fmt.Errorf("'%s' is outside of sandbox",
				filepath.Join(sandboxDir, "path/../../../funcs_test.go")),
		},
		{
			"symlink_escaping_sandbox",
			sandboxDir,
			filepath.Join(sandboxDir, "path/to/bad-symlink"),
			fmt.Errorf("'%s' is outside of sandbox",
				filepath.Join(sandboxDir, "path/to/bad-symlink")),
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			err := pathInSandbox(tc.sandbox, tc.path)
			if err != nil && tc.expected != nil {
				if err.Error() != tc.expected.Error() {
					t.Fatalf("expected %v got %v", tc.expected, err)
				}
			} else if err != tc.expected {
				t.Fatalf("expected %v got %v", tc.expected, err)
			}
		})
	}
}
