package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/consul-template/dependency"
	"github.com/hashicorp/consul-template/test"
)

func TestDependencies_empty(t *testing.T) {
	inTemplate := test.CreateTempfile(nil, t)
	defer test.DeleteTempfile(inTemplate, t)

	template, err := NewTemplate(inTemplate.Name())
	if err != nil {
		t.Fatal(err)
	}
	dependencies := template.Dependencies()

	if num := len(dependencies); num != 0 {
		t.Errorf("expected 0 Dependency, got: %d", num)
	}
}

func TestDependencies_funcs(t *testing.T) {
	inTemplate := test.CreateTempfile([]byte(`
    {{ range service "release.webapp" }}{{ end }}
    {{ key "service/redis/maxconns" }}
    {{ range storeKeyPrefix "service/redis/config" }}{{ end }}
  `), t)
	defer test.DeleteTempfile(inTemplate, t)

	template, err := NewTemplate(inTemplate.Name())
	if err != nil {
		t.Fatal(err)
	}
	dependencies := template.Dependencies()

	if num := len(dependencies); num != 3 {
		t.Fatalf("expected 3 dependencies, got: %d", num)
	}
}

func TestDependencies_funcsDuplicates(t *testing.T) {
	inTemplate := test.CreateTempfile([]byte(`
    {{ range service "release.webapp" }}{{ end }}
    {{ range service "release.webapp" }}{{ end }}
    {{ range service "release.webapp" }}{{ end }}
  `), t)
	defer test.DeleteTempfile(inTemplate, t)

	template, err := NewTemplate(inTemplate.Name())
	if err != nil {
		t.Fatal(err)
	}
	dependencies := template.Dependencies()

	if num := len(dependencies); num != 1 {
		t.Fatalf("expected 1 Dependency, got: %d", num)
	}

	dependency, expected := dependencies[0], "release.webapp [passing]"
	if dependency.Key() != expected {
		t.Errorf("expected %q to equal %q", dependency.Key(), expected)
	}
}

func TestDependencies_funcsError(t *testing.T) {
	inTemplate := test.CreateTempfile([]byte(`
    {{ range service "totally&not&a&valid&service" }}{{ end }}
  `), t)
	defer test.DeleteTempfile(inTemplate, t)

	_, err := NewTemplate(inTemplate.Name())
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "error calling service:"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q to contain %q", err.Error(), expected)
	}
}

func TestExecute_noTemplateContext(t *testing.T) {
	inTemplate := test.CreateTempfile(nil, t)
	defer test.DeleteTempfile(inTemplate, t)

	template, err := NewTemplate(inTemplate.Name())
	if err != nil {
		t.Fatal(err)
	}

	_, executeErr := template.Execute(nil)
	if executeErr == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "templateContext must be given"
	if !strings.Contains(executeErr.Error(), expected) {
		t.Errorf("expected %q to contain %q", executeErr.Error(), expected)
	}
}

func TestExecute_dependenciesError(t *testing.T) {
	inTemplate := test.CreateTempfile([]byte(`
    {{ range not_a_valid "template" }}{{ end }}
  `), t)
	defer test.DeleteTempfile(inTemplate, t)

	_, err := NewTemplate(inTemplate.Name())
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := `template: out:2: function "not_a_valid" not defined`
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q to contain %q", err.Error(), expected)
	}
}

func TestHashCode_returnsValue(t *testing.T) {
	template := &Template{Path: "/foo/bar/blitz.ctmpl"}

	expected := "Template|/foo/bar/blitz.ctmpl"
	if template.HashCode() != expected {
		t.Errorf("expected %q to equal %q", template.HashCode(), expected)
	}
}

func TestExecute_missingFile(t *testing.T) {
	inTemplate := test.CreateTempfile([]byte(`
    {{ file "/path/to/file" }}
  `), t)
	defer test.DeleteTempfile(inTemplate, t)

	template, err := NewTemplate(inTemplate.Name())
	if err != nil {
		t.Fatal(err)
	}

	context := &TemplateContext{}
	_, executeErr := template.Execute(context)
	if executeErr == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := `template context missing file "/path/to/file"`
	if !strings.Contains(executeErr.Error(), expected) {
		t.Errorf("expected %q to contain %q", executeErr.Error(), expected)
	}
}

