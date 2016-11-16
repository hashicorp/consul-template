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
		d, err := dep.ParseStoreKey("foo")
		if err != nil {
			t.Fatal(err)
		}
		data := "bar"
		r.dependencies[d.HashCode()] = d
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
		d, err := dep.ParseStoreKey("zip")
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

				d, err := dep.ParseStoreKey("foo")
				if err != nil {
					t.Fatal(err)
				}
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
				d, err := dep.ParseStoreKey("foo")
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
				r.config.Consul = config.String("1.2.3.4")
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
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
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

func TestRunner_quiescence(t *testing.T) {
	tpl := &template.Template{}

	t.Run("min", func(t *testing.T) {
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
