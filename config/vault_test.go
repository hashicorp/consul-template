package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
	"time"
)

func TestVaultConfig_Copy(t *testing.T) {
	cases := []struct {
		name string
		a    *VaultConfig
	}{
		{
			"nil",
			nil,
		},
		{
			"empty",
			&VaultConfig{},
		},
		{
			"same_enabled",
			&VaultConfig{
				Address:    String("address"),
				Enabled:    Bool(true),
				Grace:      TimeDuration(1 * time.Minute),
				RenewToken: Bool(true),
				Retry:      &RetryConfig{Enabled: Bool(true)},
				SSL:        &SSLConfig{Enabled: Bool(true)},
				Token:      String("token"),
				Transport: &TransportConfig{
					DialKeepAlive: TimeDuration(20 * time.Second),
				},
				UnwrapToken: Bool(true),
			},
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			r := tc.a.Copy()
			if !reflect.DeepEqual(tc.a, r) {
				t.Errorf("\nexp: %#v\nact: %#v", tc.a, r)
			}
		})
	}
}

func TestVaultConfig_Merge(t *testing.T) {
	cases := []struct {
		name string
		a    *VaultConfig
		b    *VaultConfig
		r    *VaultConfig
	}{
		{
			"nil_a",
			nil,
			&VaultConfig{},
			&VaultConfig{},
		},
		{
			"nil_b",
			&VaultConfig{},
			nil,
			&VaultConfig{},
		},
		{
			"nil_both",
			nil,
			nil,
			nil,
		},
		{
			"empty",
			&VaultConfig{},
			&VaultConfig{},
			&VaultConfig{},
		},
		{
			"enabled_overrides",
			&VaultConfig{Enabled: Bool(true)},
			&VaultConfig{Enabled: Bool(false)},
			&VaultConfig{Enabled: Bool(false)},
		},
		{
			"enabled_empty_one",
			&VaultConfig{Enabled: Bool(true)},
			&VaultConfig{},
			&VaultConfig{Enabled: Bool(true)},
		},
		{
			"enabled_empty_two",
			&VaultConfig{},
			&VaultConfig{Enabled: Bool(true)},
			&VaultConfig{Enabled: Bool(true)},
		},
		{
			"enabled_same",
			&VaultConfig{Enabled: Bool(true)},
			&VaultConfig{Enabled: Bool(true)},
			&VaultConfig{Enabled: Bool(true)},
		},
		{
			"address_overrides",
			&VaultConfig{Address: String("address")},
			&VaultConfig{Address: String("")},
			&VaultConfig{Address: String("")},
		},
		{
			"address_empty_one",
			&VaultConfig{Address: String("address")},
			&VaultConfig{},
			&VaultConfig{Address: String("address")},
		},
		{
			"address_empty_two",
			&VaultConfig{},
			&VaultConfig{Address: String("address")},
			&VaultConfig{Address: String("address")},
		},
		{
			"address_same",
			&VaultConfig{Address: String("address")},
			&VaultConfig{Address: String("address")},
			&VaultConfig{Address: String("address")},
		},
		{
			"grace_overrides",
			&VaultConfig{Grace: TimeDuration(5 * time.Minute)},
			&VaultConfig{Grace: TimeDuration(10 * time.Minute)},
			&VaultConfig{Grace: TimeDuration(10 * time.Minute)},
		},
		{
			"grace_empty_one",
			&VaultConfig{Grace: TimeDuration(5 * time.Minute)},
			&VaultConfig{},
			&VaultConfig{Grace: TimeDuration(5 * time.Minute)},
		},
		{
			"grace_empty_two",
			&VaultConfig{},
			&VaultConfig{Grace: TimeDuration(5 * time.Minute)},
			&VaultConfig{Grace: TimeDuration(5 * time.Minute)},
		},
		{
			"grace_same",
			&VaultConfig{Grace: TimeDuration(5 * time.Minute)},
			&VaultConfig{Grace: TimeDuration(5 * time.Minute)},
			&VaultConfig{Grace: TimeDuration(5 * time.Minute)},
		},
		{
			"token_overrides",
			&VaultConfig{Token: String("token")},
			&VaultConfig{Token: String("")},
			&VaultConfig{Token: String("")},
		},
		{
			"token_empty_one",
			&VaultConfig{Token: String("token")},
			&VaultConfig{},
			&VaultConfig{Token: String("token")},
		},
		{
			"token_empty_two",
			&VaultConfig{},
			&VaultConfig{Token: String("token")},
			&VaultConfig{Token: String("token")},
		},
		{
			"token_same",
			&VaultConfig{Token: String("token")},
			&VaultConfig{Token: String("token")},
			&VaultConfig{Token: String("token")},
		},
		{
			"unwrap_token_overrides",
			&VaultConfig{UnwrapToken: Bool(true)},
			&VaultConfig{UnwrapToken: Bool(false)},
			&VaultConfig{UnwrapToken: Bool(false)},
		},
		{
			"unwrap_token_empty_one",
			&VaultConfig{UnwrapToken: Bool(true)},
			&VaultConfig{},
			&VaultConfig{UnwrapToken: Bool(true)},
		},
		{
			"unwrap_token_empty_two",
			&VaultConfig{},
			&VaultConfig{UnwrapToken: Bool(true)},
			&VaultConfig{UnwrapToken: Bool(true)},
		},
		{
			"unwrap_token_same",
			&VaultConfig{UnwrapToken: Bool(true)},
			&VaultConfig{UnwrapToken: Bool(true)},
			&VaultConfig{UnwrapToken: Bool(true)},
		},
		{
			"renew_token_overrides",
			&VaultConfig{RenewToken: Bool(true)},
			&VaultConfig{RenewToken: Bool(false)},
			&VaultConfig{RenewToken: Bool(false)},
		},
		{
			"renew_token_empty_one",
			&VaultConfig{RenewToken: Bool(true)},
			&VaultConfig{},
			&VaultConfig{RenewToken: Bool(true)},
		},
		{
			"renew_token_empty_two",
			&VaultConfig{},
			&VaultConfig{RenewToken: Bool(true)},
			&VaultConfig{RenewToken: Bool(true)},
		},
		{
			"renew_token_same",
			&VaultConfig{RenewToken: Bool(true)},
			&VaultConfig{RenewToken: Bool(true)},
			&VaultConfig{RenewToken: Bool(true)},
		},
		{
			"retry_overrides",
			&VaultConfig{Retry: &RetryConfig{Enabled: Bool(true)}},
			&VaultConfig{Retry: &RetryConfig{Enabled: Bool(false)}},
			&VaultConfig{Retry: &RetryConfig{Enabled: Bool(false)}},
		},
		{
			"retry_empty_one",
			&VaultConfig{Retry: &RetryConfig{Enabled: Bool(true)}},
			&VaultConfig{},
			&VaultConfig{Retry: &RetryConfig{Enabled: Bool(true)}},
		},
		{
			"retry_empty_two",
			&VaultConfig{},
			&VaultConfig{Retry: &RetryConfig{Enabled: Bool(true)}},
			&VaultConfig{Retry: &RetryConfig{Enabled: Bool(true)}},
		},
		{
			"retry_same",
			&VaultConfig{Retry: &RetryConfig{Enabled: Bool(true)}},
			&VaultConfig{Retry: &RetryConfig{Enabled: Bool(true)}},
			&VaultConfig{Retry: &RetryConfig{Enabled: Bool(true)}},
		},
		{
			"ssl_overrides",
			&VaultConfig{SSL: &SSLConfig{Enabled: Bool(true)}},
			&VaultConfig{SSL: &SSLConfig{Enabled: Bool(false)}},
			&VaultConfig{SSL: &SSLConfig{Enabled: Bool(false)}},
		},
		{
			"ssl_empty_one",
			&VaultConfig{SSL: &SSLConfig{Enabled: Bool(true)}},
			&VaultConfig{},
			&VaultConfig{SSL: &SSLConfig{Enabled: Bool(true)}},
		},
		{
			"ssl_empty_two",
			&VaultConfig{},
			&VaultConfig{SSL: &SSLConfig{Enabled: Bool(true)}},
			&VaultConfig{SSL: &SSLConfig{Enabled: Bool(true)}},
		},
		{
			"ssl_same",
			&VaultConfig{SSL: &SSLConfig{Enabled: Bool(true)}},
			&VaultConfig{SSL: &SSLConfig{Enabled: Bool(true)}},
			&VaultConfig{SSL: &SSLConfig{Enabled: Bool(true)}},
		},
		{
			"transport_overrides",
			&VaultConfig{Transport: &TransportConfig{DialKeepAlive: TimeDuration(10 * time.Second)}},
			&VaultConfig{Transport: &TransportConfig{DialKeepAlive: TimeDuration(20 * time.Second)}},
			&VaultConfig{Transport: &TransportConfig{DialKeepAlive: TimeDuration(20 * time.Second)}},
		},
		{
			"transport_empty_one",
			&VaultConfig{Transport: &TransportConfig{DialKeepAlive: TimeDuration(10 * time.Second)}},
			&VaultConfig{},
			&VaultConfig{Transport: &TransportConfig{DialKeepAlive: TimeDuration(10 * time.Second)}},
		},
		{
			"transport_empty_two",
			&VaultConfig{},
			&VaultConfig{Transport: &TransportConfig{DialKeepAlive: TimeDuration(10 * time.Second)}},
			&VaultConfig{Transport: &TransportConfig{DialKeepAlive: TimeDuration(10 * time.Second)}},
		},
		{
			"transport_same",
			&VaultConfig{Transport: &TransportConfig{DialKeepAlive: TimeDuration(10 * time.Second)}},
			&VaultConfig{Transport: &TransportConfig{DialKeepAlive: TimeDuration(10 * time.Second)}},
			&VaultConfig{Transport: &TransportConfig{DialKeepAlive: TimeDuration(10 * time.Second)}},
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			r := tc.a.Merge(tc.b)
			if !reflect.DeepEqual(tc.r, r) {
				t.Errorf("\nexp: %#v\nact: %#v", tc.r, r)
			}
		})
	}
}

