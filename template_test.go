package main

import (
	"bytes"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	dep "github.com/hashicorp/consul-template/dependency"
	"github.com/hashicorp/consul-template/test"
)

func TestNewTemplate_missingPath(t *testing.T) {
	_, err := NewTemplate("/path/to/non-existent/file")
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "no such file or directory"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q to be %q", err.Error(), expected)
	}
}

func TestNewTemplate_setsPathAndContents(t *testing.T) {
	contents := []byte("some content")

	in := test.CreateTempfile(contents, t)
	defer test.DeleteTempfile(in, t)

	tmpl, err := NewTemplate(in.Name())
	if err != nil {
		t.Fatal(err)
	}

	if tmpl.Path != in.Name() {
		t.Errorf("expected %q to be %q", tmpl.Path, in.Name())
	}

	if tmpl.contents != string(contents) {
		t.Errorf("expected %q to be %q", tmpl.contents, string(contents))
	}
}

func TestNewTemplate_setsPathAndMD5(t *testing.T) {
	contents := []byte("some content")

	in := test.CreateTempfile(contents, t)
	defer test.DeleteTempfile(in, t)

	tmpl, err := NewTemplate(in.Name())
	if err != nil {
		t.Fatal(err)
	}

	if tmpl.Path != in.Name() {
		t.Errorf("expected %q to be %q", tmpl.Path, in.Name())
	}

	expect := "9893532233caff98cd083a116b013c0b"
	if tmpl.hexMD5 != expect {
		t.Errorf("expected %q to be %q", tmpl.hexMD5, expect)
	}
}

func TestExecute_noDependencies(t *testing.T) {
	contents := []byte("This is a template with just text")

	in := test.CreateTempfile(contents, t)
	defer test.DeleteTempfile(in, t)

	tmpl, err := NewTemplate(in.Name())
	if err != nil {
		t.Fatal(err)
	}

	brain := NewBrain()
	used, missing, result, err := tmpl.Execute(brain)
	if err != nil {
		t.Fatal(err)
	}

	if num := len(used); num != 0 {
		t.Errorf("expected 0 missing, got: %d", num)
	}

	if num := len(missing); num != 0 {
		t.Errorf("expected 0 missing, got: %d", num)
	}

	if !bytes.Equal(result, contents) {
		t.Errorf("expected %q to be %q", result, contents)
	}
}

func TestExecute_missingDependencies(t *testing.T) {
	contents := []byte(`{{key "foo"}}`)

	in := test.CreateTempfile(contents, t)
	defer test.DeleteTempfile(in, t)

	tmpl, err := NewTemplate(in.Name())
	if err != nil {
		t.Fatal(err)
	}

	brain := NewBrain()
	used, missing, result, err := tmpl.Execute(brain)
	if err != nil {
		t.Fatal(err)
	}

	if num := len(used); num != 1 {
		t.Fatalf("expected 1 used, got: %d", num)
	}

	if num := len(missing); num != 1 {
		t.Fatalf("expected 1 missing, got: %d", num)
	}

	expected, err := dep.ParseStoreKey("foo")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(missing[0], expected) {
		t.Errorf("expected %v to be %v", missing[0], expected)
	}

	if num := len(used); num != 1 {
		t.Fatalf("expected 1 used, got %d", num)
	}

	if !reflect.DeepEqual(used[0], expected) {
		t.Errorf("expected %v to be %v", used[0], expected)
	}

	expectedResult := []byte("")
	if !bytes.Equal(result, expectedResult) {
		t.Errorf("expected %q to be %q", result, expectedResult)
	}
}

func TestExecte_badFuncs(t *testing.T) {
	in := test.CreateTempfile([]byte(`{{ tickle_me_pink }}`), t)
	defer test.DeleteTempfile(in, t)

	tmpl, err := NewTemplate(in.Name())
	if err != nil {
		t.Fatal(err)
	}

	brain := NewBrain()
	used, missing, result, err := tmpl.Execute(brain)
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := `function "tickle_me_pink" not defined`
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q to contain %q", err.Error(), expected)
	}

	if used != nil {
		t.Errorf("expected used to be nil")
	}

	if missing != nil {
		t.Errorf("expected missing to be nil")
	}

	if result != nil {
		t.Errorf("expected result to be nil")
	}
}