func TestExecute_missingStoreKeyPrefix(t *testing.T) {
	inTemplate := test.CreateTempfile([]byte(`
    {{ range storeKeyPrefix "service/nginx/config" }}{{ end }}
  `), t)
	defer test.DeleteTempfile(inTemplate, t)

	template, err := NewTemplate(inTemplate.Name())
	if err != nil {
		t.Fatal(err)
	}

	context := &TemplateContext{}
	_, executeErr := template.Execute(context)
	if executeErr == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := `template context missing storeKeyPrefix "service/nginx/config"`
	if !strings.Contains(executeErr.Error(), expected) {
		t.Errorf("expected %q to contain %q", executeErr.Error(), expected)
	}
}

func TestExecute_missingKey(t *testing.T) {
	inTemplate := test.CreateTempfile([]byte(`
    {{ key "service/redis/online" }}
  `), t)
	defer test.DeleteTempfile(inTemplate, t)

	template, err := NewTemplate(inTemplate.Name())
	if err != nil {
		t.Fatal(err)
	}

	context := &TemplateContext{}
	_, executeErr := template.Execute(context)
	if executeErr == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := `template context missing key "service/redis/online"`
	if !strings.Contains(executeErr.Error(), expected) {
		t.Errorf("expected %q to contain %q", executeErr.Error(), expected)
	}
}

func TestExecute_missingNodes(t *testing.T) {
	inTemplate := test.CreateTempfile([]byte(`
    {{ range nodes "@nyc1" }}{{ end }}
  `), t)
	defer test.DeleteTempfile(inTemplate, t)

	template, err := NewTemplate(inTemplate.Name())
	if err != nil {
		t.Fatal(err)
	}

	context := &TemplateContext{}
	_, executeErr := template.Execute(context)
	if executeErr == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := `template context missing nodes "@nyc1"`
	if !strings.Contains(executeErr.Error(), expected) {
		t.Errorf("expected %q to contain %q", executeErr.Error(), expected)
	}
}

func TestExecute_missingService(t *testing.T) {
	inTemplate := test.CreateTempfile([]byte(`
    {{ range service "production.webapp" }}{{ end }}
  `), t)
	defer test.DeleteTempfile(inTemplate, t)

	template, err := NewTemplate(inTemplate.Name())
	if err != nil {
		t.Fatal(err)
	}

	context := &TemplateContext{}
	_, executeErr := template.Execute(context)
	if executeErr == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := `template context missing service "production.webapp [passing]"`
	if !strings.Contains(executeErr.Error(), expected) {
		t.Errorf("expected %q to contain %q", executeErr.Error(), expected)
	}
}

func TestExecute_missingServices(t *testing.T) {
	inTemplate := test.CreateTempfile([]byte(`
    {{ range services }}{{ end }}
  `), t)
	defer test.DeleteTempfile(inTemplate, t)

	template, err := NewTemplate(inTemplate.Name())
	if err != nil {
		t.Fatal(err)
	}

	context := &TemplateContext{}
	_, executeErr := template.Execute(context)
	if executeErr == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "template context missing catalog services"
	if !strings.Contains(executeErr.Error(), expected) {
		t.Errorf("expected %q to contain %q", executeErr.Error(), expected)
	}
}

func TestExecute_rendersFile(t *testing.T) {
	inTemplate := test.CreateTempfile([]byte(`
    {{ file "/path/to/file" }}
  `), t)
	defer test.DeleteTempfile(inTemplate, t)

	template, err := NewTemplate(inTemplate.Name())
	if err != nil {
		t.Fatal(err)
	}

	context := &TemplateContext{
		files: map[string]string{
			"/path/to/file": "some content",
		},
	}

	contents, err := template.Execute(context)
	if err != nil {
		t.Fatal(err)
	}

	expected := []byte(`
    some content
  `)
	if !bytes.Equal(contents, expected) {
		t.Errorf("expected \n%q\n to equal \n%q\n", contents, expected)
	}
}

