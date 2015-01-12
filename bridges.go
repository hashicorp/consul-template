package main

import (
	"fmt"

	"github.com/hashicorp/consul-template/dependency"
)

type dependencyContextBridge interface {
	dependency.Dependency
	addToContext(*TemplateContext, interface{}) error
	inContext(*TemplateContext) bool
}

// catalogServicesBridge is a bridged interface with extra helpers for
// adding and removing items from a TemplateContext.
type catalogServicesBridge struct {
	*dependency.CatalogServices
}

// addToContext accepts a TemplateContext and data. It coerces the interface{}
// data into the correct format via type assertions, returning an errors that
// occur. The data is then set on the TemplateContext.
func (d *catalogServicesBridge) addToContext(c *TemplateContext, data interface{}) error {
	coerced, ok := data.([]*dependency.CatalogService)
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
type fileBridge struct{ *dependency.File }

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

// storeKeyBridge is a bridged interface with extra helpers for
// adding and removing items from a TemplateContext.
type storeKeyBridge struct{ *dependency.StoreKey }

// addToContext accepts a TemplateContext and data. It coerces the interface{}
// data into the correct format via type assertions, returning an errors that
// occur. The data is then set on the TemplateContext.
func (d *storeKeyBridge) addToContext(c *TemplateContext, data interface{}) error {
	coerced, ok := data.(string)
	if !ok {
		return fmt.Errorf("key dependency: could not convert to string")
	}

	c.storeKeys[d.Key()] = coerced
	return nil
}

// inContext checks if the dependency is contained in the given TemplateContext.
func (d *storeKeyBridge) inContext(c *TemplateContext) bool {
	_, ok := c.storeKeys[d.Key()]
	return ok
}

// storeKeyPrefixBridge is a bridged interface with extra helpers for
// adding and removing items from a TemplateContext.
type storeKeyPrefixBridge struct{ *dependency.StoreKeyPrefix }

// data into the correct format via type assertions, returning an errors that
// occur. The data is then set on the TemplateContext.
func (d *storeKeyPrefixBridge) addToContext(c *TemplateContext, data interface{}) error {
	coerced, ok := data.([]*dependency.KeyPair)
	if !ok {
		return fmt.Errorf("key prefix dependency: could not convert to KeyPair")
	}

	c.storeKeyPrefixes[d.Key()] = coerced
	return nil
}

// InContext checks if the dependency is contained in the given TemplateContext.
func (d *storeKeyPrefixBridge) inContext(c *TemplateContext) bool {
	_, ok := c.storeKeyPrefixes[d.Key()]
	return ok
}

// catalogNodesBridge is a bridged interface with extra helpers for
// adding and removing items from a TemplateContext.
type catalogNodesBridge struct{ *dependency.CatalogNodes }

// addToContext accepts a TemplateContext and data. It coerces the interface{}
// data into the correct format via type assertions, returning an errors that
// occur. The data is then set on the TemplateContext.
func (d *catalogNodesBridge) addToContext(c *TemplateContext, data interface{}) error {
	coerced, ok := data.([]*dependency.Node)
	if !ok {
		return fmt.Errorf("nodes dependency: could not convert to Node")
	}

	c.catalogNodes[d.Key()] = coerced
	return nil
}

// inContext checks if the dependency is contained in the given TemplateContext.
func (d *catalogNodesBridge) inContext(c *TemplateContext) bool {
	_, ok := c.catalogNodes[d.Key()]
	return ok
}

// serviceDependencyBridge is a bridged interface with extra helpers for
// adding and removing items from a TemplateContext.
type serviceDependencyBridge struct{ *dependency.HealthServices }

// addToContext accepts a TemplateContext and data. It coerces the interface{}
// data into the correct format via type assertions, returning an errors that
// occur. The data is then set on the TemplateContext.
func (d *serviceDependencyBridge) addToContext(c *TemplateContext, data interface{}) error {
	coerced, ok := data.([]*dependency.HealthService)
	if !ok {
		return fmt.Errorf("service dependency: could not convert to Service")
	}

	c.healthServices[d.Key()] = coerced
	return nil
}

// inContext checks if the dependency is contained in the given TemplateContext.
func (d *serviceDependencyBridge) inContext(c *TemplateContext) bool {
	_, ok := c.healthServices[d.Key()]
	return ok
}

// datacentersDependencyBridge is a bridged interface with extra
// helpers for adding and removing items from a TemplateContext
type datacentersDependencyBridge struct{ *dependency.Datacenters }

// addToContext accepts a TemplateContext and data. It coerces the interface{}
// data into the correct format via type assertions, returning any errors that
// occur. The data is then set on the TemplateContext
func (d *datacentersDependencyBridge) addToContext(c *TemplateContext, data interface{}) error {
	coerced, ok := data.([]string)
	if !ok {
		return fmt.Errorf("key prefix dependency: could not convert to Datacenters")
	}

	c.datacenters[d.Key()] = coerced
	return nil
}

// inContext checks if the dependency is contained in the given TemplateContext.
func (d *datacentersDependencyBridge) inContext(c *TemplateContext) bool {
	_, ok := c.datacenters[d.Key()]
	return ok
}
