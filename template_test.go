package main

import (
	"bytes"
	"io/ioutil"
	"os"
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
		t.Errorf("%s", err)
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
		t.Errorf("%s", err)
	}

	if num := len(dependencies); num != 3 {
		t.Fatalf("expected 3 dependencies, got: %d", num)
	}

	service := dependencies[0]
	if service.Type != DependencyTypeService {
		t.Errorf("expected dependencies[0] to be service, was: %#v", service.Type)
	}
	if service.Value != "release.webapp" {
		t.Errorf("expected service value to be release.webapp, was: %s", service.Value)
	}

	key := dependencies[1]
	if key.Type != DependencyTypeKey {
		t.Errorf("expected dependencies[1] to be key, was: %#v", key.Type)
	}
	if key.Value != "service/redis/maxconns" {
		t.Errorf("expected key value to be service/redis/maxconns, was: %s", key.Value)
	}

	keyPrefix := dependencies[2]
	if keyPrefix.Type != DependencyTypeKeyPrefix {
		t.Errorf("expected dependencies[2] to be keyPrefix, was: %#v", keyPrefix.Type)
	}
	if keyPrefix.Value != "service/redis/config" {
		t.Errorf("expected keyPrefix value to be service/redis/config, was: %s", keyPrefix.Value)
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

/*
 * Helpers
 */
func createTempfile(b []byte, t *testing.T) *os.File {
	f, err := ioutil.TempFile(os.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}

	if len(b) > 0 {
		_, err = f.Write(b)
		if err != nil {
			t.Fatal(err)
		}
	}

	return f
}

func deleteTempfile(f *os.File, t *testing.T) {
	if err := os.Remove(f.Name()); err != nil {
		t.Fatal(err)
	}
}
