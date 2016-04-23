package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	dep "github.com/hashicorp/consul-template/dependency"
	"github.com/hashicorp/consul-template/test"
	"github.com/hashicorp/consul/testutil"
	"github.com/hashicorp/go-gatedio"
)

func TestNewRunner_initialize(t *testing.T) {
	in1 := test.CreateTempfile(nil, t)
	defer test.DeleteTempfile(in1, t)

	in2 := test.CreateTempfile(nil, t)
	defer test.DeleteTempfile(in2, t)

	in3 := test.CreateTempfile(nil, t)
	defer test.DeleteTempfile(in3, t)

	dry, once := true, true
	config := DefaultConfig()
	config.Merge(&Config{
		ConfigTemplates: []*ConfigTemplate{
			&ConfigTemplate{Source: in1.Name(), Command: "1", CommandTimeout: 1 * time.Second},
			&ConfigTemplate{Source: in1.Name(), Command: "1.1", CommandTimeout: 1 * time.Second},
			&ConfigTemplate{Source: in2.Name(), Command: "2", CommandTimeout: 1 * time.Second},
			&ConfigTemplate{Source: in3.Name(), Command: "3", CommandTimeout: 1 * time.Second},
		},
	})

	runner, err := NewRunner(config, dry, once, &sync.RWMutex{})
	if err != nil {
		t.Fatal(err)
	}

	if runner.dry != dry {
		t.Errorf("expected %#v to be %#v", runner.dry, dry)
	}

	if runner.once != once {
		t.Errorf("expected %#v to be %#v", runner.once, once)
	}

	if runner.watcher == nil {
		t.Errorf("expected %#v to be %#v", runner.watcher, nil)
	}

	if num := len(runner.templates); num != 3 {
		t.Errorf("expected %d to be %d", len(runner.templates), 3)
	}

	if runner.renderedTemplates == nil {
		t.Errorf("expected %#v to be %#v", runner.renderedTemplates, nil)
	}

	if num := len(runner.ctemplatesMap); num != 3 {
		t.Errorf("expected %d to be %d", len(runner.ctemplatesMap), 3)
	}

	ctemplates := runner.ctemplatesMap[in1.Name()]
	if num := len(ctemplates); num != 2 {
		t.Errorf("expected %d to be %d", len(ctemplates), 2)
	}

	if runner.outStream != os.Stdout {
		t.Errorf("expected %#v to be %#v", runner.outStream, os.Stdout)
	}

	if runner.errStream != os.Stderr {
		t.Errorf("expected %#v to be %#v", runner.errStream, os.Stderr)
	}

	brain := NewBrain()
	if !reflect.DeepEqual(runner.brain, brain) {
		t.Errorf("expected %#v to be %#v", runner.brain, brain)
	}

	if runner.ErrCh == nil {
		t.Errorf("expected %#v to be %#v", runner.ErrCh, nil)
	}

	if runner.DoneCh == nil {
		t.Errorf("expected %#v to be %#v", runner.DoneCh, nil)
	}
}

func TestNewRunner_badTemplate(t *testing.T) {
	config := DefaultConfig()
	config.Merge(&Config{
		ConfigTemplates: []*ConfigTemplate{
			&ConfigTemplate{Source: "/not/a/real/path"},
		},
	})

	if _, err := NewRunner(config, false, false, &sync.RWMutex{}); err == nil {
		t.Fatal("expected error, but nothing was returned")
	}
}

func TestReceive_addsToBrain(t *testing.T) {
	runner, err := NewRunner(DefaultConfig(), false, false, &sync.RWMutex{})
	if err != nil {
		t.Fatal(err)
	}

	d, err := dep.ParseStoreKey("foo")
	if err != nil {
		t.Fatal(err)
	}
	data := "some value"
	runner.dependencies[d.HashCode()] = d
	runner.Receive(d, data)

	value, ok := runner.brain.Recall(d)
	if !ok {
		t.Fatalf("expected brain to have data")
	}
	if data != value {
		t.Errorf("expected %q to be %q", data, value)
	}
}

func TestReceive_storesBrain(t *testing.T) {
	runner, err := NewRunner(DefaultConfig(), false, false, &sync.RWMutex{})
	if err != nil {
		t.Fatal(err)
	}

	d, data := &dep.File{}, "this is some data"
	runner.dependencies[d.HashCode()] = d
	runner.Receive(d, data)

	if _, ok := runner.brain.Recall(d); !ok {
		t.Errorf("expected brain to have data")
	}
}

