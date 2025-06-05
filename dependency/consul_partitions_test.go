// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package dependency

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	ListPartitionsQuerySleepTime = 50 * time.Millisecond
}

func TestListPartitionsQuery_Fetch(t *testing.T) {
	if !tenancyHelper.IsConsulEnterprise() {
		t.Skip("Enterprise only test")
	}

	expected := []*Partition{
		{
			Name:        "default",
			Description: "Builtin Default Partition",
		},
		{
			Name:        "foo",
			Description: "",
		},
	}

	d, err := NewListPartitionsQuery()
	require.NoError(t, err)

	act, _, err := d.Fetch(testClients, nil)
	require.NoError(t, err)
	assert.Equal(t, expected, act)
}

func TestListPartitionsQuery_FetchError(t *testing.T) {
	if tenancyHelper.IsConsulEnterprise() {
		t.Skip("CE only test")
	}

	d, err := NewListPartitionsQuery()
	require.NoError(t, err)

	_, _, err = d.Fetch(testClients, nil)
	require.Error(t, err)
}