func TestExecute_funcs(t *testing.T) {
	in := test.CreateTempfile([]byte(`
    {{ range service "release.webapp" }}{{.Address}}{{ end }}
    {{ key "service/redis/maxconns" }}
    {{ range ls "service/redis/config" }}{{.Key}}{{ end }}
  `), t)
	defer test.DeleteTempfile(in, t)

	tmpl, err := NewTemplate(in.Name())
	if err != nil {
		t.Fatal(err)
	}

	brain := NewBrain()
	used, missing, _, err := tmpl.Execute(brain)
	if err != nil {
		t.Fatal(err)
	}

	if num := len(missing); num != 3 {
		t.Fatalf("expected 3 missing, got: %d", num)
	}

	if num := len(used); num != 3 {
		t.Fatalf("expected 3 used, got: %d", num)
	}
}

func TestExecute_duplicateFuncs(t *testing.T) {
	in := test.CreateTempfile([]byte(`
    {{ key "service/redis/maxconns" }}
    {{ key "service/redis/maxconns" }}
    {{ key "service/redis/maxconns" }}
  `), t)
	defer test.DeleteTempfile(in, t)

	tmpl, err := NewTemplate(in.Name())
	if err != nil {
		t.Fatal(err)
	}

	brain := NewBrain()
	used, missing, _, err := tmpl.Execute(brain)
	if err != nil {
		t.Fatal(err)
	}

	if num := len(missing); num != 1 {
		t.Fatalf("expected 1 missing, got: %d", num)
	}

	if num := len(used); num != 1 {
		t.Fatalf("expected 1 used, got: %d", num)
	}
}

