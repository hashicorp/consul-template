package main

import (
	"bytes"
	"io/ioutil"
	"strings"
	"testing"
)

func TestDependencies_empty(t *testing.T) {
	inTemplate := createTempfile(nil, t)
	defer deleteTempfile(inTemplate, t)

	template, err := NewTemplate(inTemplate.Name())
	if err != nil {
		t.Fatal(err)
	}
	dependencies := template.Dependencies()

	if num := len(dependencies); num != 0 {
		t.Errorf("expected 0 dependencies, got: %d", num)
	}
}

func TestDependencies_funcs(t *testing.T) {
	inTemplate := createTempfile([]byte(`
    {{ range service "release.webapp" }}{{ end }}
    {{ key "service/redis/maxconns" }}
    {{ range keyPrefix "service/redis/config" }}{{ end }}
  `), t)
	defer deleteTempfile(inTemplate, t)

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
	inTemplate := createTempfile([]byte(`
    {{ range service "release.webapp" }}{{ end }}
    {{ range service "release.webapp" }}{{ end }}
    {{ range service "release.webapp" }}{{ end }}
  `), t)
	defer deleteTempfile(inTemplate, t)

	template, err := NewTemplate(inTemplate.Name())
	if err != nil {
		t.Fatal(err)
	}
	dependencies := template.Dependencies()

	if num := len(dependencies); num != 1 {
		t.Fatalf("expected 1 Dependency, got: %d", num)
	}

	dependency, expected := dependencies[0], "release.webapp"
	if dependency.Key() != expected {
		t.Errorf("expected %q to equal %q", dependency.Key(), expected)
	}
}

func TestDependencies_funcsError(t *testing.T) {
	inTemplate := createTempfile([]byte(`
    {{ range service "totally/not/a/valid/service" }}{{ end }}
  `), t)
	defer deleteTempfile(inTemplate, t)

	_, err := NewTemplate(inTemplate.Name())
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "error calling service:"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q to contain %q", err.Error(), expected)
	}
}

func TestExecute_noIOWriter(t *testing.T) {
	inTemplate := createTempfile(nil, t)
	defer deleteTempfile(inTemplate, t)

	template, err := NewTemplate(inTemplate.Name())
	if err != nil {
		t.Fatal(err)
	}

	executeErr := template.Execute(nil, &TemplateContext{})
	if executeErr == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "wr must be given"
	if !strings.Contains(executeErr.Error(), expected) {
		t.Errorf("expected %q to contain %q", executeErr.Error(), expected)
	}
}

func TestExecute_noTemplateContext(t *testing.T) {
	inTemplate := createTempfile(nil, t)
	defer deleteTempfile(inTemplate, t)

	template, err := NewTemplate(inTemplate.Name())
	if err != nil {
		t.Fatal(err)
	}

	var wr bytes.Buffer
	executeErr := template.Execute(&wr, nil)
	if executeErr == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "templateContext must be given"
	if !strings.Contains(executeErr.Error(), expected) {
		t.Errorf("expected %q to contain %q", executeErr.Error(), expected)
	}
}

func TestExecute_dependenciesError(t *testing.T) {
	inTemplate := createTempfile([]byte(`
    {{ range not_a_valid "template" }}{{ end }}
  `), t)
	defer deleteTempfile(inTemplate, t)

	_, err := NewTemplate(inTemplate.Name())
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := `template: out:2: function "not_a_valid" not defined`
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q to contain %q", err.Error(), expected)
	}
}

func TestExecute_missingService(t *testing.T) {
	var io bytes.Buffer

	inTemplate := createTempfile([]byte(`
    {{ range service "release.webapp" }}{{ end }}
    {{ range service "production.webapp" }}{{ end }}
  `), t)
	defer deleteTempfile(inTemplate, t)

	template, err := NewTemplate(inTemplate.Name())
	if err != nil {
		t.Fatal(err)
	}

	context := &TemplateContext{
		Services: map[string][]*Service{
			"release.webapp": []*Service{},
		},
	}

	executeErr := template.Execute(&io, context)
	if executeErr == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "templateContext missing service `production.webapp'"
	if !strings.Contains(executeErr.Error(), expected) {
		t.Errorf("expected %q to contain %q", executeErr.Error(), expected)
	}
}

func TestExecute_missingKey(t *testing.T) {
	var io bytes.Buffer

	inTemplate := createTempfile([]byte(`
    {{ key "service/redis/maxconns" }}
    {{ key "service/redis/online" }}
  `), t)
	defer deleteTempfile(inTemplate, t)

	template, err := NewTemplate(inTemplate.Name())
	if err != nil {
		t.Fatal(err)
	}

	context := &TemplateContext{
		Keys: map[string]string{
			"service/redis/maxconns": "3",
		},
	}

	executeErr := template.Execute(&io, context)
	if executeErr == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "templateContext missing key `service/redis/online'"
	if !strings.Contains(executeErr.Error(), expected) {
		t.Errorf("expected %q to contain %q", executeErr.Error(), expected)
	}
}

