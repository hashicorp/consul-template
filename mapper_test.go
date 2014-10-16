package main

import (
	"strings"
	"testing"
)

func TestNewMapper_emptyTemplate(t *testing.T) {
	_, err := NewMapper(nil)
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "mapper: must supply at least one ConfigTemplate"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q to contain %q", err.Error(), expected)
	}
}

func TestNewMapper_singleConfigTemplate(t *testing.T) {
	inTemplate := createTempfile([]byte(`
    {{ range service "consul@nyc1"}}{{end}}
  `), t)
	defer deleteTempfile(inTemplate, t)

	ctemplates := []*ConfigTemplate{
		&ConfigTemplate{Source: inTemplate.Name()},
	}

	m, err := NewMapper(ctemplates)
	if err != nil {
		t.Fatal(err)
	}

	if len(m.Dependencies()) != 1 {
		t.Errorf("expected 1 Dependency, got %d", len(m.Dependencies()))
	}

	if len(m.Templates()) != 1 {
		t.Errorf("expected 1 Template, got %d", len(m.Templates()))
	}

	if len(m.ConfigTemplates()) != 1 {
		t.Errorf("expected 1 ConfigTemplate, got %d", len(m.ConfigTemplates()))
	}

	// These tests are basically "no panic == success"
	m.DependenciesFor(m.Templates()[0])
	m.TemplatesFor(m.Dependencies()[0])
	m.ConfigTemplatesFor(m.Templates()[0])
}

func TestNewMapper_multipleConfigTemplate(t *testing.T) {
	inTemplate1 := createTempfile([]byte(`
    {{ range service "consul@nyc1"}}{{end}}
  `), t)
	defer deleteTempfile(inTemplate1, t)

	inTemplate2 := createTempfile([]byte(`
    {{ range service "consul@nyc2"}}{{end}}
  `), t)
	defer deleteTempfile(inTemplate2, t)

	inTemplate3 := createTempfile([]byte(`
    {{ range service "consul@nyc3"}}{{end}}
  `), t)
	defer deleteTempfile(inTemplate3, t)

	ctemplates := []*ConfigTemplate{
		&ConfigTemplate{Source: inTemplate1.Name()},
		&ConfigTemplate{Source: inTemplate2.Name()},
		&ConfigTemplate{Source: inTemplate3.Name()},
	}

	m, err := NewMapper(ctemplates)
	if err != nil {
		t.Fatal(err)
	}

	if num := len(m.Dependencies()); num != 3 {
		t.Errorf("expected 3 Dependency, got %d", num)
	}

	if num := len(m.Templates()); num != 3 {
		t.Errorf("expected 3 Template, got %d", num)
	}

	if num := len(m.ConfigTemplates()); num != 3 {
		t.Errorf("expected 3 ConfigTemplate, got %d", num)
	}

	for _, template := range m.Templates() {
		if dependencies := m.DependenciesFor(template); len(dependencies) != 1 {
			t.Errorf("expected 1 Dependency, got %d", len(dependencies))
		}
	}

	for _, dependency := range m.Dependencies() {
		if templates := m.TemplatesFor(dependency); len(templates) != 1 {
			t.Errorf("expected 1 Template, got %d", len(templates))
		}
	}

	for _, template := range m.Templates() {
		if ctemplates := m.ConfigTemplatesFor(template); len(ctemplates) != 1 {
			t.Errorf("expected 1 ConfigTemplate, got %d", len(ctemplates))
		}
	}
}

