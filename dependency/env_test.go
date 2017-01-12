package dependency

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func init() {
	EnvQuerySleepTime = 50 * time.Millisecond
}

func TestNewEnvQuery(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		i    string
		exp  *EnvQuery
		err  bool
	}{
		{
			"empty",
			"",
			nil,
			true,
		},
		{
			"not_empty",
			"hi",
			&EnvQuery{
				key: "hi",
			},
			false,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			act, err := NewEnvQuery(tc.i)
			if (err != nil) != tc.err {
				t.Fatal(err)
			}

			if act != nil {
				act.stopCh = nil
			}

			assert.Equal(t, tc.exp, act)
		})
	}
}

func TestEnvQuery_Fetch(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		exp  string
	}{
		{
			"default",
			"foo",
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			env := fmt.Sprintf("CT_TEST_ENV_%d_%s", i, tc.name)

			d, err := NewEnvQuery(env)
			if err != nil {
				t.Fatal(err)
			}

			os.Setenv(env, "foo")
			defer os.Unsetenv(env)

			act, _, err := d.Fetch(nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, tc.exp, act)
		})
	}

	t.Run("stops", func(t *testing.T) {
		d, err := NewEnvQuery("CT_TEST_ENV")
		if err != nil {
			t.Fatal(err)
		}

		dataCh := make(chan interface{}, 1)
		errCh := make(chan error, 1)
		go func() {
			for {
				data, _, err := d.Fetch(nil, &QueryOptions{WaitIndex: 10})
				if err != nil {
					errCh <- err
					return
				}
				dataCh <- data
			}
		}()

		select {
		case err := <-errCh:
			t.Fatal(err)
		case <-dataCh:
		}

		d.Stop()

		select {
		case err := <-errCh:
			if err != ErrStopped {
				t.Fatal(err)
			}
		case <-time.After(100 * time.Millisecond):
			t.Errorf("did not stop")
		}
	})
}

func TestEnvQuery_String(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		i    string
		exp  string
	}{
		{
			"value",
			"foo",
			"env(foo)",
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			d, err := NewEnvQuery(tc.i)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, tc.exp, d.String())
		})
	}
}
