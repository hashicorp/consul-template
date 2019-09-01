package renderer

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"testing"
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

		if err := AtomicWrite(outFile.Name(), true, nil, 0644, false, 0); err != nil {
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
		os.Chmod(outFile.Name(), 0600)

		if err := AtomicWrite(outFile.Name(), true, nil, 0, false, 0); err != nil {
			t.Fatal(err)
		}

		stat, err := os.Stat(outFile.Name())
		if err != nil {
			t.Fatal(err)
		}

		expected := os.FileMode(0600)
		if stat.Mode() != expected {
			t.Errorf("expected %q to be %q", stat.Mode(), expected)
		}
	})

	t.Run("non_existent", func(t *testing.T) {
		outDir, err := ioutil.TempDir("", "")
		if err != nil {
			t.Fatal(err)
		}
		os.RemoveAll(outDir)
		defer os.RemoveAll(outDir)

		// Try AtomicWrite to a file that doesn't exist yet
		file := filepath.Join(outDir, "nope/not/it/create")
		if err := AtomicWrite(file, true, nil, 0644, false, 0); err != nil {
			t.Fatal(err)
		}

		if _, err := os.Stat(file); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("non_existent_no_create", func(t *testing.T) {
		outDir, err := ioutil.TempDir("", "")
		if err != nil {
			t.Fatal(err)
		}
		os.RemoveAll(outDir)
		defer os.RemoveAll(outDir)

		// Try AtomicWrite to a file that doesn't exist yet
		file := filepath.Join(outDir, "nope/not/it/nope-no-create")
		if err := AtomicWrite(file, false, nil, 0644, false, 0); err != ErrNoParentDir {
			t.Fatalf("expected %q to be %q", err, ErrNoParentDir)
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

		if err := AtomicWrite(outFile.Name(), true, nil, 0644, true, 1); err != nil {
			t.Fatal(err)
		}

		backups, err := filepath.Glob(outFile.Name() + ".*.bak")
		if err != nil {
			t.Fatal(err.Error())
		}
		if len(backups) != 0 {
			t.Fatalf("file %s should not exists", backups)
		}
	})
}

func TestBackup(t *testing.T) {
	for maxBackup := 1; maxBackup <= 10; maxBackup++ {
		outDir, err := ioutil.TempDir("", "")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(outDir)
		outFile, err := ioutil.TempFile(outDir, "")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(outFile.Name(), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := outFile.Write([]byte("backup0")); err != nil {
			t.Fatal(err)
		}
		for i := 1; i < 20; i++ {
			if err := AtomicWrite(outFile.Name(), true, []byte("backup"+strconv.Itoa(i)), 0644, true, maxBackup); err != nil {
				t.Fatal(err)
			}
			backups, err := filepath.Glob(outFile.Name() + ".*.bak")
			if err != nil {
				t.Fatal(err)
			}
			if i <= maxBackup {
				if backups == nil || len(backups) != i {
					t.Fatalf("expected %d backup file exists, but %d", i, len(backups))
				}
				for j := 0; j < i; j++ {
					f, err := ioutil.ReadFile("/" + backups[j])
					if err != nil {
						t.Fatal(err)
					}
					if !bytes.Equal(f, []byte("backup"+strconv.Itoa(j))) {
						t.Fatalf("expected %q to be %q", f, []byte("backup"+strconv.Itoa(j)))
					}
				}
			} else {
				if backups == nil || len(backups) != maxBackup {
					t.Fatalf("expected %d backup file exists, but %d", maxBackup, len(backups))
				}
				for j := 0; j < maxBackup; j++ {
					f, err := ioutil.ReadFile(backups[j])
					if err != nil {
						t.Fatal(err)
					}
					if !bytes.Equal(f, []byte("backup"+strconv.Itoa(i-maxBackup+j))) {
						t.Fatalf("expected %q to be %q", f, []byte("backup"+strconv.Itoa(i-maxBackup+j)))
					}
				}
			}
		}
	}
}
