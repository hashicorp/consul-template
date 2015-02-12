package main

import "strings"

// configTemplateVar implements the Flag.Value interface and allows the user
// to specify multiple -template keys in the CLI where each option is parsed
// as a template.
type configTemplateVar []*ConfigTemplate

func (ctv *configTemplateVar) Set(value string) error {
	template, err := ParseConfigTemplate(value)
	if err != nil {
		return err
	}

	if *ctv == nil {
		*ctv = make([]*ConfigTemplate, 0, 1)
	}
	*ctv = append(*ctv, template)

	return nil
}

func (ctv *configTemplateVar) String() string {
	return ""
}

//
type authVar Auth

func (a *authVar) Set(value string) error {
	if strings.Contains(value, ":") {
		split := strings.SplitN(value, ":", 2)
		a.Username = split[0]
		a.Password = split[1]
	} else {
		a.Username = value
	}

	return nil
}

func (a *authVar) String() string {
	return ""
}
