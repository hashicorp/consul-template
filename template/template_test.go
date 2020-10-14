package template

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
	"time"

	dep "github.com/hashicorp/consul-template/dependency"
)

func TestNewTemplate(t *testing.T) {
	t.Parallel()
	f, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("test")
	defer os.Remove(f.Name())

	cases := []struct {
		name string
		i    *NewTemplateInput
		e    *Template
		err  bool
	}{
		{
			"nil",
			nil,
			nil,
			true,
		},
		{
			"source_and_contents",
			&NewTemplateInput{
				Source:   "source",
				Contents: "contents",
			},
			nil,
			true,
		},
		{
			"no_source_and_no_contents",
			&NewTemplateInput{},
			nil,
			true,
		},
		{
			"non_existent",
			&NewTemplateInput{
				Source: "/path/to/nope/not/once/not/never",
			},
			nil,
			true,
		},
		{
			"sets_contents_from_source",
			&NewTemplateInput{
				Source: f.Name(),
			},
			&Template{
				contents: "test",
				source:   f.Name(),
				hexMD5:   "098f6bcd4621d373cade4e832627b4f6",
			},
			false,
		},
		{
			"contents",
			&NewTemplateInput{
				Contents: "test",
			},
			&Template{
				contents: "test",
				hexMD5:   "098f6bcd4621d373cade4e832627b4f6",
			},
			false,
		},
		{
			"custom_delims",
			&NewTemplateInput{
				Contents:   "test",
				LeftDelim:  "<<",
				RightDelim: ">>",
			},
			&Template{
				contents:   "test",
				hexMD5:     "098f6bcd4621d373cade4e832627b4f6",
				leftDelim:  "<<",
				rightDelim: ">>",
			},
			false,
		},
		{
			"err_missing_key",
			&NewTemplateInput{
				Contents:      "test",
				ErrMissingKey: true,
			},
			&Template{
				contents:      "test",
				hexMD5:        "098f6bcd4621d373cade4e832627b4f6",
				errMissingKey: true,
			},
			false,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			a, err := NewTemplate(tc.i)
			if (err != nil) != tc.err {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(tc.e, a) {
				t.Errorf("\nexp: %#v\nact: %#v", tc.e, a)
			}
		})
	}
}

