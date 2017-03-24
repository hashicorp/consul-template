package manager

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/consul-template/config"
	dep "github.com/hashicorp/consul-template/dependency"
	"github.com/hashicorp/consul-template/template"
)

func TestRunner_Receive(t *testing.T) {
	t.Parallel()

	r, err := NewRunner(config.DefaultConfig(), true, true)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("adds_to_brain", func(t *testing.T) {
		t.Parallel()

		d, err := dep.NewKVGetQuery("foo")
		if err != nil {
			t.Fatal(err)
		}
		data := "bar"
		r.dependenciesLock.Lock()
		r.dependencies[d.String()] = d
		r.dependenciesLock.Unlock()
		r.Receive(d, data)

		val, ok := r.brain.Recall(d)
		if !ok {
			t.Fatalf("expected brain to have data")
		}
		if data != val {
			t.Errorf("\nexp: %#v\nact: %#v", data, val)
		}
	})

	t.Run("skips_brain_if_not_watching", func(t *testing.T) {
		t.Parallel()

		d, err := dep.NewKVGetQuery("zip")
		if err != nil {
			t.Fatal(err)
		}
		r.Receive(d, "")

		if _, ok := r.brain.Recall(d); ok {
			t.Fatalf("expected brain to not have data")
		}
	})
}