// DEPRECATED. Use `ls` or `tree` instead.
func TestExecute_rendersStoreKeyPrefix(t *testing.T) {
	inTemplate := test.CreateTempfile([]byte(`
    {{ range storeKeyPrefix "service/redis" }}{{.Key}}={{.Value}}{{end}}
  `), t)
	defer test.DeleteTempfile(inTemplate, t)

	template, err := NewTemplate(inTemplate.Name())
	if err != nil {
		t.Fatal(err)
	}

	context := &TemplateContext{
		storeKeyPrefixes: map[string][]*dependency.KeyPair{
			"service/redis": []*dependency.KeyPair{
				&dependency.KeyPair{Path: "/path1", Key: "key1", Value: "value1"},
				&dependency.KeyPair{Path: "/path2", Key: "key2", Value: "value2"},
			},
		},
	}

	contents, err := template.Execute(context)
	if err != nil {
		t.Fatal(err)
	}

	expected := []byte(`
    key1=value1key2=value2
  `)
	if !bytes.Equal(contents, expected) {
		t.Errorf("expected \n%q\n to equal \n%q\n", contents, expected)
	}
}

func TestExecute_rendersKey(t *testing.T) {
	inTemplate := test.CreateTempfile([]byte(`
    minconns: {{ key "service/redis/minconns" }}
    maxconns: {{ key "service/redis/maxconns" }}
  `), t)
	defer test.DeleteTempfile(inTemplate, t)

	template, err := NewTemplate(inTemplate.Name())
	if err != nil {
		t.Fatal(err)
	}

	context := &TemplateContext{
		storeKeys: map[string]string{
			"service/redis/minconns": "2",
			"service/redis/maxconns": "11",
		},
	}

	contents, err := template.Execute(context)
	if err != nil {
		t.Fatal(err)
	}

	expected := []byte(`
    minconns: 2
    maxconns: 11
  `)
	if !bytes.Equal(contents, expected) {
		t.Errorf("expected \n%q\n to equal \n%q\n", contents, expected)
	}
}

func TestExecute_rendersLs(t *testing.T) {
	inTemplate := test.CreateTempfile([]byte(`
    {{ range ls "service/redis/config" }}
    {{.Key}} = {{.Value}}{{ end }}
  `), t)
	defer test.DeleteTempfile(inTemplate, t)

	template, err := NewTemplate(inTemplate.Name())
	if err != nil {
		t.Fatal(err)
	}

	minconnsConfig := &dependency.KeyPair{
		Key:   "minconns",
		Value: "2",
	}

	maxconnsConfig := &dependency.KeyPair{
		Key:   "maxconns",
		Value: "11",
	}

	emptyFolderConfig := &dependency.KeyPair{
		Key:   "",
		Value: "",
	}

	childConfig := &dependency.KeyPair{
		Key:   "user/sethvargo",
		Value: "true",
	}

	context := &TemplateContext{
		storeKeyPrefixes: map[string][]*dependency.KeyPair{
			"service/redis/config": []*dependency.KeyPair{
				emptyFolderConfig,
				minconnsConfig,
				maxconnsConfig,
				childConfig,
			},
		},
	}

	contents, err := template.Execute(context)
	if err != nil {
		t.Fatal(err)
	}

	expected := bytes.TrimSpace([]byte(`
    minconns = 2
    maxconns = 11
  `))
	if !bytes.Equal(bytes.TrimSpace(contents), expected) {
		t.Errorf("expected \n%q\n to equal \n%q\n", bytes.TrimSpace(contents), expected)
	}
}

