module github.com/hashicorp/consul-template

go 1.12

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/frankban/quicktest v1.4.0 // indirect
	github.com/google/btree v1.0.0 // indirect
	github.com/hashicorp/consul/api v1.4.0
    github.com/hashicorp/consul/sdk v0.4.1-0.20200910203702-bb2b5dd871ca
	github.com/hashicorp/go-gatedio v0.5.0
	github.com/hashicorp/go-msgpack v0.5.5 // indirect
	github.com/hashicorp/go-multierror v1.1.0
	github.com/hashicorp/go-rootcerts v1.0.2
	github.com/hashicorp/go-sockaddr v1.0.2
	github.com/hashicorp/go-syslog v1.0.0
	github.com/hashicorp/hcat v0.0.0-20200723012536-d743bddb1b6b
	github.com/hashicorp/hcl v1.0.0
	github.com/hashicorp/logutils v1.0.0
	github.com/hashicorp/vault/api v1.0.5-0.20190730042357-746c0b111519
	github.com/mattn/go-shellwords v1.0.5
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mitchellh/hashstructure v1.0.0
	github.com/mitchellh/mapstructure v1.3.0
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.5.1
	golang.org/x/text v0.3.2 // indirect
	gopkg.in/yaml.v2 v2.2.8
)

replace github.com/golang/lint => golang.org/x/lint v0.0.0-20190409202823-959b441ac422

replace sourcegraph.com/sourcegraph/go-diff => github.com/sourcegraph/go-diff v0.5.1