func TestVaultConfig_Finalize(t *testing.T) {
	cases := []struct {
		name string
		i    *VaultConfig
		r    *VaultConfig
	}{
		{
			"empty",
			&VaultConfig{},
			&VaultConfig{
				Address:    String(""),
				Enabled:    Bool(false),
				Grace:      TimeDuration(DefaultVaultGrace),
				RenewToken: Bool(DefaultVaultRenewToken),
				Retry: &RetryConfig{
					Backoff:    TimeDuration(DefaultRetryBackoff),
					MaxBackoff: TimeDuration(DefaultRetryMaxBackoff),
					Enabled:    Bool(true),
					Attempts:   Int(DefaultRetryAttempts),
				},
				SSL: &SSLConfig{
					CaCert:     String(""),
					CaPath:     String(""),
					Cert:       String(""),
					Enabled:    Bool(true),
					Key:        String(""),
					ServerName: String(""),
					Verify:     Bool(true),
				},
				Token: String(""),
				Transport: &TransportConfig{
					DialKeepAlive:       TimeDuration(DefaultDialKeepAlive),
					DialTimeout:         TimeDuration(DefaultDialTimeout),
					DisableKeepAlives:   Bool(false),
					IdleConnTimeout:     TimeDuration(DefaultIdleConnTimeout),
					MaxIdleConns:        Int(DefaultMaxIdleConns),
					MaxIdleConnsPerHost: Int(DefaultMaxIdleConnsPerHost),
					TLSHandshakeTimeout: TimeDuration(DefaultTLSHandshakeTimeout),
				},
				UnwrapToken: Bool(DefaultVaultUnwrapToken),
			},
		},
		{
			"with_address",
			&VaultConfig{
				Address: String("address"),
			},
			&VaultConfig{
				Address:    String("address"),
				Enabled:    Bool(true),
				Grace:      TimeDuration(DefaultVaultGrace),
				RenewToken: Bool(DefaultVaultRenewToken),
				Retry: &RetryConfig{
					Backoff:    TimeDuration(DefaultRetryBackoff),
					MaxBackoff: TimeDuration(DefaultRetryMaxBackoff),
					Enabled:    Bool(true),
					Attempts:   Int(DefaultRetryAttempts),
				},
				SSL: &SSLConfig{
					CaCert:     String(""),
					CaPath:     String(""),
					Cert:       String(""),
					Enabled:    Bool(true),
					Key:        String(""),
					ServerName: String(""),
					Verify:     Bool(true),
				},
				Token: String(""),
				Transport: &TransportConfig{
					DialKeepAlive:       TimeDuration(DefaultDialKeepAlive),
					DialTimeout:         TimeDuration(DefaultDialTimeout),
					DisableKeepAlives:   Bool(false),
					IdleConnTimeout:     TimeDuration(DefaultIdleConnTimeout),
					MaxIdleConns:        Int(DefaultMaxIdleConns),
					MaxIdleConnsPerHost: Int(DefaultMaxIdleConnsPerHost),
					TLSHandshakeTimeout: TimeDuration(DefaultTLSHandshakeTimeout),
				},
				UnwrapToken: Bool(DefaultVaultUnwrapToken),
			},
		},
		{
			"with_ssl_config",
			&VaultConfig{
				Address: String("address"),
			},
			&VaultConfig{
				Address:    String("address"),
				Enabled:    Bool(true),
				Grace:      TimeDuration(DefaultVaultGrace),
				RenewToken: Bool(DefaultVaultRenewToken),
				Retry: &RetryConfig{
					Backoff:    TimeDuration(DefaultRetryBackoff),
					MaxBackoff: TimeDuration(DefaultRetryMaxBackoff),
					Enabled:    Bool(true),
					Attempts:   Int(DefaultRetryAttempts),
				},
				SSL: &SSLConfig{
					CaCert:     String(""),
					CaPath:     String(""),
					Cert:       String(""),
					Enabled:    Bool(true),
					Key:        String(""),
					ServerName: String(""),
					Verify:     Bool(true),
				},
				Token: String(""),
				Transport: &TransportConfig{
					DialKeepAlive:       TimeDuration(DefaultDialKeepAlive),
					DialTimeout:         TimeDuration(DefaultDialTimeout),
					DisableKeepAlives:   Bool(false),
					IdleConnTimeout:     TimeDuration(DefaultIdleConnTimeout),
					MaxIdleConns:        Int(DefaultMaxIdleConns),
					MaxIdleConnsPerHost: Int(DefaultMaxIdleConnsPerHost),
					TLSHandshakeTimeout: TimeDuration(DefaultTLSHandshakeTimeout),
				},
				UnwrapToken: Bool(DefaultVaultUnwrapToken),
			},
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			os.Unsetenv("VAULT_ADDR")
			os.Unsetenv("VAULT_TOKEN")
			os.Unsetenv("VAULT_DEV_ROOT_TOKEN_ID")
			homePath, _ = ioutil.TempDir("", "")

			tc.i.Finalize()
			if !reflect.DeepEqual(tc.r, tc.i) {
				t.Errorf("\nexp: %#v\nact: %#v", tc.r, tc.i)
			}
		})
	}
}
