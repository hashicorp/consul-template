package main

import (
	"bytes"
	"io/ioutil"
	"strings"
	"testing"
)

// Test that the parser does not barf when the file is empty
func TestDependencies_empty(t *testing.T) {
	inTemplate := createTempfile(nil, t)
	defer deleteTempfile(inTemplate, t)

	tmpl := &Template{
		Input: inTemplate.Name(),
	}

	dependencies, err := tmpl.Dependencies()

	if err != nil {
		t.Fatal(err)
	}

	if num := len(dependencies); num != 0 {
		t.Errorf("expected 0 dependencies, got: %d", num)
	}
}

// Test that the parser adds services, keys, and key prefixes
func TestDependencies_funcs(t *testing.T) {
	inTemplate := createTempfile([]byte(`
    {{ range service "release.webapp" }}{{ end }}
    {{ key "service/redis/maxconns" }}
    {{ range keyPrefix "service/redis/config" }}{{ end }}
  `), t)
	defer deleteTempfile(inTemplate, t)

	tmpl := &Template{
		Input: inTemplate.Name(),
	}

	dependencies, err := tmpl.Dependencies()

	if err != nil {
		t.Fatal(err)
	}

	if num := len(dependencies); num != 3 {
		t.Fatalf("expected 3 dependencies, got: %d", num)
	}

	dependency, expected := dependencies[0], "release.webapp"
	if dependency.Key() != expected {
		t.Errorf("expected %q to equal %q", dependency.Key(), expected)
	}

	dependency, expected = dependencies[1], "service/redis/maxconns"
	if dependency.Key() != expected {
		t.Errorf("expected %q to equal %q", dependency.Key(), expected)
	}

	dependency, expected = dependencies[2], "service/redis/config"
	if dependency.Key() != expected {
		t.Errorf("expected %q to equal %q", dependency.Key(), expected)
	}
}

// Test that any errors raised while parsing the dependencies are propagated up
func TestDependencies_funcsError(t *testing.T) {
	inTemplate := createTempfile([]byte(`
    {{ range service "totally/not/a/valid/service" }}{{ end }}
  `), t)
	defer deleteTempfile(inTemplate, t)

	tmpl := &Template{
		Input: inTemplate.Name(),
	}

	_, err := tmpl.Dependencies()
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "error calling service:"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q to contain %q", err.Error(), expected)
	}
}

// Test that an error is returned when no ioWriter is given
func TestExecute_noIOWriter(t *testing.T) {
	var tmpl Template
	var context TemplateContext

	err := tmpl.Execute(nil, &context)
	if err == nil {
		t.Fatal("should get error")
	}

	expectedError := "wr must be given"
	if err.Error() != expectedError {
		t.Errorf("expected error to be `%s', but was `%s'", expectedError, err.Error())
	}
}

// Test that an error is returned when no context is given
func TestExecute_noTemplateContext(t *testing.T) {
	var tmpl Template
	var wr bytes.Buffer

	err := tmpl.Execute(&wr, nil)
	if err == nil {
		t.Fatal("should get error")
	}

	expectedError := "templateContext must be given"
	if err.Error() != expectedError {
		t.Errorf("expected error to be `%s', but was `%s'", expectedError, err.Error())
	}
}

// Test that an error raised while fetching the template's dependencies is
// propagated up
func TestExecute_dependenciesError(t *testing.T) {
	var io bytes.Buffer

	inTemplate := createTempfile([]byte(`
    {{ range not_a_valid "template" }}{{ end }}
  `), t)
	defer deleteTempfile(inTemplate, t)

	tmpl := &Template{
		Input: inTemplate.Name(),
	}

	err := tmpl.Execute(&io, &TemplateContext{})
	if err == nil {
		t.Fatal("should get error")
	}

	expectedErr := "template: out:2: function \"not_a_valid\" not defined"
	if err.Error() != expectedErr {
		t.Errorf("expected error to be `%s', but was: `%s'", expectedErr, err)
	}
}