func TestExecute_rendersNodes(t *testing.T) {
	inTemplate := test.CreateTempfile([]byte(`
    {{ range nodes }}
    node {{.Node}} {{.Address}}{{ end }}
  `), t)
	defer test.DeleteTempfile(inTemplate, t)

	template, err := NewTemplate(inTemplate.Name())
	if err != nil {
		t.Fatal(err)
	}

	node1 := &dependency.Node{
		Node:    "nyc-worker-1",
		Address: "123.123.123.123",
	}

	node2 := &dependency.Node{
		Node:    "nyc-worker-2",
		Address: "456.456.456.456",
	}

	context := &TemplateContext{
		catalogNodes: map[string][]*dependency.Node{
			"": []*dependency.Node{node1, node2},
		},
	}

	contents, err := template.Execute(context)
	if err != nil {
		t.Fatal(err)
	}

	expected := bytes.TrimSpace([]byte(`
    node nyc-worker-1 123.123.123.123
    node nyc-worker-2 456.456.456.456
  `))

	if !bytes.Equal(bytes.TrimSpace(contents), expected) {
		t.Errorf("expected \n%q\n to equal \n%q\n", bytes.TrimSpace(contents), expected)
	}
}

func TestExecute_rendersService(t *testing.T) {
	inTemplate := test.CreateTempfile([]byte(`
    {{ range service "release.webapp" }}
    server {{.Name}} {{.Address}}:{{.Port}}{{ end }}
  `), t)
	defer test.DeleteTempfile(inTemplate, t)

	template, err := NewTemplate(inTemplate.Name())
	if err != nil {
		t.Fatal(err)
	}

	serviceWeb1 := &dependency.HealthService{
		Node:    "nyc-worker-1",
		Address: "123.123.123.123",
		ID:      "web1",
		Name:    "web1",
		Port:    1234,
	}

	serviceWeb2 := &dependency.HealthService{
		Node:    "nyc-worker-2",
		Address: "456.456.456.456",
		ID:      "web2",
		Name:    "web2",
		Port:    5678,
	}

	context := &TemplateContext{
		healthServices: map[string][]*dependency.HealthService{
			"release.webapp [passing]": []*dependency.HealthService{serviceWeb1, serviceWeb2},
		},
	}

	contents, err := template.Execute(context)
	if err != nil {
		t.Fatal(err)
	}

	expected := bytes.TrimSpace([]byte(`
    server web1 123.123.123.123:1234
    server web2 456.456.456.456:5678
  `))

	if !bytes.Equal(bytes.TrimSpace(contents), expected) {
		t.Errorf("expected \n%q\n to equal \n%q\n", bytes.TrimSpace(contents), expected)
	}
}

func TestExecute_rendersServiceWithHealthCheckArgument(t *testing.T) {
	inTemplate := test.CreateTempfile([]byte(`
		{{ range service "release.webapp" "any" }}
		server {{.Name}} {{.Address}}:{{.Port}}{{ end }}
	`), t)
	defer test.DeleteTempfile(inTemplate, t)

	template, err := NewTemplate(inTemplate.Name())
	if err != nil {
		t.Fatal(err)
	}

	serviceWeb1 := &dependency.HealthService{
		Node:    "nyc-worker-1",
		Address: "123.123.123.123",
		ID:      "web1",
		Name:    "web1",
		Port:    1234,
	}

	serviceWeb2 := &dependency.HealthService{
		Node:    "nyc-worker-2",
		Address: "456.456.456.456",
		ID:      "web2",
		Name:    "web2",
		Port:    5678,
	}

	context := &TemplateContext{
		healthServices: map[string][]*dependency.HealthService{
			"release.webapp [any]": []*dependency.HealthService{serviceWeb1, serviceWeb2},
		},
	}

	contents, err := template.Execute(context)
	if err != nil {
		t.Fatal(err)
	}

	expected := bytes.TrimSpace([]byte(`
		server web1 123.123.123.123:1234
		server web2 456.456.456.456:5678
	`))

	if !bytes.Equal(bytes.TrimSpace(contents), expected) {
		t.Errorf("expected \n%q\n to equal \n%q\n", bytes.TrimSpace(contents), expected)
	}
}