func TestExecute_missingKeyPrefix(t *testing.T) {
	var io bytes.Buffer

	inTemplate := createTempfile([]byte(`
    {{ range keyPrefix "service/redis/config" }}{{ end }}
    {{ range keyPrefix "service/nginx/config" }}{{ end }}
  `), t)
	defer deleteTempfile(inTemplate, t)

	template, err := NewTemplate(inTemplate.Name())
	if err != nil {
		t.Fatal(err)
	}

	context := &TemplateContext{
		KeyPrefixes: map[string][]*KeyPair{
			"service/redis/config": []*KeyPair{},
		},
	}

	executeErr := template.Execute(&io, context)
	if executeErr == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "templateContext missing keyPrefix `service/nginx/config'"
	if !strings.Contains(executeErr.Error(), expected) {
		t.Errorf("expected %q to contain %q", executeErr.Error(), expected)
	}
}

func TestExecute_rendersServices(t *testing.T) {
	var io bytes.Buffer

	inTemplate := createTempfile([]byte(`
    {{ range service "release.webapp" }}
    server {{.Name}} {{.Address}}:{{.Port}}{{ end }}
  `), t)
	defer deleteTempfile(inTemplate, t)

	template, err := NewTemplate(inTemplate.Name())
	if err != nil {
		t.Fatal(err)
	}

	serviceWeb1 := &Service{
		Node:    "nyc-worker-1",
		Address: "123.123.123.123",
		ID:      "web1",
		Name:    "web1",
		Port:    1234,
	}

	serviceWeb2 := &Service{
		Node:    "nyc-worker-2",
		Address: "456.456.456.456",
		ID:      "web2",
		Name:    "web2",
		Port:    5678,
	}

	context := &TemplateContext{
		Services: map[string][]*Service{
			"release.webapp": []*Service{serviceWeb1, serviceWeb2},
		},
	}

	if err := template.Execute(&io, context); err != nil {
		t.Fatal(err)
	}

	buffer, err := ioutil.ReadAll(&io)
	if err != nil {
		t.Fatal(err)
	}

	contents := strings.TrimSpace(string(buffer))
	expectedContents := "server web1 123.123.123.123:1234\n    server web2 456.456.456.456:5678"
	if contents != expectedContents {
		t.Errorf("expected contents to be:\n\n%#q\n\nbut was\n\n%#q\n", expectedContents, contents)
	}
}

func TestExecute_rendersKeys(t *testing.T) {
	var io bytes.Buffer

	inTemplate := createTempfile([]byte(`
    minconns: {{ key "service/redis/minconns" }}
    maxconns: {{ key "service/redis/maxconns" }}
  `), t)
	defer deleteTempfile(inTemplate, t)

	template, err := NewTemplate(inTemplate.Name())
	if err != nil {
		t.Fatal(err)
	}

	context := &TemplateContext{
		Keys: map[string]string{
			"service/redis/minconns": "2",
			"service/redis/maxconns": "11",
		},
	}

	if err := template.Execute(&io, context); err != nil {
		t.Fatal(err)
	}

	buffer, err := ioutil.ReadAll(&io)
	if err != nil {
		t.Fatal(err)
	}

	contents := strings.TrimSpace(string(buffer))
	expectedContents := "minconns: 2\n    maxconns: 11"
	if contents != expectedContents {
		t.Errorf("expected contents to be:\n\n%#q\n\nbut was\n\n%#q\n", expectedContents, contents)
	}
}

func TestExecute_rendersKeyPrefixes(t *testing.T) {
	var io bytes.Buffer

	inTemplate := createTempfile([]byte(`
    {{ range keyPrefix "service/redis/config" }}
    {{.Key}} {{.Value}}{{ end }}
  `), t)
	defer deleteTempfile(inTemplate, t)

	template, err := NewTemplate(inTemplate.Name())
	if err != nil {
		t.Fatal(err)
	}

	minconnsConfig := &KeyPair{
		Key:   "minconns",
		Value: "2",
	}

	maxconnsConfig := &KeyPair{
		Key:   "maxconns",
		Value: "11",
	}

	context := &TemplateContext{
		KeyPrefixes: map[string][]*KeyPair{
			"service/redis/config": []*KeyPair{minconnsConfig, maxconnsConfig},
		},
	}

	if err := template.Execute(&io, context); err != nil {
		t.Fatal(err)
	}

	buffer, err := ioutil.ReadAll(&io)
	if err != nil {
		t.Fatal(err)
	}

	contents := strings.TrimSpace(string(buffer))
	expectedContents := "minconns 2\n    maxconns 11"
	if contents != expectedContents {
		t.Errorf("expected contents to be:\n\n%#q\n\nbut was\n\n%#q\n", expectedContents, contents)
	}
}

func TestExecute_setsRendered(t *testing.T) {
	inTemplate := createTempfile(nil, t)
	defer deleteTempfile(inTemplate, t)

	template, err := NewTemplate(inTemplate.Name())
	if err != nil {
		t.Fatal(err)
	}

	if template.Rendered() {
		t.Fatal("template should not be rendered yet")
	}

	if err := template.Execute(ioutil.Discard, &TemplateContext{}); err != nil {
		t.Fatal(err)
	}

	if !template.Rendered() {
		t.Fatal("expected template to be rendered")
	}
}