func TestReceive_doesNotStoreIfNotWatching(t *testing.T) {
	runner, err := NewRunner(DefaultConfig(), false, false, &sync.RWMutex{})
	if err != nil {
		t.Fatal(err)
	}

	d, data := &dep.File{}, "this is some data"
	runner.Receive(d, data)

	if _, ok := runner.brain.Recall(d); ok {
		t.Errorf("expected brain to not have data")
	}
}

func TestRun_noopIfMissingData(t *testing.T) {
	in := test.CreateTempfile([]byte(`
    {{ range service "consul@nyc1" }}{{ end }}
  `), t)
	defer test.DeleteTempfile(in, t)

	config := DefaultConfig()
	config.Merge(&Config{
		ConfigTemplates: []*ConfigTemplate{
			&ConfigTemplate{Source: in.Name()},
		},
	})

	runner, err := NewRunner(config, false, false, &sync.RWMutex{})
	if err != nil {
		t.Fatal(err)
	}

	buff := gatedio.NewByteBuffer()
	runner.outStream, runner.errStream = buff, buff

	if err := runner.Run(); err != nil {
		t.Fatal(err)
	}

	if num := len(buff.Bytes()); num != 0 {
		t.Errorf("expected %d to be %d", num, 0)
	}
}

func TestRun_dry(t *testing.T) {
	in := test.CreateTempfile([]byte(`
    {{ range service "consul@nyc1" }}{{.Node}}{{ end }}
  `), t)
	defer test.DeleteTempfile(in, t)

	config := DefaultConfig()
	config.Merge(&Config{
		ConfigTemplates: []*ConfigTemplate{
			&ConfigTemplate{
				Source:      in.Name(),
				Destination: "/out/file.txt",
			},
		},
	})

	runner, err := NewRunner(config, true, false, &sync.RWMutex{})
	if err != nil {
		t.Fatal(err)
	}

	d, err := dep.ParseHealthServices("consul@nyc1")
	if err != nil {
		t.Fatal(err)
	}
	data := []*dep.HealthService{
		&dep.HealthService{Node: "consul1"},
		&dep.HealthService{Node: "consul2"},
	}
	runner.dependencies[d.HashCode()] = d
	runner.watcher.ForceWatching(d, true)
	runner.Receive(d, data)

	buff := gatedio.NewByteBuffer()
	runner.outStream, runner.errStream = buff, buff

	if err := runner.Run(); err != nil {
		t.Fatal(err)
	}

	actual := bytes.TrimSpace(buff.Bytes())
	expected := bytes.TrimSpace([]byte(`
    > /out/file.txt

    consul1consul2
  `))
	if !bytes.Equal(actual, expected) {
		t.Errorf("expected \n%q\n to equal \n%q\n", actual, expected)
	}
}

func TestRun_singlePass(t *testing.T) {
	in := test.CreateTempfile([]byte(`
    {{ range service "consul@nyc1"}}{{ end }}
    {{ range service "consul@nyc2"}}{{ end }}
    {{ range service "consul@nyc3"}}{{ end }}
  `), t)
	defer test.DeleteTempfile(in, t)

	config := DefaultConfig()
	config.Merge(&Config{
		ConfigTemplates: []*ConfigTemplate{
			&ConfigTemplate{Source: in.Name()},
		},
	})

	runner, err := NewRunner(config, true, false, &sync.RWMutex{})
	if err != nil {
		t.Fatal(err)
	}

	if len(runner.dependencies) != 0 {
		t.Errorf("expected %d to be %d", len(runner.dependencies), 0)
	}

	if err := runner.Run(); err != nil {
		t.Fatal(err)
	}

	if len(runner.dependencies) != 3 {
		t.Errorf("expected %d to be %d", len(runner.dependencies), 3)
	}
}

func TestRun_singlePassDuplicates(t *testing.T) {
	in := test.CreateTempfile([]byte(`
    {{ range service "consul@nyc1"}}{{ end }}
    {{ range service "consul@nyc1"}}{{ end }}
    {{ range service "consul@nyc1"}}{{ end }}
    {{ range service "consul@nyc2"}}{{ end }}
    {{ range service "consul@nyc2"}}{{ end }}
    {{ range service "consul@nyc3"}}{{ end }}
    {{ range service "consul@nyc3"}}{{ end }}
  `), t)
	defer test.DeleteTempfile(in, t)

	config := DefaultConfig()
	config.Merge(&Config{
		ConfigTemplates: []*ConfigTemplate{
			&ConfigTemplate{Source: in.Name()},
		},
	})

	runner, err := NewRunner(config, true, false, &sync.RWMutex{})
	if err != nil {
		t.Fatal(err)
	}

	if len(runner.dependencies) != 0 {
		t.Errorf("expected %d to be %d", len(runner.dependencies), 0)
	}

	if err := runner.Run(); err != nil {
		t.Fatal(err)
	}

	if len(runner.dependencies) != 3 {
		t.Errorf("expected %d to be %d", len(runner.dependencies), 3)
	}
}