func TestExecute_rendersServices(t *testing.T) {
	inTemplate := test.CreateTempfile([]byte(`
		{{range services}}{{.Name}}: {{range .Tags}}{{.}},{{end}}|{{end}}
  `), t)
	defer test.DeleteTempfile(inTemplate, t)

	template, err := NewTemplate(inTemplate.Name())
	if err != nil {
		t.Fatal(err)
	}

	catalogWeb1 := &dependency.CatalogService{
		Name: "nyc-worker-1",
		Tags: []string{"tag1", "tag2"},
	}

	catalogWeb2 := &dependency.CatalogService{
		Name: "nyc-worker-2",
		Tags: []string{"tag2", "tag3"},
	}

	context := &TemplateContext{
		catalogServices: map[string][]*dependency.CatalogService{
			"": []*dependency.CatalogService{catalogWeb1, catalogWeb2},
		},
	}

	contents, err := template.Execute(context)
	if err != nil {
		t.Fatal(err)
	}

	expected := bytes.TrimSpace([]byte(`
    nyc-worker-1: tag1,tag2,|nyc-worker-2: tag2,tag3,|
  `))

	if !bytes.Equal(bytes.TrimSpace(contents), expected) {
		t.Errorf("expected \n%q\n to equal \n%q\n", bytes.TrimSpace(contents), expected)
	}
}

func TestExecute_rendersTree(t *testing.T) {
	inTemplate := test.CreateTempfile([]byte(`
    {{ range tree "service/redis/config" }}
    {{.Key}} {{.Value}}{{ end }}
  `), t)
	defer test.DeleteTempfile(inTemplate, t)

	template, err := NewTemplate(inTemplate.Name())
	if err != nil {
		t.Fatal(err)
	}

	minconnsConfig := &dependency.KeyPair{
		Key:   "minconns",
		Value: "2",
	}

	maxconnsConfig := &dependency.KeyPair{
		Key:   "maxconns",
		Value: "11",
	}

	childConfig := &dependency.KeyPair{
		Key:   "user/sethvargo",
		Value: "true",
	}

	emptyFolderConfig := &dependency.KeyPair{
		Key:   "",
		Value: "",
	}

	emptyChildFolderConfig := &dependency.KeyPair{
		Key:   "user/",
		Value: "",
	}

	context := &TemplateContext{
		storeKeyPrefixes: map[string][]*dependency.KeyPair{
			"service/redis/config": []*dependency.KeyPair{
				minconnsConfig,
				maxconnsConfig,
				childConfig,
				emptyFolderConfig,
				emptyChildFolderConfig,
			},
		},
	}

	contents, err := template.Execute(context)
	if err != nil {
		t.Fatal(err)
	}

	expected := bytes.TrimSpace([]byte(`
    minconns 2
    maxconns 11
    user/sethvargo true
  `))
	if !bytes.Equal(bytes.TrimSpace(contents), expected) {
		t.Errorf("expected \n%q\n to equal \n%q\n", bytes.TrimSpace(contents), expected)
	}
}

func TestExecute_serviceTagsContains(t *testing.T) {
	inTemplate := test.CreateTempfile([]byte(`
		{{range service "web" }}
		{{.ID}}:
			{{if .Tags.Contains "auth"}}a{{else}}-{{end}}
			{{if .Tags.Contains "search"}}s{{else}}-{{end}}
			{{if .Tags.Contains "other"}}o{{else}}-{{end}}{{end}}
	`), t)
	defer test.DeleteTempfile(inTemplate, t)

	template, err := NewTemplate(inTemplate.Name())
	if err != nil {
		t.Fatal(err)
	}

	service1 := &dependency.HealthService{
		Node:    "nyc-api-1",
		Address: "127.0.0.1",
		ID:      "web1",
		Name:    "web1",
		Port:    1234,
		Tags:    []string{"auth", "search"},
	}
	service2 := &dependency.HealthService{
		Node:    "nyc-api-2",
		Address: "127.0.0.2",
		ID:      "web2",
		Name:    "web2",
		Port:    5678,
		Tags:    []string{"auth"},
	}

	context := &TemplateContext{
		healthServices: map[string][]*dependency.HealthService{
			"web [passing]": []*dependency.HealthService{service1, service2},
		},
	}

	contents, err := template.Execute(context)
	if err != nil {
		t.Fatal(err)
	}

	expected := bytes.TrimSpace([]byte(`
		web1:
			a
			s
			-
		web2:
			a
			-
			-
	`))

	if !bytes.Equal(bytes.TrimSpace(contents), expected) {
		t.Errorf("expected \n%q\n to equal \n%q\n", bytes.TrimSpace(contents), expected)
	}
}