func TestRunner_Run(t *testing.T) {
	cases := []struct {
		name   string
		before func(*testing.T, *Runner)
		c      *config.Config
		after  func(*testing.T, *Runner, string)
		err    bool
	}{
		{
			"missing_deps",
			nil,
			&config.Config{
				Templates: &config.TemplateConfigs{
					&config.TemplateConfig{
						Contents: config.String(`{{ service "consul@nyc1" }}`),
					},
				},
			},
			func(t *testing.T, r *Runner, out string) {
				exp := ""
				if out != exp {
					t.Errorf("\nexp: %#v\nact: %#v", exp, out)
				}
			},
			false,
		},
		{
			"dry",
			nil,
			&config.Config{
				Templates: &config.TemplateConfigs{
					&config.TemplateConfig{
						Contents:    config.String(`hello`),
						Destination: config.String("/foo/bar"),
					},
				},
			},
			func(t *testing.T, r *Runner, out string) {
				if _, err := os.Stat("/foo/bar"); err == nil {
					t.Errorf("expected file to not exist")
				}
				exp := "> /foo/bar\nhello"
				if out != exp {
					t.Errorf("\nexp: %#v\nact: %#v", exp, out)
				}
			},
			false,
		},
		{
			"accumulates_deps",
			nil,
			&config.Config{
				Templates: &config.TemplateConfigs{
					&config.TemplateConfig{
						Contents: config.String(`{{ key "foo" }}{{ key "bar" }}`),
					},
				},
			},
			func(t *testing.T, r *Runner, out string) {
				exp := 2
				if len(r.dependencies) != exp {
					t.Errorf("\nexp: %#v\nact: %#v\ndeps: %#v", exp, len(r.dependencies), r.dependencies)
				}
			},
			false,
		},
		{
			"no_duplicate_deps",
			nil,
			&config.Config{
				Templates: &config.TemplateConfigs{
					&config.TemplateConfig{
						Contents: config.String(`{{ key "foo" }}{{ key "foo" }}`),
					},
				},
			},
			func(t *testing.T, r *Runner, out string) {
				exp := 1
				if len(r.dependencies) != exp {
					t.Errorf("\nexp: %#v\nact: %#v\ndeps: %#v", exp, len(r.dependencies), r.dependencies)
				}
			},
			false,
		},
		{
			"multipass",
			nil,
			&config.Config{
				Templates: &config.TemplateConfigs{
					&config.TemplateConfig{
						Contents: config.String(`{{ key (key "foo") }}`),
					},
				},
			},
			func(t *testing.T, r *Runner, out string) {
				exp := 1
				if len(r.dependencies) != exp {
					t.Errorf("\nexp: %#v\nact: %#v\ndeps: %#v", exp, len(r.dependencies), r.dependencies)
				}

				d, err := dep.NewKVGetQuery("foo")
				if err != nil {
					t.Fatal(err)
				}
				d.EnableBlocking()
				r.Receive(d, "bar")

				if err := r.Run(); err != nil {
					t.Fatal(err)
				}

				exp = 2
				if len(r.dependencies) != exp {
					t.Errorf("\nexp: %#v\nact: %#v\ndeps: %#v", exp, len(r.dependencies), r.dependencies)
				}
			},
			false,
		},
		{
			"remove_unused",
			func(t *testing.T, r *Runner) {
				d, err := dep.NewKVGetQuery("foo")
				if err != nil {
					t.Fatal(err)
				}
				r.Receive(d, "bar")
			},
			&config.Config{
				Templates: &config.TemplateConfigs{
					&config.TemplateConfig{
						Contents: config.String("hello"),
					},
				},
			},
			func(t *testing.T, r *Runner, out string) {
				exp := 0
				if len(r.dependencies) != exp {
					t.Errorf("\nexp: %#v\nact: %#v\ndeps: %#v", exp, len(r.dependencies), r.dependencies)
				}
			},
			false,
		},
		{
			"runs_commands",
			func(t *testing.T, r *Runner) {
				r.dry = false
			},
			&config.Config{
				Templates: &config.TemplateConfigs{
					&config.TemplateConfig{
						Contents:    config.String("hello"),
						Command:     config.String("echo 123"),
						Destination: config.String("/tmp/ct-runs_commands_a"),
					},
					&config.TemplateConfig{
						Contents:    config.String("world"),
						Command:     config.String("echo 456"),
						Destination: config.String("/tmp/ct-runs_commands_b"),
					},
				},
			},
			func(t *testing.T, r *Runner, out string) {
				exp := "123\n456\n"
				if out != exp {
					t.Errorf("\nexp: %#v\nact: %#v", exp, out)
				}

				select {
				case <-r.TemplateRenderedCh():
				case <-time.After(1 * time.Second):
					t.Fatalf("A template should have rendered")
				}

				times := r.RenderEvents()
				if l := len(times); l != 2 {
					t.Errorf("\nexp: %#v\nact: %#v", 2, l)
				}

				os.Remove("/tmp/ct-runs_commands_a")
				os.Remove("/tmp/ct-runs_commands_b")
			},
			false,
		},
		{
			"no_command_if_same_template",
			func(t *testing.T, r *Runner) {
				r.dry = false
				if err := ioutil.WriteFile("/tmp/ct-no_command_if_same_template", []byte("hello"), 0644); err != nil {
					t.Fatal(err)
				}
			},
			&config.Config{
				Templates: &config.TemplateConfigs{
					&config.TemplateConfig{
						Contents:    config.String("hello"),
						Command:     config.String("echo 123"),
						Destination: config.String("/tmp/ct-no_command_if_same_template"),
					},
				},
			},
			func(t *testing.T, r *Runner, out string) {
				exp := ""
				if out != exp {
					t.Errorf("\nexp: %#v\nact: %#v", exp, out)
				}
				os.Remove("/tmp/ct-no_command_if_same_template_a")
			},
			false,
		},
		{
			"no_duplicate_commands",
			func(t *testing.T, r *Runner) {
				r.dry = false
			},
			&config.Config{
				Templates: &config.TemplateConfigs{
					&config.TemplateConfig{
						Contents:    config.String("hello"),
						Command:     config.String("echo 123"),
						Destination: config.String("/tmp/ct-no_duplicate_commands_a"),
					},
					&config.TemplateConfig{
						Contents:    config.String("world"),
						Command:     config.String("echo 123"),
						Destination: config.String("/tmp/ct-no_duplicate_commands_b"),
					},
				},
			},
			func(t *testing.T, r *Runner, out string) {
				// There should only be one 123, even though the command was there
				// twice
				exp := "123\n"
				if out != exp {
					t.Errorf("\nexp: %#v\nact: %#v", exp, out)
				}
				os.Remove("/tmp/ct-no_duplicate_commands_a")
				os.Remove("/tmp/ct-no_duplicate_commands_b")
			},
			false,
		},
		{
			"env",
			func(t *testing.T, r *Runner) {
				r.dry = false
				r.config.Consul.Address = config.String("1.2.3.4")
			},
			&config.Config{
				Templates: &config.TemplateConfigs{
					&config.TemplateConfig{
						Contents:    config.String("hello"),
						Command:     config.String("env"),
						Destination: config.String("/tmp/ct-env_a"),
					},
				},
			},
			func(t *testing.T, r *Runner, out string) {
				exp := "CONSUL_HTTP_ADDR=1.2.3.4"
				if !strings.Contains(out, exp) {
					t.Errorf("\nexp: %#v\nact: %#v", exp, out)
				}
				os.Remove("/tmp/ct-env_a")
			},
			false,
		},
	}

	for i, tc := range cases {
		tc := tc
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			t.Parallel()

			var out bytes.Buffer

			c := config.DefaultConfig().Merge(tc.c)
			c.Finalize()

			r, err := NewRunner(c, true, true)
			if err != nil {
				t.Fatal(err)
			}
			r.outStream, r.errStream = &out, &out
			defer r.Stop()

			if tc.before != nil {
				tc.before(t, r)
			}

			if err := r.Run(); (err != nil) != tc.err {
				t.Fatal(err)
			}

			if tc.after != nil {
				tc.after(t, r, out.String())
			}
		})
	}
}