func TestRun_doublePass(t *testing.T) {
	in := test.CreateTempfile([]byte(`
		{{ range ls "services" }}
			{{ range service .Key }}
				{{.Node}} {{.Address}}:{{.Port}}
			{{ end }}
		{{ end }}
  `), t)
	defer test.DeleteTempfile(in, t)

	config := DefaultConfig()
	config.Merge(&Config{
		ConfigTemplates: []*ConfigTemplate{
			&ConfigTemplate{Source: in.Name()},
		},
	})

	runner, err := NewRunner(config, true, false, &sync.RWMutex{})
	if err != nil {
		t.Fatal(err)
	}

	if len(runner.dependencies) != 0 {
		t.Errorf("expected %d to be %d", len(runner.dependencies), 0)
	}

	if err := runner.Run(); err != nil {
		t.Fatal(err)
	}

	if len(runner.dependencies) != 1 {
		t.Errorf("expected %d to be %d", len(runner.dependencies), 1)
	}

	d, err := dep.ParseStoreKeyPrefix("services")
	if err != nil {
		t.Fatal(err)
	}
	data := []*dep.KeyPair{
		&dep.KeyPair{Key: "service1"},
		&dep.KeyPair{Key: "service2"},
		&dep.KeyPair{Key: "service3"},
	}
	runner.Receive(d, data)

	if err := runner.Run(); err != nil {
		t.Fatal(err)
	}

	if len(runner.dependencies) != 4 {
		t.Errorf("expected %d to be %d", len(runner.dependencies), 4)
	}
}

func TestRun_removesUnusedDependencies(t *testing.T) {
	in := test.CreateTempfile([]byte(nil), t)
	defer test.DeleteTempfile(in, t)

	config := DefaultConfig()
	config.Merge(&Config{
		ConfigTemplates: []*ConfigTemplate{
			&ConfigTemplate{Source: in.Name()},
		},
	})

	runner, err := NewRunner(config, true, false, &sync.RWMutex{})
	if err != nil {
		t.Fatal(err)
	}

	d, err := dep.ParseHealthServices("consul@nyc2")
	if err != nil {
		t.Fatal(err)
	}

	runner.dependencies = map[string]dep.Dependency{"consul@nyc2": d}

	if err := runner.Run(); err != nil {
		t.Fatal(err)
	}

	if len(runner.dependencies) != 0 {
		t.Errorf("expected %d to be %d", len(runner.dependencies), 0)
	}

	if runner.watcher.Watching(d) {
		t.Errorf("expected watcher to stop watching dependency")
	}

	if _, ok := runner.brain.Recall(d); ok {
		t.Errorf("expected brain to forget dependency")
	}
}

func TestRun_multipleTemplatesRunsCommands(t *testing.T) {
	in1 := test.CreateTempfile([]byte(`
    {{ range service "consul@nyc1" }}{{.Node}}{{ end }}
  `), t)
	defer test.DeleteTempfile(in1, t)

	in2 := test.CreateTempfile([]byte(`
    {{range service "consul@nyc2"}}{{.Node}}{{ end }}
  `), t)
	defer test.DeleteTempfile(in2, t)

	out1 := test.CreateTempfile(nil, t)
	test.DeleteTempfile(out1, t)

	out2 := test.CreateTempfile(nil, t)
	test.DeleteTempfile(out2, t)

	touch1, err := ioutil.TempFile(os.TempDir(), "touch1-")
	if err != nil {
		t.Fatal(err)
	}
	os.Remove(touch1.Name())
	defer os.Remove(touch1.Name())

	touch2, err := ioutil.TempFile(os.TempDir(), "touch2-")
	if err != nil {
		t.Fatal(err)
	}
	os.Remove(touch2.Name())
	defer os.Remove(touch2.Name())

	config := DefaultConfig()
	config.Merge(&Config{
		ConfigTemplates: []*ConfigTemplate{
			&ConfigTemplate{
				Source:         in1.Name(),
				Destination:    out1.Name(),
				Command:        fmt.Sprintf("touch %s", touch1.Name()),
				CommandTimeout: 1 * time.Second,
			},
			&ConfigTemplate{
				Source:         in2.Name(),
				Destination:    out2.Name(),
				Command:        fmt.Sprintf("touch %s", touch2.Name()),
				CommandTimeout: 1 * time.Second,
			},
		},
	})

	runner, err := NewRunner(config, false, false, &sync.RWMutex{})
	if err != nil {
		t.Fatal(err)
	}

	d, err := dep.ParseHealthServices("consul@nyc1")
	if err != nil {
		t.Fatal(err)
	}
	data := []*dep.HealthService{
		&dep.HealthService{Node: "consul1"},
		&dep.HealthService{Node: "consul2"},
	}
	runner.dependencies[d.HashCode()] = d
	runner.watcher.ForceWatching(d, true)
	runner.Receive(d, data)

	if err := runner.Run(); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(touch1.Name()); err != nil {
		t.Errorf("expected first command to run, but did not: %s", err)
	}

	if _, err := os.Stat(touch2.Name()); err == nil {
		t.Errorf("expected second command to not run, but touch exists")
	}
}

