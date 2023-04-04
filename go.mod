module github.com/hashicorp/consul-template

go 1.19

require (
	github.com/BurntSushi/toml v1.2.1
	github.com/davecgh/go-spew v1.1.1
	github.com/hashicorp/consul/api v1.18.0
	github.com/hashicorp/consul/sdk v0.13.0
	github.com/hashicorp/go-gatedio v0.5.0
	github.com/hashicorp/go-hclog v1.5.0
	github.com/hashicorp/go-multierror v1.1.1
	github.com/hashicorp/go-rootcerts v1.0.2
	github.com/hashicorp/go-sockaddr v1.0.2
	github.com/hashicorp/go-syslog v1.0.0
	github.com/hashicorp/hcl v1.0.0
	github.com/hashicorp/logutils v1.0.0
	github.com/hashicorp/nomad/api v0.0.0-20230103221135-ce00d683f9be
	github.com/hashicorp/serf v0.10.1 // indirect
	github.com/hashicorp/vault/api v1.9.0
	github.com/imdario/mergo v0.3.13
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mitchellh/hashstructure v1.1.0
	github.com/mitchellh/mapstructure v1.5.0
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.8.2
	golang.org/x/crypto v0.5.0 // indirect
	golang.org/x/sys v0.5.0
	gopkg.in/yaml.v2 v2.4.0
)

require (
	github.com/Masterminds/sprig/v3 v3.2.3
	github.com/hashicorp/vault/api/auth/kubernetes v0.4.0
	golang.org/x/exp v0.0.0-20230321023759-10a507213a29
	golang.org/x/text v0.7.0
)

require (
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver/v3 v3.2.0 // indirect
	github.com/armon/go-metrics v0.4.1 // indirect
	github.com/cenkalti/backoff/v3 v3.2.2 // indirect
	github.com/fatih/color v1.13.0 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/gorilla/websocket v1.5.0 // indirect
	github.com/hashicorp/cronexpr v1.1.1 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-immutable-radix v1.3.1 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.2 // indirect
	github.com/hashicorp/go-secure-stdlib/parseutil v0.1.7 // indirect
	github.com/hashicorp/go-secure-stdlib/strutil v0.1.2 // indirect
	github.com/hashicorp/go-uuid v1.0.3 // indirect
	github.com/hashicorp/go-version v1.6.0 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/huandu/xstrings v1.4.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.17 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/ryanuber/go-glob v1.0.0 // indirect
	github.com/shopspring/decimal v1.3.1 // indirect
	github.com/spf13/cast v1.5.0 // indirect
	golang.org/x/net v0.7.0 // indirect
	golang.org/x/time v0.3.0 // indirect
	gopkg.in/square/go-jose.v2 v2.6.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/golang/lint => golang.org/x/lint v0.0.0-20190409202823-959b441ac422

replace sourcegraph.com/sourcegraph/go-diff => github.com/sourcegraph/go-diff v0.5.1
