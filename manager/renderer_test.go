package manager

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/consul-template/config"
)

func TestAtomicWrite(t *testing.T) {
	t.Run("parent_folder_missing", func(t *testing.T) {
		// Create a TempDir and a TempFile in that TempDir, then remove them to
		// "simulate" a non-existent folder
		outDir, err := ioutil.TempDir("", "")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(outDir)
		outFile, err := ioutil.TempFile(outDir, "")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.RemoveAll(outDir); err != nil {
			t.Fatal(err)
		}

		if err := AtomicWrite(outFile.Name(), nil, config.FileMode{}, false); err != nil {
			t.Fatal(err)
		}

		if _, err := os.Stat(outFile.Name()); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("retains_permissions", func(t *testing.T) {
		outDir, err := ioutil.TempDir("", "")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(outDir)
		outFile, err := ioutil.TempFile(outDir, "")
		if err != nil {
			t.Fatal(err)
		}
		os.Chmod(outFile.Name(), 0456)

		if err := AtomicWrite(outFile.Name(), nil, config.FileMode{}, false); err != nil {
			t.Fatal(err)
		}

		stat, err := os.Stat(outFile.Name())
		if err != nil {
			t.Fatal(err)
		}
		expected := os.FileMode(0456)
		if stat.Mode() != expected {
			t.Errorf("expected %q to be %q", stat.Mode(), expected)
		}
	})

	t.Run("ignores_permissions", func(t *testing.T) {
		outDir, err := ioutil.TempDir("", "")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(outDir)
		outFile, err := ioutil.TempFile(outDir, "")
		if err != nil {
			t.Fatal(err)
		}
		os.Chmod(outFile.Name(), 0456)

		if err := AtomicWrite(outFile.Name(), nil, config.NewFileMode(0654), false); err != nil {
			t.Fatal(err)
		}

		stat, err := os.Stat(outFile.Name())
		if err != nil {
			t.Fatal(err)
		}
		expected := os.FileMode(0654)
		if stat.Mode() != expected {
			t.Errorf("expected %q to be %q", stat.Mode(), expected)
		}
	})

	t.Run("default_permissions", func(t *testing.T) {
		outDir, err := ioutil.TempDir("", "")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(outDir)

		// Try AtomicWrite to a file that doesn't exist yet
		file := filepath.Join(outDir, "nope")
		if err := AtomicWrite(file, nil, config.FileMode{}, false); err != nil {
			t.Fatal(err)
		}

		stat, err := os.Stat(file)
		if err != nil {
			t.Fatal(err)
		}
		expected := os.FileMode(config.DefaultTemplateFilePerms)
		if stat.Mode() != expected {
			t.Errorf("expected %q to be %q", stat.Mode(), expected)
		}
	})

	t.Run("new_permissions", func(t *testing.T) {
		outDir, err := ioutil.TempDir("", "")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(outDir)

		// Try AtomicWrite to a file that doesn't exist yet
		file := filepath.Join(outDir, "nope")
		if err := AtomicWrite(file, nil, config.NewFileMode(0654), false); err != nil {
			t.Fatal(err)
		}

		stat, err := os.Stat(file)
		if err != nil {
			t.Fatal(err)
		}
		expected := os.FileMode(0654)
		if stat.Mode() != expected {
			t.Errorf("expected %q to be %q", stat.Mode(), expected)
		}
	})

	t.Run("backup", func(t *testing.T) {
		outDir, err := ioutil.TempDir("", "")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(outDir)
		outFile, err := ioutil.TempFile(outDir, "")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(outFile.Name(), 0456); err != nil {
			t.Fatal(err)
		}
		if _, err := outFile.Write([]byte("before")); err != nil {
			t.Fatal(err)
		}

		if err := AtomicWrite(outFile.Name(), []byte("after"), config.NewFileMode(0654), true); err != nil {
			t.Fatal(err)
		}

		f1, err := ioutil.ReadFile(outFile.Name())
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(f1, []byte("after")) {
			t.Fatalf("expected %q to be %q", f1, []byte("after"))
		}

		f2, err := ioutil.ReadFile(outFile.Name() + ".bak")
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(f2, []byte("before")) {
			t.Fatalf("expected %q to be %q", f2, []byte("before"))
		}

		stat1, err := os.Stat(outFile.Name())
		if err != nil {
			t.Fatal(err)
		}
		expected1 := os.FileMode(0654)
		if stat1.Mode() != expected1 {
			t.Errorf("expected %q to be %q", stat1.Mode(), expected1)
		}

		stat2, err := os.Stat(outFile.Name() + ".bak")
		if err != nil {
			t.Fatal(err)
		}
		expected2 := os.FileMode(0456)
		if stat2.Mode() != expected2 {
			t.Errorf("expected %q to be %q", stat2.Mode(), expected2)
		}
	})

	t.Run("backup_not_exists", func(t *testing.T) {
		outDir, err := ioutil.TempDir("", "")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(outDir)
		outFile, err := ioutil.TempFile(outDir, "")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(outFile.Name()); err != nil {
			t.Fatal(err)
		}

		if err := AtomicWrite(outFile.Name(), nil, config.FileMode{}, true); err != nil {
			t.Fatal(err)
		}

		// Shouldn't have a backup file, since the original file didn't exist
		if _, err := os.Stat(outFile.Name() + ".bak"); err == nil {
			t.Fatal("expected error")
		} else {
			if !os.IsNotExist(err) {
				t.Fatalf("bad error: %s", err)
			}
		}
	})
}