func TestRunner_quiescence(t *testing.T) {
	template := &Template{}

	// Basic min case.
	{
		ch := make(chan *Template, 1)
		q := newQuiescence(ch,
			50*time.Millisecond, 250*time.Millisecond, template)

		// Should not fire until we tick() it.
		select {
		case <-ch:
			t.Fatalf("q should not have fired")

		case <-time.After(2 * q.max):
		}

		// Tick it once and make sure it fires after the min time.
		start := time.Now()
		q.tick()
		select {
		case <-ch:
			dur := time.Now().Sub(start)
			if dur < q.min || dur > 2*q.min {
				t.Fatalf("bad duration %9.6f", dur.Seconds())
			}

		case <-time.After(2 * q.min):
			t.Fatalf("q should have fired")
		}
	}

	// Single snooze case.
	{
		ch := make(chan *Template, 1)
		q := newQuiescence(ch,
			50*time.Millisecond, 250*time.Millisecond, template)

		// Tick once with a small delay to simulate an update coming
		// in.
		q.tick()
		time.Sleep(q.min / 2)

		// Tick again to make sure it waits the full min amount.
		start := time.Now()
		q.tick()
		select {
		case <-ch:
			dur := time.Now().Sub(start)
			if dur < q.min || dur > 2*q.min {
				t.Fatalf("bad duration %9.6f", dur.Seconds())
			}

		case <-time.After(2 * q.min):
			t.Fatalf("q should have fired")
		}
	}

	// Max time case.
	{
		ch := make(chan *Template, 1)
		q := newQuiescence(ch,
			50*time.Millisecond, 250*time.Millisecond, template)

		// Keep ticking to service the min timer and make sure we get
		// cut off at the max time.
		fired := false
		start := time.Now()
		for !fired && time.Now().Sub(start) < 2*q.max {
			q.tick()
			time.Sleep(q.min / 2)
			select {
			case <-ch:
				fired = true
			default:
			}
		}

		if !fired {
			t.Fatalf("q should have fired")
		}
	}
}

// Warning: this is a super fragile and time-dependent test. If it's failing,
// check the demo Consul cluster and your own sanity before you assume your
// code broke something...
func TestRunner_quiescenceIntegrated(t *testing.T) {
	consul := testutil.NewTestServerConfig(t, func(c *testutil.TestServerConfig) {
		c.Stdout = ioutil.Discard
		c.Stderr = ioutil.Discard
	})
	defer consul.Stop()

	in := test.CreateTempfile([]byte(`
    {{ range service "consul" "any" }}{{.Node}}{{ end }}
  `), t)
	defer test.DeleteTempfile(in, t)

	out := test.CreateTempfile(nil, t)
	test.DeleteTempfile(out, t)

	config := testConfig(fmt.Sprintf(`
		consul = "%s"
		wait = "50ms:200ms"

		template {
			source = "%s"
			destination = "%s"
		}
	`, consul.HTTPAddr, in.Name(), out.Name()), t)

	runner, err := NewRunner(config, false, false, &sync.RWMutex{})
	if err != nil {
		t.Fatal(err)
	}

	go runner.Start()
	defer runner.Stop()

	min := time.After(10 * time.Millisecond)
	max := time.After(500 * time.Millisecond)
	for {
		select {
		case <-min:
			if _, err = os.Stat(out.Name()); !os.IsNotExist(err) {
				t.Errorf("expected quiescence timer to not fire for yet")
			}
			continue
		case <-max:
			if _, err = os.Stat(out.Name()); os.IsNotExist(err) {
				t.Errorf("expected template to be rendered by now")
			}
			return
		}
	}
}