// Test that an error is returned when the context is missing a service
func TestExecute_missingService(t *testing.T) {
	var io bytes.Buffer

	inTemplate := createTempfile([]byte(`
    {{ range service "release.webapp" }}{{ end }}
    {{ range service "production.webapp" }}{{ end }}
  `), t)
	defer deleteTempfile(inTemplate, t)

	tmpl := &Template{
		Input: inTemplate.Name(),
	}

	context := &TemplateContext{
		Services: map[string][]*Service{
			"release.webapp": []*Service{},
		},
	}

	err := tmpl.Execute(&io, context)
	if err == nil {
		t.Fatal("should get error")
	}

	expectedErr := "templateContext missing service `production.webapp'"
	if err.Error() != expectedErr {
		t.Errorf("expected error to be `%s', but was: `%s'", expectedErr, err)
	}
}

// Test that an error is returned when the context is missing a key
func TestExecute_missingKey(t *testing.T) {
	var io bytes.Buffer

	inTemplate := createTempfile([]byte(`
    {{ key "service/redis/maxconns" }}
    {{ key "service/redis/online" }}
  `), t)
	defer deleteTempfile(inTemplate, t)

	tmpl := &Template{
		Input: inTemplate.Name(),
	}

	context := &TemplateContext{
		Keys: map[string]string{
			"service/redis/maxconns": "3",
		},
	}

	err := tmpl.Execute(&io, context)
	if err == nil {
		t.Fatal("should get error")
	}

	expectedErr := "templateContext missing key `service/redis/online'"
	if err.Error() != expectedErr {
		t.Errorf("expected error to be `%s', but was: `%s'", expectedErr, err)
	}
}

// Test that an error is returned when the context is missing a key prefix
func TestExecute_missingKeyPrefix(t *testing.T) {
	var io bytes.Buffer

	inTemplate := createTempfile([]byte(`
    {{ range keyPrefix "service/redis/config" }}{{ end }}
    {{ range keyPrefix "service/nginx/config" }}{{ end }}
  `), t)
	defer deleteTempfile(inTemplate, t)

	tmpl := &Template{
		Input: inTemplate.Name(),
	}

	context := &TemplateContext{
		KeyPrefixes: map[string][]*KeyPair{
			"service/redis/config": []*KeyPair{},
		},
	}

	err := tmpl.Execute(&io, context)
	if err == nil {
		t.Fatal("should get error")
	}

	expectedErr := "templateContext missing keyPrefix `service/nginx/config'"
	if err.Error() != expectedErr {
		t.Errorf("expected error to be `%s', but was: `%s'", expectedErr, err)
	}
}

// Test that services are rendered in the template
func TestExecute_rendersServices(t *testing.T) {
	var io bytes.Buffer

	inTemplate := createTempfile([]byte(`
    {{ range service "release.webapp" }}
    server {{.Name}} {{.Address}}:{{.Port}}{{ end }}
  `), t)
	defer deleteTempfile(inTemplate, t)

	tmpl := &Template{
		Input: inTemplate.Name(),
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

	err := tmpl.Execute(&io, context)
	if err != nil {
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

// Test that keys are rendered in the template
func TestExecute_rendersKeys(t *testing.T) {
	var io bytes.Buffer

	inTemplate := createTempfile([]byte(`
    minconns: {{ key "service/redis/minconns" }}
    maxconns: {{ key "service/redis/maxconns" }}
  `), t)
	defer deleteTempfile(inTemplate, t)

	tmpl := &Template{
		Input: inTemplate.Name(),
	}

	context := &TemplateContext{
		Keys: map[string]string{
			"service/redis/minconns": "2",
			"service/redis/maxconns": "11",
		},
	}

	err := tmpl.Execute(&io, context)
	if err != nil {
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

// Test that key prefixes are rendered in the template
func TestExecute_rendersKeyPrefixes(t *testing.T) {
	var io bytes.Buffer

	inTemplate := createTempfile([]byte(`
    {{ range keyPrefix "service/redis/config" }}
    {{.Key}} {{.Value}}{{ end }}
  `), t)
	defer deleteTempfile(inTemplate, t)

	tmpl := &Template{
		Input: inTemplate.Name(),
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

	err := tmpl.Execute(&io, context)
	if err != nil {
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