func TestNewMapper_templateWithMultipleDependency(t *testing.T) {
	inTemplate := createTempfile([]byte(`
    {{ range service "consul@nyc1"}}{{end}}
    {{ range service "consul@nyc2"}}{{end}}
    {{ range service "consul@nyc3"}}{{end}}
  `), t)
	defer deleteTempfile(inTemplate, t)

	ctemplates := []*ConfigTemplate{
		&ConfigTemplate{Source: inTemplate.Name()},
	}

	m, err := NewMapper(ctemplates)
	if err != nil {
		t.Fatal(err)
	}

	if num := len(m.Dependencies()); num != 3 {
		t.Errorf("expected 3 Dependency, got %d", num)
	}

	if num := len(m.Templates()); num != 1 {
		t.Errorf("expected 1 Template, got %d", num)
	}

	if num := len(m.ConfigTemplates()); num != 1 {
		t.Errorf("expected 1 ConfigTemplate, got %d", num)
	}

	if template := m.Templates()[0]; template != nil {
		if num := len(m.DependenciesFor(template)); num != 3 {
			t.Errorf("expected 3 Dependency, got %d", num)
		}
	} else {
		t.Fatal("expected Template, but nothing was returned")
	}

	for _, dependency := range m.Dependencies() {
		if num := len(m.TemplatesFor(dependency)); num != 1 {
			t.Errorf("expected 1 Template, got %d", num)
		}
	}

	for _, template := range m.Templates() {
		if ctemplates := m.ConfigTemplatesFor(template); len(ctemplates) != 1 {
			t.Errorf("expected 1 ConfigTemplate, got %d", len(ctemplates))
		}
	}
}

func TestNewMapper_templatesWithDuplicateDependency(t *testing.T) {
	inTemplate := createTempfile([]byte(`
    {{ range service "consul@nyc1"}}{{end}}
    {{ range service "consul@nyc1"}}{{end}}
    {{ range service "consul@nyc1"}}{{end}}
  `), t)
	defer deleteTempfile(inTemplate, t)

	ctemplates := []*ConfigTemplate{
		&ConfigTemplate{Source: inTemplate.Name()},
	}

	m, err := NewMapper(ctemplates)
	if err != nil {
		t.Fatal(err)
	}

	if num := len(m.Dependencies()); num != 1 {
		t.Errorf("expected 1 Dependency, got %d", num)
	}

	if num := len(m.Templates()); num != 1 {
		t.Errorf("expected 1 Template, got %d", num)
	}

	if num := len(m.ConfigTemplates()); num != 1 {
		t.Errorf("expected 1 ConfigTemplate, got %d", num)
	}

	for _, template := range m.Templates() {
		if dependencies := m.DependenciesFor(template); len(dependencies) != 1 {
			t.Errorf("expected 1 Dependency, got %d", len(dependencies))
		}
	}

	for _, dependency := range m.Dependencies() {
		if templates := m.TemplatesFor(dependency); len(templates) != 1 {
			t.Errorf("expected 1 Template, got %d", len(templates))
		}
	}

	for _, template := range m.Templates() {
		if ctemplates := m.ConfigTemplatesFor(template); len(ctemplates) != 1 {
			t.Errorf("expected 1 ConfigTemplate, got %d", len(ctemplates))
		}
	}
}

func TestNewMapper_multipleTemplatesWithDuplicateDependency(t *testing.T) {
	inTemplate1 := createTempfile([]byte(`
    {{ range service "consul@nyc1"}}{{end}}
  `), t)
	defer deleteTempfile(inTemplate1, t)
	inTemplate2 := createTempfile([]byte(`
    {{ range service "consul@nyc1"}}{{end}}
  `), t)
	defer deleteTempfile(inTemplate2, t)

	ctemplates := []*ConfigTemplate{
		&ConfigTemplate{Source: inTemplate1.Name()},
		&ConfigTemplate{Source: inTemplate2.Name()},
	}

	m, err := NewMapper(ctemplates)
	if err != nil {
		t.Fatal(err)
	}

	if num := len(m.Dependencies()); num != 1 {
		t.Errorf("expected 1 Dependency, got %d", num)
	}

	if num := len(m.Templates()); num != 2 {
		t.Errorf("expected 2 Template, got %d", num)
	}

	if num := len(m.ConfigTemplates()); num != 2 {
		t.Errorf("expected 2 ConfigTemplate, got %d", num)
	}

	for _, template := range m.Templates() {
		if dependencies := m.DependenciesFor(template); len(dependencies) != 1 {
			t.Errorf("expected 1 Dependency, got %d", len(dependencies))
		}
	}

	for _, dependency := range m.Dependencies() {
		if templates := m.TemplatesFor(dependency); len(templates) != 2 {
			t.Errorf("expected 2 Template, got %d", len(templates))
		}
	}

	for _, template := range m.Templates() {
		if ctemplates := m.ConfigTemplatesFor(template); len(ctemplates) != 1 {
			t.Errorf("expected 1 ConfigTemplate, got %d", len(ctemplates))
		}
	}
}