func TestRender_sameContentsDoesNotExecuteCommand(t *testing.T) {
	outFile := test.CreateTempfile(nil, t)
	os.Remove(outFile.Name())
	defer os.Remove(outFile.Name())

	inTemplate := test.CreateTempfile([]byte(`
    {{ range service "consul@nyc1" }}{{.Node}}{{ end }}
  `), t)
	defer test.DeleteTempfile(inTemplate, t)

	outTemplate := test.CreateTempfile([]byte(`
    consul1consul2
  `), t)
	defer test.DeleteTempfile(outTemplate, t)

	config := DefaultConfig()
	config.Merge(&Config{
		ConfigTemplates: []*ConfigTemplate{
			&ConfigTemplate{
				Source:      inTemplate.Name(),
				Destination: outTemplate.Name(),
				Command:     fmt.Sprintf("echo 'foo' > %s", outFile.Name()),
			},
		},
	})

	runner, err := NewRunner(config, false, false, &sync.RWMutex{})
	if err != nil {
		t.Fatal(err)
	}

	d, err := dep.ParseHealthServices("consul@nyc1")
	if err != nil {
		t.Fatal(err)
	}
	data := []*dep.HealthService{
		&dep.HealthService{Node: "consul1"},
		&dep.HealthService{Node: "consul2"},
	}
	runner.Receive(d, data)

	if err := runner.Run(); err != nil {
		t.Fatal(err)
	}

	_, err = os.Stat(outFile.Name())
	if !os.IsNotExist(err) {
		t.Fatalf("expected command to not be run")
	}
}

func TestAtomicWrite_parentFolderMissing(t *testing.T) {
	// Create a TempDir and a TempFile in that TempDir, then remove them to
	// "simulate" a non-existent folder
	outDir, err := ioutil.TempDir(os.TempDir(), "")
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

	if err := atomicWrite(outFile.Name(), nil, 0644, false); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(outFile.Name()); err != nil {
		t.Fatal(err)
	}
}