func TestExecute_renders(t *testing.T) {
	// Stub out the time.
	now = func() time.Time { return time.Unix(0, 0).UTC() }

	in := test.CreateTempfile([]byte(`
		API Functions
		-------------
		datacenters:{{ range datacenters }}
			{{.}}{{ end }}
		file: {{ file "/path/to/file" }}
		key: {{ key "config/redis/maxconns" }}
		key_or_default (exists): {{ key_or_default "config/redis/minconns" "100" }}
		key_or_default (missing): {{ key_or_default "config/redis/maxconns" "200" }}
		ls:{{ range ls "config/redis" }}
			{{.Key}}={{.Value}}{{ end }}
		node:{{ with node }}
			{{.Node.Node}}{{ range .Services}}
				{{.Service}}{{ end }}{{ end }}
		nodes:{{ range nodes }}
			{{.Node}}{{ end }}
		secret: {{ with secret "secret/foo/bar" }}{{.Data.zip}}{{ end }}
		secrets:{{ range secrets "secret/" }}
			{{.}}{{ end }}
		service:{{ range service "webapp" }}
			{{.Address}}{{ end }}
		service (any):{{ range service "webapp" "any" }}
			{{.Address}}{{ end }}
		service (tag.Contains):{{ range service "webapp" }}{{ if .Tags.Contains "production" }}
			{{.Node}}{{ end }}{{ end }}
		services:{{ range services }}
			{{.Name}}{{ end }}
		tree:{{ range tree "config/redis" }}
			{{.Key}}={{.Value}}{{ end }}
		vault: {{ with vault "secret/foo/bar" }}{{.Data.zip}}{{ end }}

		Helper Functions
		----------------
		byKey:{{ range $key, $pairs := tree "config/redis" | byKey }}
			{{$key}}:{{ range $pairs }}
				{{.Key}}={{.Value}}{{ end }}{{ end }}
		byTag (health service):{{ range $tag, $services := service "webapp" | byTag }}
			{{$tag}}:{{ range $services }}
				{{.Address}}{{ end }}{{ end }}
		byTag (catalog services):{{ range $tag, $services := services | byTag }}
			{{$tag}}:{{ range $services }}
				{{.Name}}{{ end }}{{ end }}
		contains:{{ range service "webapp" }}{{ if .Tags | contains "production" }}
			{{.Node}}{{ end }}{{ end }}
		env: {{ env "foo" }}
		explode:{{ range $k, $v := tree "config/redis" | explode }}
			{{$k}}{{$v}}{{ end }}
		in:{{ range service "webapp" }}{{ if in .Tags "production" }}
			{{.Node}}{{ end }}{{ end }}
		loop:{{ range loop 3 }}
			test{{ end }}
		loop(i):{{ range $i := loop 5 8 }}
			test{{$i}}{{ end }}
		join: {{ "a,b,c" | split "," | join ";" }}
		trimSpace: {{ "\t Hello, World\n " | trimSpace }}
		parseBool: {{"true" | parseBool}}
		parseFloat: {{"1.2" | parseFloat}}
		parseInt: {{"-1" | parseInt}}
		parseJSON (string):{{ range $key, $value := "{\"foo\": \"bar\"}" | parseJSON }}
			{{$key}}={{$value}}{{ end }}
		parseJSON (file):{{ range $key, $value := file "/path/to/json/file" | parseJSON }}
			{{$key}}={{$value}}{{ end }}
		parseJSON (env):{{ range $key, $value := env "json" | parseJSON }}
			{{$key}}={{$value}}{{ end }}
		parseUint: {{"1" | parseUint}}
		plugin: {{ file "/path/to/json/file" | plugin "echo" }}
		timestamp: {{ timestamp }}
		timestamp (formatted): {{ timestamp "2006-01-02" }}
		regexMatch: {{ file "/path/to/file"  | regexMatch ".*[cont][a-z]+" }}
		regexMatch: {{ file "/path/to/file"  | regexMatch "v[0-9]*" }}
		regexReplaceAll: {{ file "/path/to/file" | regexReplaceAll "\\w" "x" }}
		replaceAll: {{ file "/path/to/file" | replaceAll "some" "this" }}
		split:{{ range "a,b,c" | split "," }}
			{{.}}{{end}}
		toLower: {{ file "/path/to/file" | toLower }}
		toJSON: {{ tree "config/redis" | explode | toJSON }}
		toJSONPretty:
{{ tree "config/redis" | explode | toJSONPretty }}
		toTitle: {{ file "/path/to/file" | toTitle }}
		toUpper: {{ file "/path/to/file" | toUpper }}
		toYAML:
{{ tree "config/redis" | explode | toYAML }}

		Math Functions
		--------------
		add:{{ 2 | add 2 }}
		subtract:{{ 2 | subtract 2 }}
		multiply:{{ 2 | multiply 2 }}
		divide:{{ 2 | divide 2 }}
`), t)
	defer test.DeleteTempfile(in, t)

	tmpl, err := NewTemplate(in.Name())
	if err != nil {
		t.Fatal(err)
	}

	brain := NewBrain()

	var d dep.Dependency

	d, err = dep.ParseDatacenters()
	if err != nil {
		t.Fatal(err)
	}
	brain.Remember(d, []string{"dc1", "dc2"})

	d, err = dep.ParseFile("/path/to/file")
	if err != nil {
		t.Fatal(err)
	}
	brain.Remember(d, "some content")

	d, err = dep.ParseStoreKey("config/redis/maxconns")
	if err != nil {
		t.Fatal(err)
	}
	brain.Remember(d, "5")

	d, err = dep.ParseStoreKey("config/redis/minconns")
	if err != nil {
		t.Fatal(err)
	}
	d.(*dep.StoreKey).SetDefault("100")
	brain.Remember(d, "150")

	d, err = dep.ParseStoreKey("config/redis/maxconns")
	if err != nil {
		t.Fatal(err)
	}
	d.(*dep.StoreKey).SetDefault("200")

	d, err = dep.ParseStoreKeyPrefix("config/redis")
	if err != nil {
		t.Fatal(err)
	}
	brain.Remember(d, []*dep.KeyPair{
		&dep.KeyPair{Key: "", Value: ""},
		&dep.KeyPair{Key: "admin/port", Value: "1134"},
		&dep.KeyPair{Key: "maxconns", Value: "5"},
		&dep.KeyPair{Key: "minconns", Value: "2"},
	})

	d, err = dep.ParseCatalogNode()
	if err != nil {
		t.Fatal(err)
	}
	brain.Remember(d, &dep.NodeDetail{
		Node: &dep.Node{Node: "node1"},
		Services: dep.NodeServiceList([]*dep.NodeService{
			&dep.NodeService{
				Service: "service1",
			},
		}),
	})

	d, err = dep.ParseCatalogNodes("")
	if err != nil {
		t.Fatal(err)
	}
	brain.Remember(d, []*dep.Node{
		&dep.Node{Node: "node1"},
		&dep.Node{Node: "node2"},
	})

	d, err = dep.ParseHealthServices("webapp")
	if err != nil {
		t.Fatal(err)
	}
	brain.Remember(d, []*dep.HealthService{
		&dep.HealthService{
			Node:    "node1",
			Address: "1.2.3.4",
			Tags:    []string{"release"},
		},
		&dep.HealthService{
			Node:    "node2",
			Address: "5.6.7.8",
			Tags:    []string{"release", "production"},
		},
		&dep.HealthService{
			Node:    "node3",
			Address: "9.10.11.12",
			Tags:    []string{"production"},
		},
	})

	d, err = dep.ParseHealthServices("webapp", "any")
	if err != nil {
		t.Fatal(err)
	}
	brain.Remember(d, []*dep.HealthService{
		&dep.HealthService{Node: "node1", Address: "1.2.3.4"},
		&dep.HealthService{Node: "node2", Address: "5.6.7.8"},
	})

	d, err = dep.ParseCatalogServices("")
	if err != nil {
		t.Fatal(err)
	}
	brain.Remember(d, []*dep.CatalogService{
		&dep.CatalogService{
			Name: "service1",
			Tags: []string{"production"},
		},
		&dep.CatalogService{
			Name: "service2",
			Tags: []string{"release", "production"},
		},
	})

	d, err = dep.ParseVaultSecrets("secret/")
	if err != nil {
		t.Fatal(err)
	}
	brain.Remember(d, []string{"bar", "foo"})

	d, err = dep.ParseVaultSecret("secret/foo/bar")
	if err != nil {
		t.Fatal(err)
	}
	brain.Remember(d, &dep.Secret{
		LeaseID:       "abcd1234",
		LeaseDuration: 120,
		Renewable:     true,
		Data:          map[string]interface{}{"zip": "zap"},
	})

	if err := os.Setenv("foo", "bar"); err != nil {
		t.Fatal(err)
	}

	d, err = dep.ParseFile("/path/to/json/file")
	if err != nil {
		t.Fatal(err)
	}
	brain.Remember(d, `{"foo": "bar"}`)

	if err := os.Setenv("json", `{"foo": "bar"}`); err != nil {
		t.Fatal(err)
	}

	_, _, result, err := tmpl.Execute(brain)
	if err != nil {
		t.Fatal(err)
	}

	expected := []byte(`
		API Functions
		-------------
		datacenters:
			dc1
			dc2
		file: some content
		key: 5
		key_or_default (exists): 150
		key_or_default (missing): 200
		ls:
			maxconns=5
			minconns=2
		node:
			node1
				service1
		nodes:
			node1
			node2
		secret: zap
		secrets:
			bar
			foo
		service:
			1.2.3.4
			5.6.7.8
			9.10.11.12
		service (any):
			1.2.3.4
			5.6.7.8
		service (tag.Contains):
			node2
			node3
		services:
			service1
			service2
		tree:
			admin/port=1134
			maxconns=5
			minconns=2
		vault: zap

		Helper Functions
		----------------
		byKey:
			admin:
				port=1134
		byTag (health service):
			production:
				5.6.7.8
				9.10.11.12
			release:
				1.2.3.4
				5.6.7.8
		byTag (catalog services):
			production:
				service1
				service2
			release:
				service2
		contains:
			node2
			node3
		env: bar
		explode:
			adminmap[port:1134]
			maxconns5
			minconns2
		in:
			node2
			node3
		loop:
			test
			test
			test
		loop(i):
			test5
			test6
			test7
		join: a;b;c
		trimSpace: Hello, World
		parseBool: true
		parseFloat: 1.2
		parseInt: -1
		parseJSON (string):
			foo=bar
		parseJSON (file):
			foo=bar
		parseJSON (env):
			foo=bar
		parseUint: 1
		plugin: {"foo": "bar"}
		timestamp: 1970-01-01T00:00:00Z
		timestamp (formatted): 1970-01-01
		regexMatch: true
		regexMatch: false
		regexReplaceAll: xxxx xxxxxxx
		replaceAll: this content
		split:
			a
			b
			c
		toLower: some content
		toJSON: {"admin":{"port":"1134"},"maxconns":"5","minconns":"2"}
		toJSONPretty:
{
  "admin": {
    "port": "1134"
  },
  "maxconns": "5",
  "minconns": "2"
}
		toTitle: Some Content
		toUpper: SOME CONTENT
		toYAML:
admin:
  port: "1134"
maxconns: "5"
minconns: "2"

		Math Functions
		--------------
		add:4
		subtract:0
		multiply:4
		divide:1
`)

	if !bytes.Equal(result, expected) {
		t.Errorf("expected \n\n%q\n\n to be \n\n%q\n\n", result, expected)
	}
}

