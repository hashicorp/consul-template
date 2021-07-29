package dependency

import (
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
)

func TestConnectCAQuery_Fetch(t *testing.T) {

	d := NewConnectCAQuery()
	raw, _, err := d.Fetch(testClients, nil)
	assert.NoError(t, err)
	act := raw.([]*api.CARoot)
	if assert.Len(t, act, 1) {
		root := act[0]
		assert.Equal(t, root.Name, "Consul CA Root Cert")
		assert.True(t, root.Active)
		assert.NotEmpty(t, root.RootCertPEM)
	}
}
