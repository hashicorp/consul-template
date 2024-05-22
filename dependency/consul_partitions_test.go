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

func TestNewListPartitionsQuery(t *testing.T) {
	cases := []struct {
		name string
		exp  *ListPartitionsQuery
		err  bool
	}{
		{
			"empty",
			&ListPartitionsQuery{},
			false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			act, err := NewListPartitionsQuery()
			if !tc.err {
				require.NoError(t, err)
			} else {
				assert.Error(t, err)
			}

			if act != nil {
				act.stopCh = nil
			}

			assert.Equal(t, tc.exp, act)
		})
	}
}

func TestListPartitionsQuery_Fetch(t *testing.T) {
	cases := []struct {
		name string
		exp  []Partition
	}{
		{
			"default",
			[]Partition{{Name: "dc1", Description: "dc1"}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d, err := NewListPartitionsQuery()
			require.NoError(t, err)

			act, _, err := d.Fetch(nil, nil)
			require.NoError(t, err)
			assert.Equal(t, tc.exp, act)
		})
	}
}
