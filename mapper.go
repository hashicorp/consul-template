package main

import (
	"errors"
)

// Mapper is responsible for keeping a map between templates and dependencies
// for querying.
type Mapper struct {
	//
	configTemplates []*ConfigTemplate
	templates       []*Template
	dependencies    []Dependency

	//
	templateConfigTemplateMap map[string][]*ConfigTemplate
	dependencyTemplateMap     map[string][]*Template
	templateDependencyMap     map[string][]Dependency
}

//
func NewMapper(configTemplates []*ConfigTemplate) (*Mapper, error) {
	mapper := &Mapper{configTemplates: configTemplates}
	if err := mapper.init(); err != nil {
		return nil, err
	}

	return mapper, nil
}

// ConfigTemplates returns the slice of ConfigTemplate in the Mapper
func (m *Mapper) ConfigTemplates() []*ConfigTemplate {
	return m.configTemplates
}

// Dependencies returns the unique slice of Dependency in the Mapper
func (m *Mapper) Dependencies() []Dependency {
	return m.dependencies
}

// Templates returns the slice of Template in the Mapper
func (m *Mapper) Templates() []*Template {
	return m.templates
}

// ConfigTemplateFor returns the ConfigTemplate for the given Template
func (m *Mapper) ConfigTemplatesFor(template *Template) []*ConfigTemplate {
	return m.templateConfigTemplateMap[template.HashCode()]
}

// DependenciesFor returns the slice of Dependency for the Template
func (m *Mapper) DependenciesFor(template *Template) []Dependency {
	return m.templateDependencyMap[template.HashCode()]
}

// TemplatesFor returns the slice of Template for the Dependency
func (m *Mapper) TemplatesFor(dependency Dependency) []*Template {
	return m.dependencyTemplateMap[dependency.HashCode()]
}

// init takes the slice of templates and constructs new internal data structures
// that may be queried for fast lookups.
func (m *Mapper) init() error {
	if len(m.configTemplates) == 0 {
		return errors.New("mapper: must supply at least one ConfigTemplate")
	}

	templatesMap := make(map[string]*Template)
	m.templateDependencyMap = make(map[string][]Dependency)

	dependenciesMap := make(map[string]Dependency)
	m.dependencyTemplateMap = make(map[string][]*Template)

	m.templateConfigTemplateMap = make(map[string][]*ConfigTemplate)

	// Process all Template first, so we can return errors
	for _, configTemplate := range m.configTemplates {
		template := &Template{path: configTemplate.Source}
		if _, ok := templatesMap[template.HashCode()]; !ok {
			template, err := NewTemplate(configTemplate.Source)
			if err != nil {
				return err
			}
			templatesMap[template.HashCode()] = template
		}

		if len(m.templateConfigTemplateMap[template.HashCode()]) == 0 {
			m.templateConfigTemplateMap[template.HashCode()] = make([]*ConfigTemplate, 0, 1)
		}
		m.templateConfigTemplateMap[template.HashCode()] = append(m.templateConfigTemplateMap[template.HashCode()], configTemplate)
	}

	// For each Template, setup the mappings for O(1) lookups
	for templateKey, template := range templatesMap {
		// Create the Template -> []Dependency map - this is already unique because
		// Template#Dependencies() returns the unique list.
		m.templateDependencyMap[templateKey] = template.Dependencies()

		// For each Dependency, map back to this Template
		for _, dep := range template.Dependencies() {
			// Add the Dependency to the unique slice of Dependencies
			if _, ok := dependenciesMap[dep.HashCode()]; !ok {
				dependenciesMap[dep.HashCode()] = dep
			}

			// Append to the Dependency -> []Template map
			if len(m.dependencyTemplateMap[dep.HashCode()]) == 0 {
				m.dependencyTemplateMap[dep.HashCode()] = make([]*Template, 0, 1)
			}
			m.dependencyTemplateMap[dep.HashCode()] = append(m.dependencyTemplateMap[dep.HashCode()], template)
		}
	}

	// Calculate the list of Templates
	m.templates = make([]*Template, 0, len(templatesMap))
	for _, template := range templatesMap {
		m.templates = append(m.templates, template)
	}

	// Calculate the list of Dependency
	m.dependencies = make([]Dependency, 0, len(dependenciesMap))
	for _, dependency := range dependenciesMap {
		m.dependencies = append(m.dependencies, dependency)
	}

	return nil
}