func TestExecute_byTag(t *testing.T) {
	inTemplate := test.CreateTempfile([]byte(`
		{{range $t, $s := service "webapp" | byTag}}{{$t}}
		{{range $s}}	server {{.Name}} {{.Address}}:{{.Port}}
		{{end}}{{end}}
	`), t)
	defer test.DeleteTempfile(inTemplate, t)

	template, err := NewTemplate(inTemplate.Name())
	if err != nil {
		t.Fatal(err)
	}

	serviceWeb1 := &dependency.HealthService{
		Node:    "nyc-api-1",
		Address: "127.0.0.1",
		ID:      "web1",
		Name:    "web1",
		Port:    1234,
		Tags:    []string{"auth", "search"},
	}

	serviceWeb2 := &dependency.HealthService{
		Node:    "nyc-api-2",
		Address: "127.0.0.2",
		ID:      "web2",
		Name:    "web2",
		Port:    5678,
		Tags:    []string{"search"},
	}

	serviceWeb3 := &dependency.HealthService{
		Node:    "nyc-api-3",
		Address: "127.0.0.3",
		ID:      "web3",
		Name:    "web3",
		Port:    9012,
		Tags:    []string{"metric"},
	}

	context := &TemplateContext{
		healthServices: map[string][]*dependency.HealthService{
			"webapp [passing]": []*dependency.HealthService{serviceWeb1, serviceWeb2, serviceWeb3},
		},
	}

	contents, err := template.Execute(context)
	if err != nil {
		t.Fatal(err)
	}

	expected := bytes.TrimSpace([]byte(`
		auth
			server web1 127.0.0.1:1234
		metric
			server web3 127.0.0.3:9012
		search
			server web1 127.0.0.1:1234
			server web2 127.0.0.2:5678
	`))

	if !bytes.Equal(bytes.TrimSpace(contents), expected) {
		t.Errorf("expected \n%q\n to equal \n%q\n", bytes.TrimSpace(contents), expected)
	}
}

func TestExecute_env(t *testing.T) {
	inTemplate := test.CreateTempfile([]byte(`{{env "CONSUL_TEMPLATE_TESTVAR"}}`), t)
	defer test.DeleteTempfile(inTemplate, t)

	template, err := NewTemplate(inTemplate.Name())
	if err != nil {
		t.Fatal(err)
	}

	os.Setenv("CONSUL_TEMPLATE_TESTVAR", "F0F0F0")
	contents, err := template.Execute(&TemplateContext{})
	if err != nil {
		t.Fatal(err)
	}

	expected := []byte("F0F0F0")
	if !bytes.Equal(contents, expected) {
		t.Fatalf("expected %q to be %q", contents, expected)
	}
}

func TestExecute_parseJSON(t *testing.T) {
	inTemplate := test.CreateTempfile([]byte(`
		{{with $d := file "data.json" | parseJSON}}
		{{$d.foo}}
		{{end}}
	`), t)
	defer test.DeleteTempfile(inTemplate, t)

	template, err := NewTemplate(inTemplate.Name())
	if err != nil {
		t.Fatal(err)
	}

	context := &TemplateContext{
		files: map[string]string{
			"data.json": `{"foo":"bar"}`,
		},
	}

	data, err := template.Execute(context)
	if err != nil {
		t.Fatal(err)
	}

	result, expected := bytes.TrimSpace(data), []byte("bar")
	if !bytes.Equal(result, expected) {
		t.Errorf("expected %q to equal %q", result, expected)
	}
}

func TestExecute_parseJSONArray(t *testing.T) {
	inTemplate := test.CreateTempfile([]byte(`
		{{with $d := file "data.json" | parseJSON}}
		{{range $i := $d}}{{$i}}{{end}}
		{{end}}
	`), t)
	defer test.DeleteTempfile(inTemplate, t)

	template, err := NewTemplate(inTemplate.Name())
	if err != nil {
		t.Fatal(err)
	}

	context := &TemplateContext{
		files: map[string]string{
			"data.json": `["1", "2", "3"]`,
		},
	}
	data, err := template.Execute(context)
	if err != nil {
		t.Fatal(err)
	}
	result, expected := bytes.TrimSpace(data), []byte("123")
	if !bytes.Equal(result, expected) {
		t.Errorf("expected %q to equal %q", result, expected)
	}
}

