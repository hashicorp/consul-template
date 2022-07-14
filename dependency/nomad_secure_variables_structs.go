package dependency

import (
	"fmt"
	"sort"
	"time"

	"github.com/hashicorp/nomad/api"
)

// NewNomadSVMeta is used to create a NomadSVMeta from a Nomad API
// SecureVariableMetadata response.
func NewNomadSVMeta(in *api.SecureVariableMetadata) *NomadSVMeta {
	return &NomadSVMeta{
		Namespace:   in.Namespace,
		Path:        in.Path,
		CreateIndex: in.CreateIndex,
		ModifyIndex: in.ModifyIndex,
		CreateTime:  nanoTime(in.CreateTime),
		ModifyTime:  nanoTime(in.ModifyTime),
	}
}

// NewNomadSecureVariable is used to create a NomadSecureVariable from a Nomad
// API SecureVariable response.
func NewNomadSecureVariable(in *api.SecureVariable) *NomadSecureVariable {
	out := NomadSecureVariable{
		Namespace:   in.Namespace,
		Path:        in.Path,
		CreateIndex: in.CreateIndex,
		ModifyIndex: in.ModifyIndex,
		CreateTime:  nanoTime(in.CreateTime),
		ModifyTime:  nanoTime(in.ModifyTime),
		Items:       map[string]NomadSVItem{},
		sv:          in,
	}

	items := make(NomadSVItems, len(in.Items))
	for k, v := range in.Items {
		items[k] = NomadSVItem{k, v, &out}
	}
	out.Items = items
	return &out
}

// NomadSecureVariable is a template friendly container struct that allows for
// the NomadVar funcs to start inside of Items and have a rational way back up
// to the SecureVariable that is JSON structurally equivalent to the API response.
// This struct's zero value is not trivially usable and should be created with
// NewNomadSecureVariable--especially when outside of the dependency package as
// there is no access to sv.
type NomadSecureVariable struct {
	Namespace, Path          string
	CreateIndex, ModifyIndex uint64
	CreateTime, ModifyTime   nanoTime
	Items                    NomadSVItems
	sv                       *api.SecureVariable
}

func (cv NomadSecureVariable) Metadata() *NomadSVMeta {
	return NewNomadSVMeta(cv.sv.Metadata())
}

type NomadSVItems map[string]NomadSVItem

// Keys returns a sorted list of the Item map's keys.
func (v NomadSVItems) Keys() []string {
	out := make([]string, 0, len(v))
	for k := range v {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Values produces a key-sorted list of the Items map's values
func (v NomadSVItems) Values() []string {
	out := make([]string, 0, len(v))
	for _, k := range v.Keys() {
		out = append(out, v[k].String())
	}
	return out
}

// Tuples produces a key-sorted list of K,V tuple structs from the Items map's
// values
func (v NomadSVItems) Tuples() []struct{ K, V string } {
	out := make([]struct{ K, V string }, 0, len(v))
	for _, k := range v.Keys() {
		out = append(out, struct{ K, V string }{K: k, V: v[k].String()})
	}
	return out
}

// Metadata returns this item's parent's metadata
func (i NomadSVItems) Metadata() *NomadSVMeta {
	for _, v := range i {
		return v.parent.Metadata()
	}
	return nil
}

// Parent returns the item's container object
func (i NomadSVItems) Parent() *NomadSecureVariable {
	for _, v := range i {
		return v.Parent()
	}
	return nil
}

// NomadSVItem enriches the basic string values in a api.SecureVariable's Items
// map with additional helper funcs for formatting and access to it's parent
// item. This enables us to have the template funcs start at the Items
// collection without the user having to delve to it themselves and to minimize
// the number of template funcs that we have to provide for coverage.
type NomadSVItem struct {
	Key, Value string
	parent     *NomadSecureVariable
}

func (v NomadSVItem) String() string               { return v.Value }
func (v NomadSVItem) Metadata() *NomadSVMeta       { return v.parent.Metadata() }
func (v NomadSVItem) Parent() *NomadSecureVariable { return v.parent }
func (v NomadSVItem) MarshalJSON() ([]byte, error) { return []byte(fmt.Sprintf("%q", v.Value)), nil }

// NomadSVMeta provides the same fields as api.SecureVariableMetadata
// but aliases the times into a more template friendly alternative.
type NomadSVMeta struct {
	Namespace, Path          string
	CreateIndex, ModifyIndex uint64
	CreateTime, ModifyTime   nanoTime
}

// nanoTime is the typical storage encoding for times in Nomad's backend. They
// are not pretty for consul-template consumption, so this gives us a type to
// add receivers on.
type nanoTime int64

func (t nanoTime) String() string  { return fmt.Sprintf("%v", time.Unix(0, int64(t))) }
func (t nanoTime) Time() time.Time { return time.Unix(0, int64(t)) }
