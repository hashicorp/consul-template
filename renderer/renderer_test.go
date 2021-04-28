package renderer

import (
	"bytes"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
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

		if err := AtomicWrite(outFile.Name(), true, nil, 0644, false); err != nil {
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

		if err := AtomicWrite(outFile.Name(), true, nil, 0, false); err != nil {
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
		if err := AtomicWrite(file, true, nil, 0644, false); err != nil {
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
		if err := AtomicWrite(file, false, nil, 0644, false); err != ErrNoParentDir {
			t.Fatalf("expected %q to be %q", err, ErrNoParentDir)
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
		if err := os.Chmod(outFile.Name(), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := outFile.Write([]byte("before")); err != nil {
			t.Fatal(err)
		}

		if err := AtomicWrite(outFile.Name(), true, []byte("after"), 0644, true); err != nil {
			t.Fatal(err)
		}

		f, err := ioutil.ReadFile(outFile.Name() + ".bak")
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(f, []byte("before")) {
			t.Fatalf("expected %q to be %q", f, []byte("before"))
		}

		if stat, err := os.Stat(outFile.Name() + ".bak"); err != nil {
			t.Fatal(err)
		} else {
			if stat.Mode() != 0600 {
				t.Fatalf("expected %d to be %d", stat.Mode(), 0600)
			}
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

		if err := AtomicWrite(outFile.Name(), true, nil, 0644, true); err != nil {
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

	t.Run("backup_backup", func(t *testing.T) {
		outDir, err := ioutil.TempDir("", "")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(outDir)
		outFile, err := ioutil.TempFile(outDir, "")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := outFile.Write([]byte("first")); err != nil {
			t.Fatal(err)
		}

		contains := func(filename, content string) {
			f, err := ioutil.ReadFile(filename + ".bak")
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(f, []byte(content)) {
				t.Fatalf("expected %q to be %q", f, []byte(content))
			}
		}

		err = AtomicWrite(outFile.Name(), true, []byte("second"), 0644, true)
		if err != nil {
			t.Fatal(err)
		}
		contains(outFile.Name(), "first")

		err = AtomicWrite(outFile.Name(), true, []byte("third"), 0644, true)
		if err != nil {
			t.Fatal(err)
		}
		contains(outFile.Name(), "second")
	})
}

func TestRender(t *testing.T) {
	t.Run("file-exists-same-content", func(t *testing.T) {
		outDir, err := ioutil.TempDir("", "")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(outDir)
		outFile, err := ioutil.TempFile(outDir, "")
		if err != nil {
			t.Fatal(err)
		}
		contents := []byte("first")
		if _, err := outFile.Write(contents); err != nil {
			t.Fatal(err)
		}
		path := outFile.Name()
		if err = outFile.Close(); err != nil {
			t.Fatal(err)
		}

		rr, err := Render(&RenderInput{
			Path:     path,
			Contents: contents,
		})
		if err != nil {
			t.Fatal(err)
		}
		switch {
		case rr.WouldRender && !rr.DidRender:
		default:
			t.Fatalf("Bad render results; would: %v, did: %v",
				rr.WouldRender, rr.DidRender)
		}
	})
	t.Run("file-exists-diff-content", func(t *testing.T) {
		outDir, err := ioutil.TempDir("", "")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(outDir)
		outFile, err := ioutil.TempFile(outDir, "")
		if err != nil {
			t.Fatal(err)
		}
		contents := []byte("first")
		if _, err := outFile.Write(contents); err != nil {
			t.Fatal(err)
		}
		path := outFile.Name()
		if err = outFile.Close(); err != nil {
			t.Fatal(err)
		}

		diff_contents := []byte("not-first")
		rr, err := Render(&RenderInput{
			Path:     path,
			Contents: diff_contents,
		})
		if err != nil {
			t.Fatal(err)
		}
		switch {
		case rr.WouldRender && rr.DidRender:
		default:
			t.Fatalf("Bad render results; would: %v, did: %v",
				rr.WouldRender, rr.DidRender)
		}
	})
	t.Run("file-no-exists", func(t *testing.T) {
		outDir, err := ioutil.TempDir("", "")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(outDir)
		path := path.Join(outDir, "no-exists")
		contents := []byte("first")

		rr, err := Render(&RenderInput{
			Path:     path,
			Contents: contents,
		})
		if err != nil {
			t.Fatal(err)
		}
		switch {
		case rr.WouldRender && rr.DidRender:
		default:
			t.Fatalf("Bad render results; would: %v, did: %v",
				rr.WouldRender, rr.DidRender)
		}
	})
	t.Run("empty-file-no-exists", func(t *testing.T) {
		outDir, err := ioutil.TempDir("", "")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(outDir)
		path := path.Join(outDir, "no-exists")
		contents := []byte{}

		rr, err := Render(&RenderInput{
			Path:     path,
			Contents: contents,
		})
		if err != nil {
			t.Fatal(err)
		}
		switch {
		case rr.WouldRender && rr.DidRender:
		default:
			t.Fatalf("Bad render results; would: %v, did: %v",
				rr.WouldRender, rr.DidRender)
		}
	})
}