func TestNewMapper_multipleTemplatesWithMultipleDependencies(t *testing.T) {
	inTemplate1 := createTempfile([]byte(`
    {{ range service "consul@nyc1"}}{{end}}
    {{ range service "consul@nyc2"}}{{end}}
  `), t)
	defer deleteTempfile(inTemplate1, t)
	inTemplate2 := createTempfile([]byte(`
    {{ range service "consul@nyc2"}}{{end}}
    {{ range service "consul@nyc3"}}{{end}}
  `), t)
	defer deleteTempfile(inTemplate2, t)
	inTemplate3 := createTempfile([]byte(`
    {{ range service "consul@nyc3"}}{{end}}
    {{ range service "consul@nyc4"}}{{end}}
  `), t)
	defer deleteTempfile(inTemplate3, t)

	ctemplates := []*ConfigTemplate{
		&ConfigTemplate{Source: inTemplate1.Name()},
		&ConfigTemplate{Source: inTemplate2.Name()},
		&ConfigTemplate{Source: inTemplate3.Name()},
	}

	m, err := NewMapper(ctemplates)
	if err != nil {
		t.Fatal(err)
	}

	if num := len(m.Dependencies()); num != 4 {
		t.Errorf("expected 4 Dependency, got %d", num)
	}

	if num := len(m.Templates()); num != 3 {
		t.Errorf("expected 3 Template, got %d", num)
	}

	if num := len(m.ConfigTemplates()); num != 3 {
		t.Errorf("expected 3 ConfigTemplate, got %d", num)
	}

	for _, template := range m.Templates() {
		if dependencies := m.DependenciesFor(template); len(dependencies) != 2 {
			t.Errorf("expected 2 Dependency, got %d", len(dependencies))
		}
	}

	for _, dependency := range m.Dependencies() {
		if templates := m.TemplatesFor(dependency); len(templates) < 1 {
			t.Errorf("expected at least 1 Template, got %d", len(templates))
		}
	}

	for _, template := range m.Templates() {
		if ctemplates := m.ConfigTemplatesFor(template); len(ctemplates) != 1 {
			t.Errorf("expected 1 ConfigTemplate, got %d", len(ctemplates))
		}
	}
}

func TestNewMapper_multipleConfigTemplateSameTemplate(t *testing.T) {
	inTemplate := createTempfile([]byte(`
    {{ range service "consul@nyc1"}}{{end}}
  `), t)
	defer deleteTempfile(inTemplate, t)

	ctemplates := []*ConfigTemplate{
		&ConfigTemplate{Source: inTemplate.Name()},
		&ConfigTemplate{Source: inTemplate.Name()},
	}

	m, err := NewMapper(ctemplates)
	if err != nil {
		t.Fatal(err)
	}

	if num := len(m.Dependencies()); num != 1 {
		t.Errorf("expected 1 Dependency, got %d", num)
	}

	if num := len(m.Templates()); num != 1 {
		t.Errorf("expected 1 Template, got %d", num)
	}

	if num := len(m.ConfigTemplates()); num != 2 {
		t.Errorf("expected 2 ConfigTemplate, got %d", num)
	}

	for _, template := range m.Templates() {
		if dependencies := m.DependenciesFor(template); len(dependencies) != 1 {
			t.Errorf("expected 1 Dependency, got %d", len(dependencies))
		}
	}

	for _, dependency := range m.Dependencies() {
		if templates := m.TemplatesFor(dependency); len(templates) != 1 {
			t.Errorf("expected 1 Template, got %d", len(templates))
		}
	}

	for _, template := range m.Templates() {
		if ctemplates := m.ConfigTemplatesFor(template); len(ctemplates) != 2 {
			t.Errorf("expected 2 ConfigTemplate, got %d", len(ctemplates))
		}
	}
}
