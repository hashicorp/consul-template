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
	now = func() time.Time { return time.Unix(0, 0).UTC() }

	f, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("test")
	defer os.Remove(f.Name())

	cases := []struct {
		name string
		c    string
		i    *ExecuteInput
		e    string
		err  bool
	}{
		{
			"nil",
			`test`,
			nil,
			"test",
			false,
		},
		{
			"bad_func",
			`{{ bad_func }}`,
			nil,
			"",
			true,
		},
		{
			"missing_deps",
			`{{ key "foo" }}`,
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"",
			false,
		},

		// funcs
		{
			"func_datacenters",
			`{{ datacenters }}`,
			&ExecuteInput{
				Brain: func() *Brain {
					b := NewBrain()
					d, err := dep.NewCatalogDatacentersQuery()
					if err != nil {
						t.Fatal(err)
					}
					b.Remember(d, []string{"dc1", "dc2"})
					return b
				}(),
			},
			"[dc1 dc2]",
			false,
		},
		{
			"func_file",
			`{{ file "/path/to/file" }}`,
			&ExecuteInput{
				Brain: func() *Brain {
					b := NewBrain()
					d, err := dep.NewFileQuery("/path/to/file")
					if err != nil {
						t.Fatal(err)
					}
					b.Remember(d, "content")
					return b
				}(),
			},
			"content",
			false,
		},
		{
			"func_key",
			`{{ key "key" }}`,
			&ExecuteInput{
				Brain: func() *Brain {
					b := NewBrain()
					d, err := dep.NewKVGetQuery("key")
					if err != nil {
						t.Fatal(err)
					}
					d.EnableBlocking()
					b.Remember(d, "5")
					return b
				}(),
			},
			"5",
			false,
		},
		{
			"func_keyExists",
			`{{ keyExists "key" }} {{ keyExists "no_key" }}`,
			&ExecuteInput{
				Brain: func() *Brain {
					b := NewBrain()
					d, err := dep.NewKVGetQuery("key")
					if err != nil {
						t.Fatal(err)
					}
					b.Remember(d, true)
					return b
				}(),
			},
			"true false",
			false,
		},
		{
			"func_keyOrDefault",
			`{{ keyOrDefault "key" "100" }} {{ keyOrDefault "no_key" "200" }}`,
			&ExecuteInput{
				Brain: func() *Brain {
					b := NewBrain()
					d, err := dep.NewKVGetQuery("key")
					if err != nil {
						t.Fatal(err)
					}
					b.Remember(d, "150")
					return b
				}(),
			},
			"150 200",
			false,
		},
		{
			"func_ls",
			`{{ range ls "list" }}{{ .Key }}={{ .Value }}{{ end }}`,
			&ExecuteInput{
				Brain: func() *Brain {
					b := NewBrain()
					d, err := dep.NewKVListQuery("list")
					if err != nil {
						t.Fatal(err)
					}
					b.Remember(d, []*dep.KeyPair{
						&dep.KeyPair{Key: "", Value: ""},
						&dep.KeyPair{Key: "foo", Value: "bar"},
						&dep.KeyPair{Key: "foo/zip", Value: "zap"},
					})
					return b
				}(),
			},
			"foo=bar",
			false,
		},
		{
			"func_node",
			`{{ with node }}{{ .Node.Node }}{{ range .Services }}{{ .Service }}{{ end }}{{ end }}`,
			&ExecuteInput{
				Brain: func() *Brain {
					b := NewBrain()
					d, err := dep.NewCatalogNodeQuery("")
					if err != nil {
						t.Fatal(err)
					}
					b.Remember(d, &dep.CatalogNode{
						Node: &dep.Node{Node: "node1"},
						Services: []*dep.CatalogNodeService{
							&dep.CatalogNodeService{
								Service: "service1",
							},
						},
					})
					return b
				}(),
			},
			"node1service1",
			false,
		},
		{
			"func_nodes",
			`{{ range nodes }}{{ .Node }}{{ end }}`,
			&ExecuteInput{
				Brain: func() *Brain {
					b := NewBrain()
					d, err := dep.NewCatalogNodesQuery("")
					if err != nil {
						t.Fatal(err)
					}
					b.Remember(d, []*dep.Node{
						&dep.Node{Node: "node1"},
						&dep.Node{Node: "node2"},
					})
					return b
				}(),
			},
			"node1node2",
			false,
		},
		{
			"func_secret",
			`{{ with secret "secret/foo" }}{{ .Data.zip }}{{ end }}`,
			&ExecuteInput{
				Brain: func() *Brain {
					b := NewBrain()
					d, err := dep.NewVaultReadQuery("secret/foo")
					if err != nil {
						t.Fatal(err)
					}
					b.Remember(d, &dep.Secret{
						LeaseID:       "abcd1234",
						LeaseDuration: 120,
						Renewable:     true,
						Data:          map[string]interface{}{"zip": "zap"},
					})
					return b
				}(),
			},
			"zap",
			false,
		},
		{
			"func_secrets",
			`{{ range secrets "secret/" }}{{ . }}{{ end }}`,
			&ExecuteInput{
				Brain: func() *Brain {
					b := NewBrain()
					d, err := dep.NewVaultListQuery("secret/")
					if err != nil {
						t.Fatal(err)
					}
					b.Remember(d, []string{"bar", "foo"})
					return b
				}(),
			},
			"barfoo",
			false,
		},
		{
			"func_service",
			`{{ range service "webapp" }}{{ .Address }}{{ end }}`,
			&ExecuteInput{
				Brain: func() *Brain {
					b := NewBrain()
					d, err := dep.NewHealthServiceQuery("webapp")
					if err != nil {
						t.Fatal(err)
					}
					b.Remember(d, []*dep.HealthService{
						&dep.HealthService{
							Node:    "node1",
							Address: "1.2.3.4",
						},
						&dep.HealthService{
							Node:    "node2",
							Address: "5.6.7.8",
						},
					})
					return b
				}(),
			},
			"1.2.3.45.6.7.8",
			false,
		},
		{
			"func_services",
			`{{ range services }}{{ .Name }}{{ end }}`,
			&ExecuteInput{
				Brain: func() *Brain {
					b := NewBrain()
					d, err := dep.NewCatalogServicesQuery("")
					if err != nil {
						t.Fatal(err)
					}
					b.Remember(d, []*dep.CatalogSnippet{
						&dep.CatalogSnippet{
							Name: "service1",
						},
						&dep.CatalogSnippet{
							Name: "service2",
						},
					})
					return b
				}(),
			},
			"service1service2",
			false,
		},
		{
			"func_tree",
			`{{ range tree "key" }}{{ .Key }}={{ .Value }}{{ end }}`,
			&ExecuteInput{
				Brain: func() *Brain {
					b := NewBrain()
					d, err := dep.NewKVListQuery("key")
					if err != nil {
						t.Fatal(err)
					}
					b.Remember(d, []*dep.KeyPair{
						&dep.KeyPair{Key: "", Value: ""},
						&dep.KeyPair{Key: "admin/port", Value: "1134"},
						&dep.KeyPair{Key: "maxconns", Value: "5"},
						&dep.KeyPair{Key: "minconns", Value: "2"},
					})
					return b
				}(),
			},
			"admin/port=1134maxconns=5minconns=2",
			false,
		},

		// scratch
		{
			"scratch.Key",
			`{{ scratch.Set "a" "2" }}{{ scratch.Key "a" }}`,
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"true",
			false,
		},
		{
			"scratch.Get",
			`{{ scratch.Set "a" "2" }}{{ scratch.Get "a" }}`,
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"2",
			false,
		},
		{
			"scratch.SetX",
			`{{ scratch.SetX "a" "2" }}{{ scratch.SetX "a" "1" }}{{ scratch.Get "a" }}`,
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"2",
			false,
		},
		{
			"scratch.MapSet",
			`{{ scratch.MapSet "a" "foo" "bar" }}{{ scratch.MapValues "a" }}`,
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"[bar]",
			false,
		},
		{
			"scratch.MapSetX",
			`{{ scratch.MapSetX "a" "foo" "bar" }}{{ scratch.MapSetX "a" "foo" "baz" }}{{ scratch.MapValues "a" }}`,
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"[bar]",
			false,
		},

		// helpers
		{
			"helper_by_key",
			`{{ range $key, $pairs := tree "list" | byKey }}{{ $key }}:{{ range $pairs }}{{ .Key }}={{ .Value }}{{ end }}{{ end }}`,
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
			`{{ range $tag, $services := service "webapp" | byTag }}{{ $tag }}:{{ range $services }}{{ .Address }}{{ end }}{{ end }}`,
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
			`{{ range service "webapp" }}{{ if .Tags | contains "prod" }}{{ .Address }}{{ end }}{{ end }}`,
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
			"helper_env",
			`{{ env "CT_TEST" }}`,
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
			`{{ env "CT_TEST" }}`,
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
			`{{ define "custom" }}{{ key "foo" }}{{ end }}{{ executeTemplate "custom" }}`,
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
			`{{ define "custom" }}{{ key . }}{{ end }}{{ executeTemplate "custom" "foo" }}`,
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
			`{{ range $k, $v := tree "list" | explode }}{{ $k }}{{ $v }}{{ end }}`,
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
			"helper_in",
			`{{ range service "webapp" }}{{ if "prod" | in .Tags }}{{ .Address }}{{ end }}{{ end }}`,
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
			"helper_loop",
			`{{ range loop 3 }}1{{ end }}`,
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"111",
			false,
		},
		{
			"helper_loop__i",
			`{{ range $i := loop 3 }}{{ $i }}{{ end }}`,
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"012",
			false,
		},
		{
			"helper_join",
			`{{ "a,b,c" | split "," | join ";" }}`,
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"a;b;c",
			false,
		},
		{
			"helper_parseBool",
			`{{ "true" | parseBool }}`,
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"true",
			false,
		},
		{
			"helper_parseFloat",
			`{{ "1.2" | parseFloat }}`,
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"1.2",
			false,
		},
		{
			"helper_parseInt",
			`{{ "-1" | parseInt }}`,
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"-1",
			false,
		},
		{
			"helper_parseJSON",
			`{{ "{\"foo\": \"bar\"}" | parseJSON }}`,
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"map[foo:bar]",
			false,
		},
		{
			"helper_parseUint",
			`{{ "1" | parseUint }}`,
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"1",
			false,
		},
		{
			"helper_plugin",
			`{{ "1" | plugin "echo" }}`,
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"1",
			false,
		},
		{
			"helper_regexMatch",
			`{{ "foo" | regexMatch "[a-z]+" }}`,
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"true",
			false,
		},
		{
			"helper_regexReplaceAll",
			`{{ "foo" | regexReplaceAll "\\w" "x" }}`,
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"xxx",
			false,
		},
		{
			"helper_replaceAll",
			`{{ "hello my hello" | regexReplaceAll "hello" "bye" }}`,
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"bye my bye",
			false,
		},
		{
			"helper_split",
			`{{ "a,b,c" | split "," }}`,
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"[a b c]",
			false,
		},
		{
			"helper_timestamp",
			`{{ timestamp }}`,
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"1970-01-01T00:00:00Z",
			false,
		},
		{
			"helper_helper_timestamp__formatted",
			`{{ timestamp "2006-01-02" }}`,
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"1970-01-01",
			false,
		},
		{
			"helper_toJSON",
			`{{ "a,b,c" | split "," | toJSON }}`,
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"[\"a\",\"b\",\"c\"]",
			false,
		},
		{
			"helper_toLower",
			`{{ "HI" | toLower }}`,
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"hi",
			false,
		},
		{
			"helper_toTitle",
			`{{ "this is a sentence" | toTitle }}`,
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"This Is A Sentence",
			false,
		},
		{
			"helper_toTOML",
			`{{ "{\"foo\":\"bar\"}" | parseJSON | toTOML }}`,
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"foo = \"bar\"",
			false,
		},
		{
			"helper_toUpper",
			`{{ "hi" | toUpper }}`,
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"HI",
			false,
		},
		{
			"helper_toYAML",
			`{{ "{\"foo\":\"bar\"}" | parseJSON | toYAML }}`,
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"foo: bar",
			false,
		},
		{
			"helper_trimSpace",
			`{{ "\t hi\n " | trimSpace }}`,
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"hi",
			false,
		},
		{
			"math_add",
			`{{ 2 | add 2 }}`,
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"4",
			false,
		},
		{
			"math_subtract",
			`{{ 2 | subtract 2 }}`,
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"0",
			false,
		},
		{
			"math_multiply",
			`{{ 2 | multiply 2 }}`,
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"4",
			false,
		},
		{
			"math_divide",
			`{{ 2 | divide 2 }}`,
			&ExecuteInput{
				Brain: NewBrain(),
			},
			"1",
			false,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			tpl, err := NewTemplate(&NewTemplateInput{
				Contents: tc.c,
			})
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
