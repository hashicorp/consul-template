package dependency

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVaultAgentTokenQuery_String(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		exp  string
	}{
		{
			"default",
			"vault-agent.token",
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			d, err := NewVaultAgentTokenQuery("/tmp/token")
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, tc.exp, d.String())
		})
	}
}
