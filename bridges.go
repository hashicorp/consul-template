package main

import (
	"fmt"

	"github.com/hashicorp/consul-template/util"
)

type dependencyContextBridge interface {
	util.Dependency
	addToContext(*TemplateContext, interface{}) error
	inContext(*TemplateContext) bool
}

// catalogServicesBridge is a bridged interface with extra helpers for
// adding and removing items from a TemplateContext.
type catalogServicesBridge struct {
	*util.CatalogServices
}

// addToContext accepts a TemplateContext and data. It coerces the interface{}
// data into the correct format via type assertions, returning an errors that
// occur. The data is then set on the TemplateContext.
func (d *catalogServicesBridge) addToContext(c *TemplateContext, data interface{}) error {
	coerced, ok := data.([]*util.CatalogService)
	if !ok {
		return fmt.Errorf("services dependency: could not convert to CatalogService")
	}

	c.catalogServices[d.Key()] = coerced
	return nil
}

// inContext checks if the dependency is contained in the given TemplateContext.
func (d *catalogServicesBridge) inContext(c *TemplateContext) bool {
	_, ok := c.catalogServices[d.Key()]
	return ok
}

// fileBridge is a bridged interface with extra helpers for
// adding and removing items from a TemplateContext.
type fileBridge struct{ *util.File }

// addToContext accepts a TemplateContext and data. It coerces the interface{}
// data into the correct format via type assertions, returning an errors that
// occur. The data is then set on the TemplateContext.
func (d *fileBridge) addToContext(c *TemplateContext, data interface{}) error {
	coerced, ok := data.(string)
	if !ok {
		return fmt.Errorf("file dependency: could not convert to string")
	}

	c.files[d.Key()] = coerced
	return nil
}

// inContext checks if the dependency is contained in the given TemplateContext.
func (d *fileBridge) inContext(c *TemplateContext) bool {
	_, ok := c.files[d.Key()]
	return ok
}

// keyDependencyBridge is a bridged interface with extra helpers for
// adding and removing items from a TemplateContext.
type keyDependencyBridge struct{ *util.KeyDependency }

// addToContext accepts a TemplateContext and data. It coerces the interface{}
// data into the correct format via type assertions, returning an errors that
// occur. The data is then set on the TemplateContext.
func (d *keyDependencyBridge) addToContext(c *TemplateContext, data interface{}) error {
	coerced, ok := data.(string)
	if !ok {
		return fmt.Errorf("key dependency: could not convert to string")
	}

	c.keys[d.Key()] = coerced
	return nil
}

// inContext checks if the dependency is contained in the given TemplateContext.
func (d *keyDependencyBridge) inContext(c *TemplateContext) bool {
	_, ok := c.keys[d.Key()]
	return ok
}

// keyPrefixBridge is a bridged interface with extra helpers for
// adding and removing items from a TemplateContext.
type keyPrefixBridge struct{ *util.KeyPrefix }

// data into the correct format via type assertions, returning an errors that
// occur. The data is then set on the TemplateContext.
func (d *keyPrefixBridge) addToContext(c *TemplateContext, data interface{}) error {
	coerced, ok := data.([]*util.KeyPair)
	if !ok {
		return fmt.Errorf("key prefix dependency: could not convert to KeyPair")
	}

	c.keyPrefixes[d.Key()] = coerced
	return nil
}

// InContext checks if the dependency is contained in the given TemplateContext.
func (d *keyPrefixBridge) inContext(c *TemplateContext) bool {
	_, ok := c.keyPrefixes[d.Key()]
	return ok
}

// nodesDependencyBridge is a bridged interface with extra helpers for
// adding and removing items from a TemplateContext.
type nodesDependencyBridge struct{ *util.NodesDependency }

// addToContext accepts a TemplateContext and data. It coerces the interface{}
// data into the correct format via type assertions, returning an errors that
// occur. The data is then set on the TemplateContext.
func (d *nodesDependencyBridge) addToContext(c *TemplateContext, data interface{}) error {
	coerced, ok := data.([]*util.Node)
	if !ok {
		return fmt.Errorf("nodes dependency: could not convert to Node")
	}

	c.nodes[d.Key()] = coerced
	return nil
}

// inContext checks if the dependency is contained in the given TemplateContext.
func (d *nodesDependencyBridge) inContext(c *TemplateContext) bool {
	_, ok := c.nodes[d.Key()]
	return ok
}

// serviceDependencyBridge is a bridged interface with extra helpers for
// adding and removing items from a TemplateContext.
type serviceDependencyBridge struct{ *util.HealthServices }

// addToContext accepts a TemplateContext and data. It coerces the interface{}
// data into the correct format via type assertions, returning an errors that
// occur. The data is then set on the TemplateContext.
func (d *serviceDependencyBridge) addToContext(c *TemplateContext, data interface{}) error {
	coerced, ok := data.([]*util.HealthService)
	if !ok {
		return fmt.Errorf("service dependency: could not convert to Service")
	}

	c.services[d.Key()] = coerced
	return nil
}

// inContext checks if the dependency is contained in the given TemplateContext.
func (d *serviceDependencyBridge) inContext(c *TemplateContext) bool {
	_, ok := c.services[d.Key()]
	return ok
}
