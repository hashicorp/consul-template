package main

import (
	"bytes"
	"os"
	"reflect"
	"strings"
	"testing"

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

func TestExecute_noDependencies(t *testing.T) {
	contents := []byte("This is a template with just text")

	in := test.CreateTempfile(contents, t)
	defer test.DeleteTempfile(in, t)

	tmpl, err := NewTemplate(in.Name())
	if err != nil {
		t.Fatal(err)
	}

	brain := NewBrain()
	missing, result, err := tmpl.Execute(brain)
	if err != nil {
		t.Fatal(err)
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
	missing, result, err := tmpl.Execute(brain)
	if err != nil {
		t.Fatal(err)
	}

	if num := len(missing); num != 1 {
		t.Fatalf("expected 1 missing, got: %d", num)
	}

	expected, err := dep.ParseStoreKey("foo")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(missing[0], expected) {
		t.Errorf("expected %q to be %q", missing[0], expected)
	}

	if num := len(tmpl.dependencies); num != 1 {
		t.Fatalf("expected 1 used, got %d", num)
	}

	if !reflect.DeepEqual(tmpl.dependencies[0], expected) {
		t.Errorf("expected %q to be %q", tmpl.dependencies[0], expected)
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
	missing, result, err := tmpl.Execute(brain)
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := `function "tickle_me_pink" not defined`
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q to contain %q", err.Error(), expected)
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
	missing, _, err := tmpl.Execute(brain)
	if err != nil {
		t.Fatal(err)
	}

	if num := len(missing); num != 3 {
		t.Fatalf("expected 3 missing, got: %d", num)
	}

	if num := len(tmpl.dependencies); num != 3 {
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
	missing, _, err := tmpl.Execute(brain)
	if err != nil {
		t.Fatal(err)
	}

	if num := len(missing); num != 1 {
		t.Fatalf("expected 1 missing, got: %d", num)
	}

	if num := len(tmpl.dependencies); num != 1 {
		t.Fatalf("expected 1 used, got: %d", num)
	}
}

func TestExecute_renders(t *testing.T) {
	in := test.CreateTempfile([]byte(`
		API Functions
		-------------
		file: {{ file "/path/to/file" }}
		key: {{ key "config/redis/maxconns" }}
		ls:{{ range ls "config/redis" }}
			{{.Key}}={{.Value}}{{ end }}
		nodes:{{ range nodes }}
			{{.Node}}{{ end }}
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

		Helper Functions
		----------------
		byTag:{{ range $tag, $services := service "webapp" | byTag }}
			{{$tag}}:{{ range $services }}
				{{.Address}}{{ end }}{{ end }}
		env: {{ env "foo" }}
		parseJSON:{{ range $key, $value := "{\"foo\": \"bar\"}" | parseJSON }}
			{{$key}}={{$value}}{{ end }}
		regexReplaceAll: {{ file "/path/to/file" | regexReplaceAll "\\w" "x" }}
		replaceAll: {{ file "/path/to/file" | replaceAll "some" "this" }}
		toLower: {{ file "/path/to/file" | toLower }}
		toTitle: {{ file "/path/to/file" | toTitle }}
		toUpper: {{ file "/path/to/file" | toUpper }}
	`), t)
	defer test.DeleteTempfile(in, t)

	tmpl, err := NewTemplate(in.Name())
	if err != nil {
		t.Fatal(err)
	}

	brain := NewBrain()

	var d dep.Dependency

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
		&dep.CatalogService{Name: "service1"},
		&dep.CatalogService{Name: "service2"},
	})

	if err := os.Setenv("foo", "bar"); err != nil {
		t.Fatal(err)
	}

	_, result, err := tmpl.Execute(brain)
	if err != nil {
		t.Fatal(err)
	}

	expected := []byte(`
		API Functions
		-------------
		file: some content
		key: 5
		ls:
			maxconns=5
			minconns=2
		nodes:
			node1
			node2
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

		Helper Functions
		----------------
		byTag:
			production:
				5.6.7.8
				9.10.11.12
			release:
				1.2.3.4
				5.6.7.8
		env: bar
		parseJSON:
			foo=bar
		regexReplaceAll: xxxx xxxxxxx
		replaceAll: this content
		toLower: some content
		toTitle: Some Content
		toUpper: SOME CONTENT
	`)

	if !bytes.Equal(result, expected) {
		t.Errorf("expected \n%s\n to be \n%s\n", result, expected)
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

	missing, result, err := tmpl.Execute(brain)
	if err != nil {
		t.Fatal(err)
	}

	if num := len(missing); num != 1 {
		t.Errorf("expected 1 missing, got: %d", num)
	}

	if num := len(tmpl.dependencies); num != 1 {
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

	missing, result, err = tmpl.Execute(brain)
	if err != nil {
		t.Fatal(err)
	}

	if num := len(missing); num != 2 {
		t.Errorf("expected 2 missing, got: %d", num)
	}

	if num := len(tmpl.dependencies); num != 3 {
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

	missing, result, err = tmpl.Execute(brain)
	if err != nil {
		t.Fatal(err)
	}

	if num := len(missing); num != 0 {
		t.Errorf("expected 0 missing, got: %d", num)
	}

	if num := len(tmpl.dependencies); num != 3 {
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