func TestExecute_parseJSONDeep(t *testing.T) {
	inTemplate := test.CreateTempfile([]byte(`
		{{with $d := file "data.json" | parseJSON}}
		{{$d.foo.bar.zip}}
		{{end}}
	`), t)
	defer test.DeleteTempfile(inTemplate, t)

	template, err := NewTemplate(inTemplate.Name())
	if err != nil {
		t.Fatal(err)
	}

	context := &TemplateContext{
		files: map[string]string{
			"data.json": `{
				"foo": {
					"bar": {
						"zip": "zap"
					}
				}
			}`,
		},
	}

	data, err := template.Execute(context)
	if err != nil {
		t.Fatal(err)
	}

	result, expected := bytes.TrimSpace(data), []byte("zap")
	if !bytes.Equal(result, expected) {
		t.Errorf("expected %q to equal %q", result, expected)
	}
}

func TestExecute_regexReplaceAll(t *testing.T) {
	inTemplate := test.CreateTempfile([]byte(`{{"random:name:532" | regexReplaceAll ":(name):" "($1)"}}`), t)
	defer test.DeleteTempfile(inTemplate, t)

	template, err := NewTemplate(inTemplate.Name())
	if err != nil {
		t.Fatal(err)
	}

	contents, err := template.Execute(&TemplateContext{})
	if err != nil {
		t.Fatal(err)
	}

	expected := []byte("random(name)532")
	if !bytes.Equal(contents, expected) {
		t.Fatalf("expected %q to be %q", contents, expected)
	}
}

func TestExecute_replaceAll(t *testing.T) {
	inTemplate := test.CreateTempfile([]byte(`{{"random:name:532" | replaceAll ":" "_"}}`), t)
	defer test.DeleteTempfile(inTemplate, t)

	template, err := NewTemplate(inTemplate.Name())
	if err != nil {
		t.Fatal(err)
	}

	contents, err := template.Execute(&TemplateContext{})
	if err != nil {
		t.Fatal(err)
	}

	expected := []byte("random_name_532")
	if !bytes.Equal(contents, expected) {
		t.Fatalf("expected %q to be %q", contents, expected)
	}
}

func TestExecute_toLower(t *testing.T) {
	inTemplate := test.CreateTempfile([]byte(`{{"BACON" | toLower}}`), t)
	defer test.DeleteTempfile(inTemplate, t)

	template, err := NewTemplate(inTemplate.Name())
	if err != nil {
		t.Fatal(err)
	}

	contents, err := template.Execute(&TemplateContext{})
	if err != nil {
		t.Fatal(err)
	}

	expected := []byte("bacon")
	if !bytes.Equal(contents, expected) {
		t.Fatalf("expected %q to be %q", contents, expected)
	}
}

func TestExecute_toTitle(t *testing.T) {
	inTemplate := test.CreateTempfile([]byte(`{{"eat more bacon" | toTitle}}`), t)
	defer test.DeleteTempfile(inTemplate, t)

	template, err := NewTemplate(inTemplate.Name())
	if err != nil {
		t.Fatal(err)
	}

	contents, err := template.Execute(&TemplateContext{})
	if err != nil {
		t.Fatal(err)
	}

	expected := []byte("Eat More Bacon")
	if !bytes.Equal(contents, expected) {
		t.Fatalf("expected %q to be %q", contents, expected)
	}
}

func TestExecute_toUpper(t *testing.T) {
	inTemplate := test.CreateTempfile([]byte(`{{"bacon" | toUpper}}`), t)
	defer test.DeleteTempfile(inTemplate, t)

	template, err := NewTemplate(inTemplate.Name())
	if err != nil {
		t.Fatal(err)
	}

	contents, err := template.Execute(&TemplateContext{})
	if err != nil {
		t.Fatal(err)
	}

	expected := []byte("BACON")
	if !bytes.Equal(contents, expected) {
		t.Fatalf("expected %q to be %q", contents, expected)
	}
}