func TestRunner_Start(t *testing.T) {
	t.Parallel()

	t.Run("store_pid", func(t *testing.T) {
		t.Parallel()

		pid, err := ioutil.TempFile("", "")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(pid.Name())

		out, err := ioutil.TempFile("", "")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(out.Name())

		c := config.DefaultConfig().Merge(&config.Config{
			PidFile: config.String(pid.Name()),
			Templates: &config.TemplateConfigs{
				&config.TemplateConfig{
					Contents:    config.String("test"),
					Destination: config.String(out.Name()),
				},
			},
		})
		c.Finalize()

		r, err := NewRunner(c, false, false)
		if err != nil {
			t.Fatal(err)
		}

		go r.Start()
		defer r.Stop()

		select {
		case err := <-r.ErrCh:
			t.Fatal(err)
		case <-r.renderedCh:
			c, err := ioutil.ReadFile(pid.Name())
			if err != nil {
				t.Fatal(err)
			}
			if l := len(c); l == 0 {
				t.Errorf("\nexp: %#v\nact: %#v", "> 0", l)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("timeout")
		}
	})

	t.Run("run_no_deps", func(t *testing.T) {
		t.Parallel()

		out, err := ioutil.TempFile("", "")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(out.Name())

		c := config.DefaultConfig().Merge(&config.Config{
			Templates: &config.TemplateConfigs{
				&config.TemplateConfig{
					Contents:    config.String(`test`),
					Destination: config.String(out.Name()),
				},
			},
		})
		c.Finalize()

		r, err := NewRunner(c, false, false)
		if err != nil {
			t.Fatal(err)
		}

		go r.Start()
		defer r.Stop()

		select {
		case err := <-r.ErrCh:
			t.Fatal(err)
		case <-r.renderedCh:
			act, err := ioutil.ReadFile(out.Name())
			if err != nil {
				t.Fatal(err)
			}
			exp := "test"
			if exp != string(act) {
				t.Errorf("\nexp: %#v\nact: %#v", exp, string(act))
			}
		case <-time.After(2 * time.Second):
			t.Fatal("timeout")
		}
	})

	t.Run("single_dependency", func(t *testing.T) {
		t.Parallel()

		testConsul.SetKVString(t, "single-dep-foo", "bar")

		out, err := ioutil.TempFile("", "")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(out.Name())

		c := config.DefaultConfig().Merge(&config.Config{
			Consul: &config.ConsulConfig{
				Address: config.String(testConsul.HTTPAddr),
			},
			Templates: &config.TemplateConfigs{
				&config.TemplateConfig{
					Contents:    config.String(`{{ key "single-dep-foo" }}`),
					Destination: config.String(out.Name()),
				},
			},
		})
		c.Finalize()

		r, err := NewRunner(c, false, false)
		if err != nil {
			t.Fatal(err)
		}

		go r.Start()
		defer r.Stop()

		select {
		case err := <-r.ErrCh:
			t.Fatal(err)
		case <-r.renderedCh:
			act, err := ioutil.ReadFile(out.Name())
			if err != nil {
				t.Fatal(err)
			}
			exp := "bar"
			if exp != string(act) {
				t.Errorf("\nexp: %#v\nact: %#v", exp, string(act))
			}
		case <-time.After(2 * time.Second):
			t.Fatal("timeout")
		}
	})

	t.Run("multipass", func(t *testing.T) {
		t.Parallel()

		testConsul.SetKVString(t, "multipass-foo", "multipass-bar")
		testConsul.SetKVString(t, "multipass-bar", "zip")

		out, err := ioutil.TempFile("", "")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(out.Name())

		c := config.DefaultConfig().Merge(&config.Config{
			Consul: &config.ConsulConfig{
				Address: config.String(testConsul.HTTPAddr),
			},
			Templates: &config.TemplateConfigs{
				&config.TemplateConfig{
					Contents:    config.String(`{{ key (key "multipass-foo") }}`),
					Destination: config.String(out.Name()),
				},
			},
		})
		c.Finalize()

		r, err := NewRunner(c, false, false)
		if err != nil {
			t.Fatal(err)
		}

		go r.Start()
		defer r.Stop()

		select {
		case err := <-r.ErrCh:
			t.Fatal(err)
		case <-r.renderedCh:
			act, err := ioutil.ReadFile(out.Name())
			if err != nil {
				t.Fatal(err)
			}
			exp := "zip"
			if exp != string(act) {
				t.Errorf("\nexp: %#v\nact: %#v", exp, string(act))
			}
		case <-time.After(2 * time.Second):
			t.Fatal("timeout")
		}
	})

	t.Run("exec", func(t *testing.T) {
		t.Parallel()

		out, err := ioutil.TempFile("", "")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(out.Name())

		c := config.DefaultConfig().Merge(&config.Config{
			Exec: &config.ExecConfig{
				Command: config.String(`sleep 30`),
			},
			Templates: &config.TemplateConfigs{
				&config.TemplateConfig{
					Contents:    config.String(`test`),
					Destination: config.String(out.Name()),
				},
			},
		})
		c.Finalize()

		r, err := NewRunner(c, false, false)
		if err != nil {
			t.Fatal(err)
		}

		go r.Start()
		defer r.Stop()

		select {
		case err := <-r.ErrCh:
			t.Fatal(err)
		case <-r.renderedCh:
			found := false
			for i := 0; i < 5; i++ {
				if found {
					break
				}

				time.Sleep(100 * time.Millisecond)

				r.childLock.RLock()
				if r.child != nil {
					found = true
				}
				r.childLock.RUnlock()
			}
			if !found {
				t.Error("missing child")
			}
		case <-time.After(2 * time.Second):
			t.Fatal("timeout")
		}
	})

	t.Run("exec_once", func(t *testing.T) {
		t.Parallel()

		out, err := ioutil.TempFile("", "")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(out.Name())

		c := config.DefaultConfig().Merge(&config.Config{
			Exec: &config.ExecConfig{
				Command: config.String(`sleep 30`),
			},
			Templates: &config.TemplateConfigs{
				&config.TemplateConfig{
					Contents:    config.String(`test`),
					Destination: config.String(out.Name()),
				},
			},
		})
		c.Finalize()

		r, err := NewRunner(c, false, true)
		if err != nil {
			t.Fatal(err)
		}

		go r.Start()
		defer r.Stop()

		select {
		case err := <-r.ErrCh:
			t.Fatal(err)
		case <-r.renderedCh:
			// Just assert there is no panic
		case <-time.After(2 * time.Second):
			t.Fatal("timeout")
		}
	})
}

func TestRunner_quiescence(t *testing.T) {
	t.Parallel()

	tpl := &template.Template{}

	t.Run("min", func(t *testing.T) {
		t.Parallel()

		ch := make(chan *template.Template, 1)
		q := newQuiescence(ch,
			50*time.Millisecond, 250*time.Millisecond, tpl)

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
	})

	// Single snooze case.
	t.Run("snooze", func(t *testing.T) {
		t.Parallel()

		ch := make(chan *template.Template, 1)
		q := newQuiescence(ch,
			50*time.Millisecond, 250*time.Millisecond, tpl)

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
	})

	// Max time case.
	t.Run("max", func(t *testing.T) {
		t.Parallel()

		ch := make(chan *template.Template, 1)
		q := newQuiescence(ch,
			50*time.Millisecond, 250*time.Millisecond, tpl)

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
	})
}