func TestExecute_multipass(t *testing.T) {
	in := test.CreateTempfile([]byte(`
		{{ range ls "services" }}{{.Key}}:{{ range service .Key }}
			{{.Node}} {{.Address}}:{{.Port}}{{ end }}
		{{ end }}
	`), t)
	defer test.DeleteTempfile(in, t)

	tmpl, err := NewTemplate(in.Name())
	if err != nil {
		t.Fatal(err)
	}

	brain := NewBrain()

	used, missing, result, err := tmpl.Execute(brain)
	if err != nil {
		t.Fatal(err)
	}

	if num := len(missing); num != 1 {
		t.Errorf("expected 1 missing, got: %d", num)
	}

	if num := len(used); num != 1 {
		t.Errorf("expected 1 used, got: %d", num)
	}

	expected := bytes.TrimSpace([]byte(""))
	result = bytes.TrimSpace(result)
	if !bytes.Equal(result, expected) {
		t.Errorf("expected %q to be %q", result, expected)
	}

	// Receive data for the key prefix dependency
	d1, err := dep.ParseStoreKeyPrefix("services")
	brain.Remember(d1, []*dep.KeyPair{
		&dep.KeyPair{Key: "webapp", Value: "1"},
		&dep.KeyPair{Key: "database", Value: "1"},
	})

	used, missing, result, err = tmpl.Execute(brain)
	if err != nil {
		t.Fatal(err)
	}

	if num := len(missing); num != 2 {
		t.Errorf("expected 2 missing, got: %d", num)
	}

	if num := len(used); num != 3 {
		t.Errorf("expected 3 used, got: %d", num)
	}

	expected = bytes.TrimSpace([]byte(`
		webapp:
		database:
	`))
	result = bytes.TrimSpace(result)
	if !bytes.Equal(result, expected) {
		t.Errorf("expected \n%q\n to be \n%q\n", result, expected)
	}

	// Receive data for the services
	d2, err := dep.ParseHealthServices("webapp")
	brain.Remember(d2, []*dep.HealthService{
		&dep.HealthService{Node: "web01", Address: "1.2.3.4", Port: 1234},
	})

	d3, err := dep.ParseHealthServices("database")
	brain.Remember(d3, []*dep.HealthService{
		&dep.HealthService{Node: "db01", Address: "5.6.7.8", Port: 5678},
	})

	used, missing, result, err = tmpl.Execute(brain)
	if err != nil {
		t.Fatal(err)
	}

	if num := len(missing); num != 0 {
		t.Errorf("expected 0 missing, got: %d", num)
	}

	if num := len(used); num != 3 {
		t.Errorf("expected 3 used, got: %d", num)
	}

	expected = bytes.TrimSpace([]byte(`
		webapp:
			web01 1.2.3.4:1234
		database:
			db01 5.6.7.8:5678
	`))
	result = bytes.TrimSpace(result)
	if !bytes.Equal(result, expected) {
		t.Errorf("expected \n%q\n to be \n%q\n", result, expected)
	}
}