func TestTemplate_Execute(t *testing.T) {
	t.Parallel()
	now = func() time.Time { return time.Unix(0, 0).UTC() }

	f, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("test")
	defer os.Remove(f.Name())

	cases := []struct {
		name string
		ti   *NewTemplateInput
		i    *ExecuteInput
		e    string
		err  bool
	}{
		{
			"nil",
			&NewTemplateInput{
				Contents: `test`,
			},
			nil,
			"test",
			false,
		},
		{
			"bad_func",
			&NewTemplateInput{
				Contents: `{{ bad_func }}`,
			},
			nil,
			"",
			true,
		},
		{
			"missing_deps",
			&NewTemplateInput{
				Contents: `{{ key "foo" }}`,
			},
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"",
			false,
		},

		// missing keys
		{
			"err_missing_keys__true",
			&NewTemplateInput{
				Contents:      `{{ .Data.Foo }}`,
				ErrMissingKey: true,
			},
			nil,
			"",
			true,
		},
		{
			"err_missing_keys__false",
			&NewTemplateInput{
				Contents:      `{{ .Data.Foo }}`,
				ErrMissingKey: false,
			},
			nil,
			"<no value>",
			false,
		},

		// funcs
		{
			"func_base64Decode",
			&NewTemplateInput{
				Contents: `{{ base64Decode "aGVsbG8=" }}`,
			},
			nil,
			"hello",
			false,
		},
		{
			"func_base64Decode_bad",
			&NewTemplateInput{
				Contents: `{{ base64Decode "aGVsxxbG8=" }}`,
			},
			nil,
			"",
			true,
		},
		{
			"func_base64Encode",
			&NewTemplateInput{
				Contents: `{{ base64Encode "hello" }}`,
			},
			nil,
			"aGVsbG8=",
			false,
		},
		{
			"func_base64URLDecode",
			&NewTemplateInput{
				Contents: `{{ base64URLDecode "dGVzdGluZzEyMw==" }}`,
			},
			nil,
			"testing123",
			false,
		},
		{
			"func_base64URLDecode_bad",
			&NewTemplateInput{
				Contents: `{{ base64URLDecode "aGVsxxbG8=" }}`,
			},
			nil,
			"",
			true,
		},
		{
			"func_base64URLEncode",
			&NewTemplateInput{
				Contents: `{{ base64URLEncode "testing123" }}`,
			},
			nil,
			"dGVzdGluZzEyMw==",
			false,
		},
		// scratch
		{
			"scratch.Key",
			&NewTemplateInput{
				Contents: `{{ scratch.Set "a" "2" }}{{ scratch.Key "a" }}`,
			},
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"true",
			false,
		},
		{
			"scratch.Get",
			&NewTemplateInput{
				Contents: `{{ scratch.Set "a" "2" }}{{ scratch.Get "a" }}`,
			},
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"2",
			false,
		},
		{
			"scratch.SetX",
			&NewTemplateInput{
				Contents: `{{ scratch.SetX "a" "2" }}{{ scratch.SetX "a" "1" }}{{ scratch.Get "a" }}`,
			},
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"2",
			false,
		},
		{
			"scratch.MapSet",
			&NewTemplateInput{
				Contents: `{{ scratch.MapSet "a" "foo" "bar" }}{{ scratch.MapValues "a" }}`,
			},
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"[bar]",
			false,
		},
		{
			"scratch.MapSetX",
			&NewTemplateInput{
				Contents: `{{ scratch.MapSetX "a" "foo" "bar" }}{{ scratch.MapSetX "a" "foo" "baz" }}{{ scratch.MapValues "a" }}`,
			},
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"[bar]",
			false,
		},

		// helpers
		{
			"helper_by_key",
			&NewTemplateInput{
				Contents: `{{ range $key, $pairs := tree "list" | byKey }}{{ $key }}:{{ range $pairs }}{{ .Key }}={{ .Value }}{{ end }}{{ end }}`,
			},
			&ExecuteInput{
				Brain: func() *Brain {
					b := NewBrain()
					d, err := dep.NewKVListQuery("list")
					if err != nil {
						t.Fatal(err)
					}
					b.Remember(d, []*dep.KeyPair{
						&dep.KeyPair{Key: "", Value: ""},
						&dep.KeyPair{Key: "foo/bar", Value: "a"},
						&dep.KeyPair{Key: "zip/zap", Value: "b"},
					})
					return b
				}(),
			},
			"foo:bar=azip:zap=b",
			false,
		},
		{
			"helper_by_tag",
			&NewTemplateInput{
				Contents: `{{ range $tag, $services := service "webapp" | byTag }}{{ $tag }}:{{ range $services }}{{ .Address }}{{ end }}{{ end }}`,
			},
			&ExecuteInput{
				Brain: func() *Brain {
					b := NewBrain()
					d, err := dep.NewHealthServiceQuery("webapp")
					if err != nil {
						t.Fatal(err)
					}
					b.Remember(d, []*dep.HealthService{
						&dep.HealthService{
							Address: "1.2.3.4",
							Tags:    []string{"prod", "staging"},
						},
						&dep.HealthService{
							Address: "5.6.7.8",
							Tags:    []string{"staging"},
						},
					})
					return b
				}(),
			},
			"prod:1.2.3.4staging:1.2.3.45.6.7.8",
			false,
		},
		{
			"helper_contains",
			&NewTemplateInput{
				Contents: `{{ range service "webapp" }}{{ if .Tags | contains "prod" }}{{ .Address }}{{ end }}{{ end }}`,
			},
			&ExecuteInput{
				Brain: func() *Brain {
					b := NewBrain()
					d, err := dep.NewHealthServiceQuery("webapp")
					if err != nil {
						t.Fatal(err)
					}
					b.Remember(d, []*dep.HealthService{
						&dep.HealthService{
							Address: "1.2.3.4",
							Tags:    []string{"prod", "staging"},
						},
						&dep.HealthService{
							Address: "5.6.7.8",
							Tags:    []string{"staging"},
						},
					})
					return b
				}(),
			},
			"1.2.3.4",
			false,
		},
		{
			"helper_containsAll",
			&NewTemplateInput{
				Contents: `{{ $requiredTags := parseJSON "[\"prod\",\"us-realm\"]" }}{{ range service "webapp" }}{{ if .Tags | containsAll $requiredTags }}{{ .Address }}{{ end }}{{ end }}`,
			},
			&ExecuteInput{
				Brain: func() *Brain {
					b := NewBrain()
					d, err := dep.NewHealthServiceQuery("webapp")
					if err != nil {
						t.Fatal(err)
					}
					b.Remember(d, []*dep.HealthService{
						&dep.HealthService{
							Address: "1.2.3.4",
							Tags:    []string{"prod", "us-realm"},
						},
						&dep.HealthService{
							Address: "5.6.7.8",
							Tags:    []string{"prod", "ca-realm"},
						},
					})
					return b
				}(),
			},
			"1.2.3.4",
			false,
		},
		{
			"helper_containsAll__empty",
			&NewTemplateInput{
				Contents: `{{ $requiredTags := parseJSON "[]" }}{{ range service "webapp" }}{{ if .Tags | containsAll $requiredTags }}{{ .Address }}{{ end }}{{ end }}`,
			},
			&ExecuteInput{
				Brain: func() *Brain {
					b := NewBrain()
					d, err := dep.NewHealthServiceQuery("webapp")
					if err != nil {
						t.Fatal(err)
					}
					b.Remember(d, []*dep.HealthService{
						&dep.HealthService{
							Address: "1.2.3.4",
							Tags:    []string{"prod", "us-realm"},
						},
						&dep.HealthService{
							Address: "5.6.7.8",
							Tags:    []string{"prod", "ca-realm"},
						},
					})
					return b
				}(),
			},
			"1.2.3.45.6.7.8",
			false,
		},
		{
			"helper_containsAny",
			&NewTemplateInput{
				Contents: `{{ $acceptableTags := parseJSON "[\"v2\",\"v3\"]" }}{{ range service "webapp" }}{{ if .Tags | containsAny $acceptableTags }}{{ .Address }}{{ end }}{{ end }}`,
			},
			&ExecuteInput{
				Brain: func() *Brain {
					b := NewBrain()
					d, err := dep.NewHealthServiceQuery("webapp")
					if err != nil {
						t.Fatal(err)
					}
					b.Remember(d, []*dep.HealthService{
						&dep.HealthService{
							Address: "1.2.3.4",
							Tags:    []string{"prod", "v1"},
						},
						&dep.HealthService{
							Address: "5.6.7.8",
							Tags:    []string{"prod", "v2"},
						},
					})
					return b
				}(),
			},
			"5.6.7.8",
			false,
		},
		{
			"helper_containsAny__empty",
			&NewTemplateInput{
				Contents: `{{ $acceptableTags := parseJSON "[]" }}{{ range service "webapp" }}{{ if .Tags | containsAny $acceptableTags }}{{ .Address }}{{ end }}{{ end }}`,
			},
			&ExecuteInput{
				Brain: func() *Brain {
					b := NewBrain()
					d, err := dep.NewHealthServiceQuery("webapp")
					if err != nil {
						t.Fatal(err)
					}
					b.Remember(d, []*dep.HealthService{
						&dep.HealthService{
							Address: "1.2.3.4",
							Tags:    []string{"prod", "v1"},
						},
						&dep.HealthService{
							Address: "5.6.7.8",
							Tags:    []string{"prod", "v2"},
						},
					})
					return b
				}(),
			},
			"",
			false,
		},
		{
			"helper_containsNone",
			&NewTemplateInput{
				Contents: `{{ $forbiddenTags := parseJSON "[\"devel\",\"staging\"]" }}{{ range service "webapp" }}{{ if .Tags | containsNone $forbiddenTags }}{{ .Address }}{{ end }}{{ end }}`,
			},
			&ExecuteInput{
				Brain: func() *Brain {
					b := NewBrain()
					d, err := dep.NewHealthServiceQuery("webapp")
					if err != nil {
						t.Fatal(err)
					}
					b.Remember(d, []*dep.HealthService{
						&dep.HealthService{
							Address: "1.2.3.4",
							Tags:    []string{"prod", "v1"},
						},
						&dep.HealthService{
							Address: "5.6.7.8",
							Tags:    []string{"devel", "v2"},
						},
					})
					return b
				}(),
			},
			"1.2.3.4",
			false,
		},
		{
			"helper_containsNone__empty",
			&NewTemplateInput{
				Contents: `{{ $forbiddenTags := parseJSON "[]" }}{{ range service "webapp" }}{{ if .Tags | containsNone $forbiddenTags }}{{ .Address }}{{ end }}{{ end }}`,
			},
			&ExecuteInput{
				Brain: func() *Brain {
					b := NewBrain()
					d, err := dep.NewHealthServiceQuery("webapp")
					if err != nil {
						t.Fatal(err)
					}
					b.Remember(d, []*dep.HealthService{
						&dep.HealthService{
							Address: "1.2.3.4",
							Tags:    []string{"staging", "v1"},
						},
						&dep.HealthService{
							Address: "5.6.7.8",
							Tags:    []string{"devel", "v2"},
						},
					})
					return b
				}(),
			},
			"1.2.3.45.6.7.8",
			false,
		},
		{
			"helper_containsNotAll",
			&NewTemplateInput{
				Contents: `{{ $excludingTags := parseJSON "[\"es-v1\",\"es-v2\"]" }}{{ range service "webapp" }}{{ if .Tags | containsNotAll $excludingTags }}{{ .Address }}{{ end }}{{ end }}`,
			},
			&ExecuteInput{
				Brain: func() *Brain {
					b := NewBrain()
					d, err := dep.NewHealthServiceQuery("webapp")
					if err != nil {
						t.Fatal(err)
					}
					b.Remember(d, []*dep.HealthService{
						&dep.HealthService{
							Address: "1.2.3.4",
							Tags:    []string{"prod", "es-v1"},
						},
						&dep.HealthService{
							Address: "5.6.7.8",
							Tags:    []string{"prod", "hybrid", "es-v1", "es-v2"},
						},
					})
					return b
				}(),
			},
			"1.2.3.4",
			false,
		},
		{
			"helper_containsNotAll__empty",
			&NewTemplateInput{
				Contents: `{{ $excludingTags := parseJSON "[]" }}{{ range service "webapp" }}{{ if .Tags | containsNotAll $excludingTags }}{{ .Address }}{{ end }}{{ end }}`,
			},
			&ExecuteInput{
				Brain: func() *Brain {
					b := NewBrain()
					d, err := dep.NewHealthServiceQuery("webapp")
					if err != nil {
						t.Fatal(err)
					}
					b.Remember(d, []*dep.HealthService{
						&dep.HealthService{
							Address: "1.2.3.4",
							Tags:    []string{"prod", "es-v1"},
						},
						&dep.HealthService{
							Address: "5.6.7.8",
							Tags:    []string{"prod", "hybrid", "es-v1", "es-v2"},
						},
					})
					return b
				}(),
			},
			"",
			false,
		},
		{
			"helper_env",
			&NewTemplateInput{
				Contents: `{{ env "CT_TEST" }}`,
			},
			&ExecuteInput{
				Brain: func() *Brain {
					// Cheat and use the brain callback here to set the env.
					if err := os.Setenv("CT_TEST", "1"); err != nil {
						t.Fatal(err)
					}
					return NewBrain()
				}(),
			},
			"1",
			false,
		},
		{
			"helper_env__override",
			&NewTemplateInput{
				Contents: `{{ env "CT_TEST" }}`,
			},
			&ExecuteInput{
				Env: []string{
					"CT_TEST=2",
				},
				Brain: NewBrain(),
			},
			"2",
			false,
		},
		{
			"helper_executeTemplate",
			&NewTemplateInput{
				Contents: `{{ define "custom" }}{{ key "foo" }}{{ end }}{{ executeTemplate "custom" }}`,
			},
			&ExecuteInput{
				Brain: func() *Brain {
					b := NewBrain()
					d, err := dep.NewKVGetQuery("foo")
					if err != nil {
						t.Fatal(err)
					}
					d.EnableBlocking()
					b.Remember(d, "bar")
					return b
				}(),
			},
			"bar",
			false,
		},
		{
			"helper_executeTemplate__dot",
			&NewTemplateInput{
				Contents: `{{ define "custom" }}{{ key . }}{{ end }}{{ executeTemplate "custom" "foo" }}`,
			},
			&ExecuteInput{
				Brain: func() *Brain {
					b := NewBrain()
					d, err := dep.NewKVGetQuery("foo")
					if err != nil {
						t.Fatal(err)
					}
					d.EnableBlocking()
					b.Remember(d, "bar")
					return b
				}(),
			},
			"bar",
			false,
		},
		{
			"helper_explode",
			&NewTemplateInput{
				Contents: `{{ range $k, $v := tree "list" | explode }}{{ $k }}{{ $v }}{{ end }}`,
			},
			&ExecuteInput{
				Brain: func() *Brain {
					b := NewBrain()
					d, err := dep.NewKVListQuery("list")
					if err != nil {
						t.Fatal(err)
					}
					b.Remember(d, []*dep.KeyPair{
						&dep.KeyPair{Key: "", Value: ""},
						&dep.KeyPair{Key: "foo/bar", Value: "a"},
						&dep.KeyPair{Key: "zip/zap", Value: "b"},
					})
					return b
				}(),
			},
			"foomap[bar:a]zipmap[zap:b]",
			false,
		},
		{
			"helper_explodemap",
			&NewTemplateInput{
				Contents: `{{ scratch.MapSet "explode-test" "foo/bar" "a"}}{{ scratch.MapSet "explode-test" "qux" "c"}}{{ scratch.MapSet "explode-test" "zip/zap" "d"}}{{ scratch.Get "explode-test" | explodeMap }}`,
			},
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"map[foo:map[bar:a] qux:c zip:map[zap:d]]",
			false,
		},
		{
			"helper_in",
			&NewTemplateInput{
				Contents: `{{ range service "webapp" }}{{ if "prod" | in .Tags }}{{ .Address }}{{ end }}{{ end }}`,
			},
			&ExecuteInput{
				Brain: func() *Brain {
					b := NewBrain()
					d, err := dep.NewHealthServiceQuery("webapp")
					if err != nil {
						t.Fatal(err)
					}
					b.Remember(d, []*dep.HealthService{
						&dep.HealthService{
							Address: "1.2.3.4",
							Tags:    []string{"prod", "staging"},
						},
						&dep.HealthService{
							Address: "5.6.7.8",
							Tags:    []string{"staging"},
						},
					})
					return b
				}(),
			},
			"1.2.3.4",
			false,
		},
		{
			"helper_indent",
			&NewTemplateInput{
				Contents: `{{ "hello\nhello\r\nHELLO\r\nhello\nHELLO" | indent 4 }}`,
			},
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"    hello\n    hello\r\n    HELLO\r\n    hello\n    HELLO",
			false,
		},
		{
			"helper_indent_negative",
			&NewTemplateInput{
				Contents: `{{ "hello\nhello\r\nHELLO\r\nhello\nHELLO" | indent -4 }}`,
			},
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"    hello\n    hello\r\n    HELLO\r\n    hello\n    HELLO",
			true,
		},
		{
			"helper_indent_zero",
			&NewTemplateInput{
				Contents: `{{ "hello\nhello\r\nHELLO\r\nhello\nHELLO" | indent 0 }}`,
			},
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"hello\nhello\r\nHELLO\r\nhello\nHELLO",
			false,
		},
		{
			"helper_loop",
			&NewTemplateInput{
				Contents: `{{ range loop 3 }}1{{ end }}`,
			},
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"111",
			false,
		},
		{
			"helper_loop__i",
			&NewTemplateInput{
				Contents: `{{ range $i := loop 3 }}{{ $i }}{{ end }}`,
			},
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"012",
			false,
		},
		{
			"helper_loop_start",
			&NewTemplateInput{
				Contents: `{{ range loop 1 3 }}1{{ end }}`,
			},
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"11",
			false,
		},
		{
			"helper_loop_text",
			&NewTemplateInput{
				Contents: `{{ range loop 1 "3" }}1{{ end }}`,
			},
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"11",
			false,
		},
		{
			"helper_loop_parseInt",
			&NewTemplateInput{
				Contents: `{{ $i := print "3" | parseInt }}{{ range loop 1 $i }}1{{ end }}`,
			},
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"11",
			false,
		},
		{
			// GH-1143
			"helper_loop_var",
			&NewTemplateInput{
				Contents: `{{$n := 3 }}` +
					`{{ range $i := loop $n }}{{ $i }}{{ end }}`,
			},
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"012",
			false,
		},
		{
			"helper_join",
			&NewTemplateInput{
				Contents: `{{ "a,b,c" | split "," | join ";" }}`,
			},
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"a;b;c",
			false,
		},
		{
			"helper_parseBool",
			&NewTemplateInput{
				Contents: `{{ "true" | parseBool }}`,
			},
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"true",
			false,
		},
		{
			"helper_parseFloat",
			&NewTemplateInput{
				Contents: `{{ "1.2" | parseFloat }}`,
			},
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"1.2",
			false,
		},
		{
			"helper_parseFloat_format",
			&NewTemplateInput{
				Contents: `{{ "1.0" | parseFloat | printf "%.1f"}}`,
			},
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"1.0",
			false,
		},
		{
			"helper_parseInt",
			&NewTemplateInput{
				Contents: `{{ "-1" | parseInt }}`,
			},
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"-1",
			false,
		},
		{
			"helper_parseJSON",
			&NewTemplateInput{
				Contents: `{{ "{\"foo\": \"bar\"}" | parseJSON }}`,
			},
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"map[foo:bar]",
			false,
		},
		{
			"helper_parseUint",
			&NewTemplateInput{
				Contents: `{{ "1" | parseUint }}`,
			},
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"1",
			false,
		},
		{
			"helper_parseYAML",
			&NewTemplateInput{
				Contents: `{{ "foo: bar" | parseYAML }}`,
			},
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"map[foo:bar]",
			false,
		},
		{
			"helper_parseYAMLv2",
			&NewTemplateInput{
				Contents: `{{ "foo: bar\nbaz: \"foo\"" | parseYAML }}`,
			},
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"map[baz:foo foo:bar]",
			false,
		},
		{
			"helper_parseYAMLnested",
			&NewTemplateInput{
				Contents: `{{ "foo:\n  bar: \"baz\"\n  baz: 7" | parseYAML }}`,
			},
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"map[foo:map[bar:baz baz:7]]",
			false,
		},
		{
			"helper_plugin",
			&NewTemplateInput{
				Contents: `{{ "1" | plugin "echo" }}`,
			},
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"1",
			false,
		},
		{
			"helper_plugin_disabled",
			&NewTemplateInput{
				Contents:         `{{ "1" | plugin "echo" }}`,
				FunctionDenylist: []string{"plugin"},
			},
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"",
			true,
		},
		{
			"helper_regexMatch",
			&NewTemplateInput{
				Contents: `{{ "foo" | regexMatch "[a-z]+" }}`,
			},
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"true",
			false,
		},
		{
			"helper_regexReplaceAll",
			&NewTemplateInput{
				Contents: `{{ "foo" | regexReplaceAll "\\w" "x" }}`,
			},
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"xxx",
			false,
		},
		{
			"helper_replaceAll",
			&NewTemplateInput{
				Contents: `{{ "hello my hello" | regexReplaceAll "hello" "bye" }}`,
			},
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"bye my bye",
			false,
		},
		{
			"helper_split",
			&NewTemplateInput{
				Contents: `{{ "a,b,c" | split "," }}`,
			},
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"[a b c]",
			false,
		},
		{
			"helper_timestamp",
			&NewTemplateInput{
				Contents: `{{ timestamp }}`,
			},
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"1970-01-01T00:00:00Z",
			false,
		},
		{
			"helper_helper_timestamp__formatted",
			&NewTemplateInput{
				Contents: `{{ timestamp "2006-01-02" }}`,
			},
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"1970-01-01",
			false,
		},
		{
			"helper_toJSON",
			&NewTemplateInput{
				Contents: `{{ "a,b,c" | split "," | toJSON }}`,
			},
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"[\"a\",\"b\",\"c\"]",
			false,
		},
		{
			"helper_toLower",
			&NewTemplateInput{
				Contents: `{{ "HI" | toLower }}`,
			},
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"hi",
			false,
		},
		{
			"helper_toTitle",
			&NewTemplateInput{
				Contents: `{{ "this is a sentence" | toTitle }}`,
			},
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"This Is A Sentence",
			false,
		},
		{
			"helper_toTOML",
			&NewTemplateInput{
				Contents: `{{ "{\"foo\":\"bar\"}" | parseJSON | toTOML }}`,
			},
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"foo = \"bar\"",
			false,
		},
		{
			"helper_toUpper",
			&NewTemplateInput{
				Contents: `{{ "hi" | toUpper }}`,
			},
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"HI",
			false,
		},
		{
			"helper_toYAML",
			&NewTemplateInput{
				Contents: `{{ "{\"foo\":\"bar\"}" | parseJSON | toYAML }}`,
			},
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"foo: bar",
			false,
		},
		{
			"helper_trimSpace",
			&NewTemplateInput{
				Contents: `{{ "\t hi\n " | trimSpace }}`,
			},
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"hi",
			false,
		},
		{
			"helper_sockaddr",
			&NewTemplateInput{
				Contents: `{{ sockaddr "GetAllInterfaces | include \"flag\" \"loopback\" | include \"type\" \"IPv4\" | sort \"address\" | limit 1 | attr \"address\""}}`,
			},
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"127.0.0.1",
			false,
		},
		{
			"math_add",
			&NewTemplateInput{
				Contents: `{{ 2 | add 2 }}`,
			},
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"4",
			false,
		},
		{
			"math_subtract",
			&NewTemplateInput{
				Contents: `{{ 2 | subtract 2 }}`,
			},
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"0",
			false,
		},
		{
			"math_multiply",
			&NewTemplateInput{
				Contents: `{{ 2 | multiply 2 }}`,
			},
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"4",
			false,
		},
		{
			"math_divide",
			&NewTemplateInput{
				Contents: `{{ 2 | divide 2 }}`,
			},
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"1",
			false,
		},
		{
			"math_modulo",
			&NewTemplateInput{
				Contents: `{{ 3 | modulo 2 }}`,
			},
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"1",
			false,
		},
		{
			"math_minimum",
			&NewTemplateInput{
				Contents: `{{ 3 | minimum 2 }}`,
			},
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"2",
			false,
		},
		{
			"math_maximum",
			&NewTemplateInput{
				Contents: `{{ 3 | maximum 2 }}`,
			},
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"3",
			false,
		},
	}

	//	struct {
	//		name string
	//		ti   *NewTemplateInput
	//		i    *ExecuteInput
	//		e    string
	//		err  bool
	//	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			tpl, err := NewTemplate(tc.ti)
			if err != nil {
				t.Fatal(err)
			}

			a, err := tpl.Execute(tc.i)
			if (err != nil) != tc.err {
				t.Fatal(err)
			}
			if a != nil && !bytes.Equal([]byte(tc.e), a.Output) {
				t.Errorf("\nexp: %#v\nact: %#v", tc.e, string(a.Output))
			}
		})
	}
}