func TestAtomicWrite_retainsPermissions(t *testing.T) {
	outDir, err := ioutil.TempDir(os.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(outDir)
	outFile, err := ioutil.TempFile(outDir, "")
	if err != nil {
		t.Fatal(err)
	}
	os.Chmod(outFile.Name(), 0644)

	if err := atomicWrite(outFile.Name(), nil, 0644, false); err != nil {
		t.Fatal(err)
	}

	stat, err := os.Stat(outFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	expected := os.FileMode(0644)
	if stat.Mode() != expected {
		t.Errorf("expected %q to be %q", stat.Mode(), expected)
	}
}

func TestAtomicWrite_nonExistent(t *testing.T) {
	// Create a temp dir
	outDir, err := ioutil.TempDir(os.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(outDir)

	// Try atomicWrite to a file that doesn't exist yet
	file := filepath.Join(outDir, "nope")
	if err := atomicWrite(file, nil, 0644, false); err != nil {
		t.Fatal(err)
	}

	// File was created
	if _, err := os.Stat(file); err != nil {
		t.Fatal(err)
	}
}

func TestAtomicWrite_backup(t *testing.T) {
	outDir, err := ioutil.TempDir(os.TempDir(), "")
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

	if err := atomicWrite(outFile.Name(), []byte("after"), 0644, true); err != nil {
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
}

func TestAtomicWrite_backupNoExist(t *testing.T) {
	outDir, err := ioutil.TempDir(os.TempDir(), "")
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

	if err := atomicWrite(outFile.Name(), nil, 0644, true); err != nil {
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
}

func TestRun_doesNotExecuteCommandMissingDependencies(t *testing.T) {
	outFile := test.CreateTempfile(nil, t)
	os.Remove(outFile.Name())
	defer os.Remove(outFile.Name())

	inTemplate := test.CreateTempfile([]byte(`
    {{ range service "consul@nyc1"}}{{ end }}
  `), t)
	defer test.DeleteTempfile(inTemplate, t)

	outTemplate := test.CreateTempfile(nil, t)
	defer test.DeleteTempfile(outTemplate, t)

	config := DefaultConfig()
	config.Merge(&Config{
		ConfigTemplates: []*ConfigTemplate{
			&ConfigTemplate{
				Source:      inTemplate.Name(),
				Destination: outTemplate.Name(),
				Command:     fmt.Sprintf("echo 'foo' > %s", outFile.Name()),
			},
		},
	})

	runner, err := NewRunner(config, false, false, &sync.RWMutex{})
	if err != nil {
		t.Fatal(err)
	}

	if err := runner.Run(); err != nil {
		t.Fatal(err)
	}

	_, err = os.Stat(outFile.Name())
	if !os.IsNotExist(err) {
		t.Fatalf("expected command to not be run")
	}
}

func TestRun_executesCommand(t *testing.T) {
	outFile := test.CreateTempfile(nil, t)
	os.Remove(outFile.Name())
	defer os.Remove(outFile.Name())

	inTemplate := test.CreateTempfile([]byte(`
    {{ range service "consul@nyc1"}}{{ end }}
  `), t)
	defer test.DeleteTempfile(inTemplate, t)

	outTemplate := test.CreateTempfile(nil, t)
	defer test.DeleteTempfile(outTemplate, t)

	config := DefaultConfig()
	config.Merge(&Config{
		ConfigTemplates: []*ConfigTemplate{
			&ConfigTemplate{
				Source:         inTemplate.Name(),
				Destination:    outTemplate.Name(),
				Command:        fmt.Sprintf("echo 'foo' > %s", outFile.Name()),
				CommandTimeout: 1 * time.Second,
			},
		},
	})

	runner, err := NewRunner(config, false, false, &sync.RWMutex{})
	if err != nil {
		t.Fatal(err)
	}

	d, err := dep.ParseHealthServices("consul@nyc1")
	if err != nil {
		t.Fatal(err)
	}
	data := []*dep.HealthService{
		&dep.HealthService{
			Node:    "consul",
			Address: "1.2.3.4",
			ID:      "consul@nyc1",
			Name:    "consul",
		},
	}
	runner.dependencies[d.HashCode()] = d
	runner.watcher.ForceWatching(d, true)
	runner.Receive(d, data)

	if err := runner.Run(); err != nil {
		t.Fatal(err)
	}

	_, err = os.Stat(outFile.Name())
	if err != nil {
		t.Fatal(err)
	}
}

func TestRun_doesNotExecuteCommandMoreThanOnce(t *testing.T) {
	outFile := test.CreateTempfile(nil, t)
	os.Remove(outFile.Name())
	defer os.Remove(outFile.Name())

	inTemplate := test.CreateTempfile([]byte(`
    {{ range service "consul@nyc1"}}{{ end }}
  `), t)
	defer test.DeleteTempfile(inTemplate, t)

	outTemplateA := test.CreateTempfile(nil, t)
	defer test.DeleteTempfile(outTemplateA, t)

	outTemplateB := test.CreateTempfile(nil, t)
	defer test.DeleteTempfile(outTemplateB, t)

	config := DefaultConfig()
	config.Merge(&Config{
		ConfigTemplates: []*ConfigTemplate{
			&ConfigTemplate{
				Source:         inTemplate.Name(),
				Destination:    outTemplateA.Name(),
				Command:        fmt.Sprintf("echo 'foo' >> %s", outFile.Name()),
				CommandTimeout: 1 * time.Second,
			},
			&ConfigTemplate{
				Source:         inTemplate.Name(),
				Destination:    outTemplateB.Name(),
				Command:        fmt.Sprintf("echo 'foo' >> %s", outFile.Name()),
				CommandTimeout: 1 * time.Second,
			},
		},
	})

	runner, err := NewRunner(config, false, false, &sync.RWMutex{})
	if err != nil {
		t.Fatal(err)
	}

	d, err := dep.ParseHealthServices("consul@nyc1")
	if err != nil {
		t.Fatal(err)
	}
	data := []*dep.HealthService{
		&dep.HealthService{
			Node:    "consul",
			Address: "1.2.3.4",
			ID:      "consul@nyc1",
			Name:    "consul",
		},
	}
	runner.dependencies[d.HashCode()] = d
	runner.watcher.ForceWatching(d, true)
	runner.Receive(d, data)

	if err := runner.Run(); err != nil {
		t.Fatal(err)
	}

	_, err = os.Stat(outFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	output, err := ioutil.ReadFile(outFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	if strings.Count(string(output), "foo") > 1 {
		t.Fatalf("expected command to be run once.")
	}
}

func TestRunner_pidCreate(t *testing.T) {
	pidfile := test.CreateTempfile(nil, t)
	os.Remove(pidfile.Name())
	defer os.Remove(pidfile.Name())

	config := testConfig(fmt.Sprintf(`
		pid_file = "%s"
	`, pidfile.Name()), t)

	runner, err := NewRunner(config, false, false, &sync.RWMutex{})
	if err != nil {
		t.Fatal(err)
	}

	go runner.Start()
	defer runner.Stop()

	select {
	case err := <-runner.ErrCh:
		t.Fatal(err)
	case <-time.After(100 * time.Millisecond):
	}

	_, err = os.Stat(pidfile.Name())
	if err != nil {
		t.Fatal("expected pidfile to exist")
	}
}

func TestRunner_pidDelete(t *testing.T) {
	pidfile := test.CreateTempfile(nil, t)
	os.Remove(pidfile.Name())
	defer os.Remove(pidfile.Name())

	config := testConfig(fmt.Sprintf(`
		pid_file = "%s"
	`, pidfile.Name()), t)

	runner, err := NewRunner(config, false, false, &sync.RWMutex{})
	if err != nil {
		t.Fatal(err)
	}

	go runner.Start()

	select {
	case err := <-runner.ErrCh:
		t.Fatal(err)
	case <-time.After(100 * time.Millisecond):
	}

	_, err = os.Stat(pidfile.Name())
	if err != nil {
		t.Fatal("expected pidfile to exist")
	}

	runner.Stop()
	select {
	case err := <-runner.ErrCh:
		t.Fatal(err)
	case <-runner.DoneCh:
	}

	_, err = os.Stat(pidfile.Name())
	if err == nil {
		t.Fatal("expected pidfile to be cleaned up")
	}
	if !os.IsNotExist(err) {
		t.Fatal(err)
	}
}

func TestRunner_onceAlreadyRenderedDoesNotHangOrRunCommands(t *testing.T) {
	outFile := test.CreateTempfile(nil, t)
	os.Remove(outFile.Name())
	defer os.Remove(outFile.Name())

	out := test.CreateTempfile([]byte("redis"), t)
	defer os.Remove(out.Name())

	in := test.CreateTempfile([]byte(`{{ key "service_name"}}`), t)
	defer test.DeleteTempfile(in, t)

	outTemplateA := test.CreateTempfile(nil, t)
	defer test.DeleteTempfile(outTemplateA, t)

	config := DefaultConfig()
	config.Merge(&Config{
		ConfigTemplates: []*ConfigTemplate{
			&ConfigTemplate{
				Source:      in.Name(),
				Destination: out.Name(),
				Command:     fmt.Sprintf("echo 'foo' >> %s", outFile.Name()),
			},
		},
	})

	runner, err := NewRunner(config, false, true, &sync.RWMutex{})
	if err != nil {
		t.Fatal(err)
	}

	d, err := dep.ParseStoreKey("service_name")
	if err != nil {
		t.Fatal(err)
	}
	data := "redis"
	runner.dependencies[d.HashCode()] = d
	runner.watcher.ForceWatching(d, true)
	runner.Receive(d, data)

	go runner.Start()

	select {
	case err := <-runner.ErrCh:
		t.Fatal(err)
	case <-runner.DoneCh:
	case <-time.After(5 * time.Millisecond):
		t.Fatal("runner should have stopped")
		runner.Stop()
	}

	_, err = os.Stat(outFile.Name())
	if err == nil {
		t.Fatal("expected command to not be run")
	}
	if !os.IsNotExist(err) {
		t.Fatal(err)
	}
}

func TestExecute_setsEnv(t *testing.T) {
	tmpfile := test.CreateTempfile(nil, t)
	defer test.DeleteTempfile(tmpfile, t)

	config := testConfig(`
		consul = "1.2.3.4:5678"
		token = "abcd1234"

		vault {
			address = "5.6.7.8:1234"
			ssl {
				verify = false
			}
		}

		auth {
			username = "username"
			password = "password"
		}

		ssl {
			enabled = true
			verify = false
		}
	`, t)

	runner, err := NewRunner(config, false, false, &sync.RWMutex{})
	if err != nil {
		t.Fatal(err)
	}

	command := fmt.Sprintf("env > %s", tmpfile.Name())
	if err := runner.execute(command, 1*time.Second); err != nil {
		t.Fatal(err)
	}

	bytes, err := ioutil.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	contents := string(bytes)

	if !strings.Contains(contents, "CONSUL_HTTP_ADDR=1.2.3.4:5678") {
		t.Errorf("expected env to contain CONSUL_HTTP_ADDR")
	}

	if !strings.Contains(contents, "CONSUL_HTTP_TOKEN=abcd1234") {
		t.Errorf("expected env to contain CONSUL_HTTP_TOKEN")
	}

	if !strings.Contains(contents, "CONSUL_HTTP_AUTH=username:password") {
		t.Errorf("expected env to contain CONSUL_HTTP_AUTH")
	}

	if !strings.Contains(contents, "CONSUL_HTTP_SSL=true") {
		t.Errorf("expected env to contain CONSUL_HTTP_SSL")
	}

	if !strings.Contains(contents, "CONSUL_HTTP_SSL_VERIFY=false") {
		t.Errorf("expected env to contain CONSUL_HTTP_SSL_VERIFY")
	}

	if !strings.Contains(contents, "VAULT_ADDR=5.6.7.8:1234") {
		t.Errorf("expected env to contain VAULT_ADDR")
	}

	if !strings.Contains(contents, "VAULT_SKIP_VERIFY=true") {
		t.Errorf("expected env to contain VAULT_SKIP_VERIFY")
	}
}

func TestExecute_timeout(t *testing.T) {
	tmpfile := test.CreateTempfile(nil, t)
	defer test.DeleteTempfile(tmpfile, t)

	config := testConfig(`
		consul = "1.2.3.4:5678"
		token = "abcd1234"

		auth {
			username = "username"
			password = "password"
		}

		ssl {
			enabled = true
			verify = false
		}
	`, t)

	runner, err := NewRunner(config, false, false, &sync.RWMutex{})
	if err != nil {
		t.Fatal(err)
	}

	err = runner.execute("sleep 10", 100*time.Millisecond)
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "did not return for 100ms"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q to include %q", err.Error(), expected)
	}
}

func TestRunner_dedup(t *testing.T) {
	t.Parallel()

	// Create a template
	in := test.CreateTempfile([]byte(`
    {{ range service "consul" }}{{.Node}}{{ end }}
  `), t)
	defer test.DeleteTempfile(in, t)

	out1 := test.CreateTempfile(nil, t)
	defer test.DeleteTempfile(out1, t)

	out2 := test.CreateTempfile(nil, t)
	defer test.DeleteTempfile(out2, t)

	// Start consul
	consul := testutil.NewTestServerConfig(t, func(c *testutil.TestServerConfig) {
		c.Stdout = ioutil.Discard
		c.Stderr = ioutil.Discard
	})
	defer consul.Stop()

	// Setup the runner config
	config := DefaultConfig()
	config.Merge(&Config{
		ConfigTemplates: []*ConfigTemplate{
			&ConfigTemplate{Source: in.Name(), Destination: out1.Name()},
		},
	})
	config.Deduplicate.Enabled = true
	config.Consul = consul.HTTPAddr
	config.set("consul")
	config.set("deduplicate")
	config.set("deduplicate.enabled")

	config2 := DefaultConfig()
	config2.Merge(&Config{
		ConfigTemplates: []*ConfigTemplate{
			&ConfigTemplate{Source: in.Name(), Destination: out2.Name()},
		},
	})
	config2.Deduplicate.Enabled = true
	config2.Consul = consul.HTTPAddr
	config2.set("consul")
	config2.set("deduplicate")
	config2.set("deduplicate.enabled")

	// Create the runners
	var reapLock sync.RWMutex
	r1, err := NewRunner(config, false, false, &reapLock)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	go r1.Start()
	defer r1.Stop()

	r2, err := NewRunner(config2, false, false, &reapLock)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	go r2.Start()
	defer r2.Stop()

	// Wait until the output file exists
	testutil.WaitForResult(func() (bool, error) {
		_, err := os.Stat(out1.Name())
		if err != nil {
			return false, nil
		}
		return true, nil
	}, func(err error) {
		t.Fatalf("err: %v", err)
	})

	// Wait until the output file exists
	testutil.WaitForResult(func() (bool, error) {
		_, err := os.Stat(out2.Name())
		if err != nil {
			return false, nil
		}
		return true, nil
	}, func(err error) {
		t.Fatalf("err: %v", err)
	})

	// Should only be a single total watcher
	total := r1.watcher.Size() + r2.watcher.Size()
	if total > 1 {
		t.Fatalf("too many watchers: %d", total)
	}
}
