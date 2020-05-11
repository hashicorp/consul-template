# Consul Template

[![CircleCI](https://circleci.com/gh/hashicorp/consul-template.svg?style=svg)](https://circleci.com/gh/hashicorp/consul-template)
[![Go Documentation](http://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)](https://godoc.org/github.com/hashicorp/consul-template)

This project provides a convenient way to populate values from [Consul][consul]
into the file system using the `consul-template` daemon.

The daemon `consul-template` queries a [Consul][consul] or [Vault][vault]
cluster and updates any number of specified templates on the file system. As an
added bonus, it can optionally run arbitrary commands when the update process
completes. Please see the [examples folder][examples] for some scenarios where
this functionality might prove useful.

---

**The documentation in this README corresponds to the master branch of Consul Template. It may contain unreleased features or different APIs than the most recently released version.**

**Please see the [Git tag](https://github.com/hashicorp/consul-template/releases) that corresponds to your version of Consul Template for the proper documentation.**

---

## Table of Contents

- [Community Support](#community-support)
- [Installation](#installation)
- [Quick Example](#quick-example)
- [Usage](#usage)
- [Configuration](docs/config.md)
  - [Command Line Flags](docs/config.md#command-line-flags)
  - [Configuration File](docs/config.md#configuration-file)
- [Templating Language](#templating-language)
  - [API Functions](#api-functions)
    - [caLeaf](#caleaf)
    - [caRoot](#caroot)
    - [connect](#connect)
    - [datacenters](#datacenters)
    - [file](#file)
    - [key](#key)
    - [keyExists](#keyexists)
    - [keyOrDefault](#keyordefault)
    - [ls](#ls)
    - [safeLs](#safels)
    - [node](#node)
    - [nodes](#nodes)
    - [secret](#secret)
    - [secrets](#secrets)
    - [service](#service)
    - [services](#services)
    - [tree](#tree)
    - [safeTree](#safetree)
  - [Scratch](#scratch)
    - [scratch.Key](#scratchkey)
    - [scratch.Get](#scratchget)
    - [scratch.Set](#scratchset)
    - [scratch.SetX](#scratchsetx)
    - [scratch.MapSet](#scratchmapset)
    - [scratch.MapSetX](#scratchmapsetx)
    - [scratch.MapValues](#scratchmapvalues)
  - [Helper Functions](#helper-functions)
    - [base64Decode](#base64decode)
    - [base64Encode](#base64encode)
    - [base64URLDecode](#base64urldecode)
    - [base64URLEncode](#base64urlencode)
    - [byKey](#bykey)
    - [byTag](#bytag)
    - [byMeta](#bymeta)
    - [contains](#contains)
    - [containsAll](#containsall)
    - [containsAny](#containsany)
    - [containsNone](#containsnone)
    - [containsNotAll](#containsnotall)
    - [env](#env)
    - [executeTemplate](#executetemplate)
    - [explode](#explode)
    - [explodeMap](#explodemap)
    - [indent](#indent)
    - [in](#in)
    - [loop](#loop)
    - [join](#join)
    - [trimSpace](#trimspace)
    - [parseBool](#parsebool)
    - [parseFloat](#parsefloat)
    - [parseInt](#parseint)
    - [parseJSON](#parsejson)
    - [parseUint](#parseuint)
    - [parseYAML](#parseyaml)
    - [plugin](#plugin)
    - [regexMatch](#regexmatch)
    - [regexReplaceAll](#regexreplaceall)
    - [replaceAll](#replaceall)
    - [sha256Hex](#sha256hex)
    - [split](#split)
    - [timestamp](#timestamp)
    - [toJSON](#tojson)
    - [toJSONPretty](#tojsonpretty)
    - [toLower](#tolower)
    - [toTitle](#totitle)
    - [toTOML](#totoml)
    - [toUpper](#toupper)
    - [toYAML](#toyaml)
    - [sockaddr](#sockaddr)
  - [Math Functions](#math-functions)
    - [add](#add)
    - [subtract](#subtract)
    - [multiply](#multiply)
    - [divide](#divide)
    - [modulo](#modulo)
    - [minimum](#minimum)
    - [maximum](#maximum)
- [Plugins](#plugins)
  - [Authoring Plugins](#authoring-plugins)
    - [Important Notes](#important-notes)
- [Caveats](#caveats)
  - [Docker Image Use](#docker-image-use)
  - [Dots in Service Names](#dots-in-service-names)
  - [Once Mode](#once-mode)
  - [Exec Mode](#exec-mode)
  - [De-Duplication Mode](#de-duplication-mode)
  - [Termination on Error](#termination-on-error)
  - [Commands](#commands)
    - [Environment](#environment)
    - [Multiple Commands](#multiple-commands)
  - [Multi-phase Execution](#multi-phase-execution)
- [Running and Process Lifecycle](#running-and-process-lifecycle)
- [Debugging](#debugging)
- [Telemetry](#telemetry)
- [FAQ](#faq)
- [Contributing](#contributing)


## Community Support

If you have questions about how consul-template works, its capabilities or
anything other than a bug or feature request (use github's issue tracker for
those), please see our community support resources.

Community portal: https://discuss.hashicorp.com/c/consul

Other resources: https://www.consul.io/community.html

Additionally, for issues and pull requests, we'll be using the :+1: reactions
as a rough voting system to help gauge community priorities. So please add :+1:
to any issue or pull request you'd like to see worked on. Thanks.


## Installation

1. Download a pre-compiled, released version from the [Consul Template releases page][releases].

1. Extract the binary using `unzip` or `tar`.

1. Move the binary into `$PATH`.

To compile from source, please see the instructions in the
[contributing section](#contributing).

## Quick Example

This short example assumes Consul is installed locally.

1. Start a Consul cluster in dev mode:

    ```shell
    $ consul agent -dev
    ```

1. Author a template `in.tpl` to query the kv store:

    ```liquid
    {{ key "foo" }}
    ```

1. Start Consul Template:

    ```shell
    $ consul-template -template "in.tpl:out.txt" -once
    ```

1. Write data to the key in Consul:

    ```shell
    $ consul kv put foo bar
    ```

1. Observe Consul Template has written the file `out.txt`:

    ```shell
    $ cat out.txt
    bar
    ```

For more examples and use cases, please see the [examples folder][examples] in
this repository.

### Templating Language

Consul Template parses files authored in the [Go Template][text-template]
format. If you are not familiar with the syntax, please read Go's documentation
and examples. In addition to the Go-provided template functions, Consul Template
provides the following functions:

#### API Functions

API functions interact with remote API calls, communicating with external
services like [Consul][consul] and [Vault][vault].

##### `caLeaf`

Query [Consul][consul] for the leaf certificate representing a single service.

```liquid
{{ caLeaf "<NAME>" }}
```

For example:
```liquid
{{ with caLeaf "proxy" }}{{ .CertPEM }}{{ end }}
```

renders
```text
-----BEGIN CERTIFICATE-----
MIICizCCAjGgAwIBAgIBCDAKBggqhkjOPQQDAjAWMRQwEgYDVQQDEwtDb25zdWwg
...
lXcQzfKlIYeFWvcAv4cA4W258gTtqaFRDRJ2i720eQ==
-----END CERTIFICATE-----
```

The two most useful fields are `.CertPEM` and `.PrivateKeyPEM`. For a complete
list of available fields, see consul's documentation on
[LeafCert](https://godoc.org/github.com/hashicorp/consul/api#LeafCert).

##### `caRoot`

Query [Consul][consul] for all [connect][connect] trusted certificate authority
(CA) root certificates.

```liquid
{{ caRoots }}
```

For example:
```liquid
{{ range caRoots }}{{ .RootCertPEM }}{{ end }}
```

renders
```text
-----BEGIN CERTIFICATE-----
MIICWDCCAf+gAwIBAgIBBzAKBggqhkjOPQQDAjAWMRQwEgYDVQQDEwtDb25zdWwg
...
bcA+Su3r8qSRppTlc6D0UOYOWc1ykQKQOK7mIg==
-----END CERTIFICATE-----

```

The most useful field is `.RootCertPEM`. For a complete list of available
fields, see consul's documentation on
[CARootList](https://godoc.org/github.com/hashicorp/consul/api#CARootList).


##### `connect`

Query [Consul][consul] for [connect][connect]-capable services based on their
health.

```liquid
{{ connect "<TAG>.<NAME>@<DATACENTER>~<NEAR>|<FILTER>" }}
```

Syntax is exactly the same as for the [service](#service) function below.


```liquid
{{ range connect "web" }}
server {{ .Name }} {{ .Address }}:{{ .Port }}{{ end }}
```

renders the IP addresses of all _healthy_ nodes with a logical
[connect][connect]-capable service named "web":

```text
server web01 10.5.2.45:21000
server web02 10.2.6.61:21000
```


##### `datacenters`

Query [Consul][consul] for all datacenters in its catalog.

```liquid
{{ datacenters }}
```

For example:

```liquid
{{ range datacenters }}
{{ . }}{{ end }}
```

renders

```text
dc1
dc2
```

An optional boolean can be specified which instructs Consul Template to ignore
datacenters which are inaccessible or do not have a current leader. Enabling
this option requires an O(N+1) operation and therefore is not recommended in
environments where performance is a factor.

```liquid
// Ignores datacenters which are inaccessible
{{ datacenters true }}
```

##### `file`

Read and output the contents of a local file on disk. If the file cannot be
read, an error will occur. When the file changes, Consul Template will pick up
the change and re-render the template.

```liquid
{{ file "<PATH>" }}
```

For example:

```liquid
{{ file "/path/to/my/file" }}
```

renders

```text
file contents
```

This does not process nested templates. See
[`executeTemplate`](#executeTemplate) for a way to render nested templates.

##### `key`

Query [Consul][consul] for the value at the given key path. If the key does not
exist, Consul Template will block rendering until the key is present. To avoid
blocking, use `keyOrDefault` or `keyExists`.

```liquid
{{ key "<PATH>@<DATACENTER>" }}
```

The `<DATACENTER>` attribute is optional; if omitted, the local datacenter is
used.

For example:

```liquid
{{ key "service/redis/maxconns" }}
```

renders

```text
15
```

##### `keyExists`

Query [Consul][consul] for the value at the given key path. If the key exists,
this will return true, false otherwise. Unlike `key`, this function will not
block if the key does not exist. This is useful for controlling flow.

```liquid
{{ keyExists "<PATH>@<DATACENTER>" }}
```

The `<DATACENTER>` attribute is optional; if omitted, the local datacenter is
used.

For example:

```liquid
{{ if keyExists "app/beta_active" }}
  # ...
{{ else }}
  # ...
{{ end }}
```

##### `keyOrDefault`

Query [Consul][consul] for the value at the given key path. If the key does not
exist, the default value will be used instead. Unlike `key`, this function will
not block if the key does not exist.

```liquid
{{ keyOrDefault "<PATH>@<DATACENTER>" "<DEFAULT>" }}
```

The `<DATACENTER>` attribute is optional; if omitted, the local datacenter is
used.

For example:

```liquid
{{ keyOrDefault "service/redis/maxconns" "5" }}
```

renders

```text
5
```

Note that Consul Template uses a [multi-phase
execution](#multi-phase-execution). During the first phase of evaluation, Consul
Template will have no data from Consul and thus will _always_ fall back to the
default value. Subsequent reads from Consul will pull in the real value from
Consul (if the key exists) on the next template pass. This is important because
it means that Consul Template will never "block" the rendering of a template due
to a missing key from a `keyOrDefault`. Even if the key exists, if Consul has
not yet returned data for the key, the default value will be used instead.

##### `ls`

Query [Consul][consul] for all top-level kv pairs at the given key path.

```liquid
{{ ls "<PATH>@<DATACENTER>" }}
```

The `<DATACENTER>` attribute is optional; if omitted, the local datacenter is
used.

For example:

```liquid
{{ range ls "service/redis" }}
{{ .Key }}:{{ .Value }}{{ end }}
```

renders

```text
maxconns:15
minconns:5
```

##### `safeLs`

Same as [ls](#ls), but refuse to render template, if the KV prefix query return blank/empty data.

This is especially useful, for rendering mission critical files, that are being populated by consul-template.

For example:

```text
/root/.ssh/authorized_keys
/etc/sysconfig/iptables
```

Using `safeLs` on empty prefixes will result in template output not being rendered at all.

To learn how `safeLs` was born see [CT-1131](https://github.com/hashicorp/consul-template/issues/1131) [C-3975](https://github.com/hashicorp/consul/issues/3975) and [CR-82](https://github.com/hashicorp/consul-replicate/issues/82).

##### `node`

Query [Consul][consul] for a node in the catalog.

```liquid
{{node "<NAME>@<DATACENTER>"}}
```

The `<NAME>` attribute is optional; if omitted, the local agent node is used.

The `<DATACENTER>` attribute is optional; if omitted, the local datacenter is
used.

For example:

```liquid
{{ with node }}
{{ .Node.Address }}{{ end }}
```

renders

```text
10.5.2.6
```

To query a different node:

```liquid
{{ with node "node1@dc2" }}
{{ .Node.Address }}{{ end }}
```

renders

```text
10.4.2.6
```

To access map data such as `TaggedAddresses` or `Meta`, use
[Go's text/template][text-template] map indexing.

##### `nodes`

Query [Consul][consul] for all nodes in the catalog.

```liquid
{{ nodes "@<DATACENTER>~<NEAR>" }}
```

The `<DATACENTER>` attribute is optional; if omitted, the local datacenter is
used.

The `<NEAR>` attribute is optional; if omitted, results are specified in lexical
order. If provided a node name, results are ordered by shortest round-trip time
to the provided node. If provided `_agent`, results are ordered by shortest
round-trip time to the local agent.

For example:

```liquid
{{ range nodes }}
{{ .Address }}{{ end }}
```

renders

```text
10.4.2.13
10.46.2.5
```

To query a different data center and order by shortest trip time to ourselves:

```liquid
{{ range nodes "@dc2~_agent" }}
{{ .Address }}{{ end }}
```

To access map data such as `TaggedAddresses` or `Meta`, use
[Go's text/template][text-template] map indexing.

##### `secret`

Query [Vault][vault] for the secret at the given path.

```liquid
{{ secret "<PATH>" "<DATA>" }}
```

The `<DATA>` attribute is optional; if omitted, the request will be a `vault
read` (HTTP GET) request. If provided, the request will be a `vault write` (HTTP
PUT/POST) request.

For example:

```liquid
{{ with secret "secret/passwords" }}
{{ .Data.wifi }}{{ end }}
```

renders

```text
FORWARDSoneword
```

To access a versioned secret value (for the K/V version 2 backend):

```liquid
{{ with secret "secret/passwords?version=1" }}
{{ .Data.data.wifi }}{{ end }}
```

When omitting the `?version` parameter, the latest version of the secret will be
fetched. Note the nested `.Data.data` syntax when referencing the secret value.
For more information about using the K/V v2 backend, see the
[Vault Documentation](https://www.vaultproject.io/docs/secrets/kv/kv-v2.html).

When using Vault versions 0.10.0/0.10.1, the secret path will have to be prefixed
with "data", i.e. `secret/data/passwords` for the example above. This is not
necessary for Vault versions after 0.10.1, as consul-template will detect the KV
backend version being used. The version 2 KV backend did not exist prior to 0.10.0,
so these are the only affected versions.

An example using write to generate PKI certificates:

```liquid
{{ with secret "pki/issue/my-domain-dot-com" "common_name=foo.example.com" }}
{{ .Data.certificate }}{{ end }}
```

The parameters must be `key=value` pairs, and each pair must be its own argument
to the function:

Please always consider the security implications of having the contents of a
secret in plain-text on disk. If an attacker is able to get access to the file,
they will have access to plain-text secrets.

Please note that Vault does not support blocking queries. As a result, Consul
Template will not immediately reload in the event a secret is changed as it
does with Consul's key-value store. Consul Template will renew the secret with
Vault's [Renewer API](https://godoc.org/github.com/hashicorp/vault/api#Renewer).
The Renew API tries to use most of the time the secret is good, renewing at
around 90% of the lease time (as set by Vault).

Also consider enabling `error_on_missing_key` when working with templates that
will interact with Vault. By default, Consul Template uses Go's templating
language. When accessing a struct field or map key that does not exist, it
defaults to printing `<no value>`. This may not be the desired behavior,
especially when working with passwords or other data. As such, it is recommended
you set:

```hcl
template {
  error_on_missing_key = true
}
```

You can also guard against empty values using `if` or `with` blocks.

```liquid
{{ with secret "secret/foo"}}
{{ if .Data.password }}
password = "{{ .Data.password }}"
{{ end }}
{{ end }}
```

##### `secrets`

Query [Vault][vault] for the list of secrets at the given path. Not all
endpoints support listing.

```liquid
{{ secrets "<PATH>" }}
```

For example:

```liquid
{{ range secrets "secret/" }}
{{ . }}{{ end }}
```

renders

```text
bar
foo
zip
```

To iterate and list over every secret in the generic secret backend in Vault:

```liquid
{{ range secrets "secret/" }}
{{ with secret (printf "secret/%s" .) }}{{ range $k, $v := .Data }}
{{ $k }}: {{ $v }}
{{ end }}{{ end }}{{ end }}
```

You should probably never do this.

Please also note that Vault does not support
blocking queries. To understand the implications, please read the note at the
end of the `secret` function.

##### `service`

Query [Consul][consul] for services based on their health.

```liquid
{{ service "<TAG>.<NAME>@<DATACENTER>~<NEAR>|<FILTER>" }}
```

The `<TAG>` attribute is optional; if omitted, all nodes will be queried.

The `<DATACENTER>` attribute is optional; if omitted, the local datacenter is
used.

The `<NEAR>` attribute is optional; if omitted, results are specified in lexical
order. If provided a node name, results are ordered by shortest round-trip time
to the provided node. If provided `_agent`, results are ordered by shortest
round-trip time to the local agent.

The `<FILTER>` attribute is optional; if omitted, only health services are
returned. Providing a filter allows for client-side filtering of services.

For example:

The example above is querying Consul for healthy "web" services, in the "east-aws" data center. The tag and data center attributes are optional. To query all nodes of the "web" service (regardless of tag) for the current data center:

```liquid
{{ range service "web" }}
server {{ .Name }} {{ .Address }}:{{ .Port }}{{ end }}
```

renders the IP addresses of all _healthy_ nodes with a logical service named
"web":

```text
server web01 10.5.2.45:2492
server web02 10.2.6.61:2904
```

To access map data such as `NodeTaggedAddresses` or `NodeMeta`, use
[Go's text/template][text-template] map indexing.

By default only healthy services are returned. To list all services, pass the
"any" filter:

```liquid
{{ service "web|any" }}
```

This will return all services registered to the agent, regardless of their
status.

To filter services by a specific set of healths, specify a comma-separated list
of health statuses:

```liquid
{{ service "web|passing,warning" }}
```

This will returns services which are deemed "passing" or "warning" according to
their node and service-level checks defined in Consul. Please note that the
comma implies an "or", not an "and".

**Note:** Due to the use of dot `.` to delimit TAG, the `service` command will
not recognize service names containing dots.

**Note:** There is an architectural difference between the following:

```liquid
{{ service "web" }}
{{ service "web|passing" }}
```

The former will return all services which Consul considers "healthy" and
passing. The latter will return all services registered with the Consul agent
and perform client-side filtering. As a general rule, do not use the "passing"
argument alone if you want only healthy services - simply omit the second
argument instead.


##### `services`

Query [Consul][consul] for all services in the catalog.

```liquid
{{ services "@<DATACENTER>" }}
```

The `<DATACENTER>` attribute is optional; if omitted, the local datacenter is
used.

For example:

```liquid
{{ range services }}
{{ .Name }}: {{ .Tags | join "," }}{{ end }}
```

renders

```text
node01 tag1,tag2,tag3
```

##### `tree`

Query [Consul][consul] for all kv pairs at the given key path.

```liquid
{{ tree "<PATH>@<DATACENTER>" }}
```

The `<DATACENTER>` attribute is optional; if omitted, the local datacenter is
used.

For example:

```liquid
{{ range tree "service/redis" }}
{{ .Key }}:{{ .Value }}{{ end }}
```
renders

```text
minconns 2
maxconns 12
nested/config/value "value"
```

Unlike `ls`, `tree` returns **all** keys under the prefix, just like the Unix
`tree` command.

##### `safeTree`

Same as [tree](#tree), but refuse to render template, if the KV prefix query return blank/empty data.

This is especially useful, for rendering mission critical files, that are being populated by consul-template.

For example:

```text
/root/.ssh/authorized_keys
/etc/sysconfig/iptables
```

Using `safeTree` on empty prefixes will result in template output not being rendered at all.

To learn how `safeTree` was born see [CT-1131](https://github.com/hashicorp/consul-template/issues/1131) [C-3975](https://github.com/hashicorp/consul/issues/3975) and [CR-82](https://github.com/hashicorp/consul-replicate/issues/82).

---

#### Scratch

The scratchpad (or "scratch" for short) is available within the context of a
template to store temporary data or computations. Scratch data is not shared
between templates and is not cached between invocations.

##### `scratch.Key`

Returns a boolean if data exists in the scratchpad at the named key. Even if the
data at that key is "nil", this still returns true.

```liquid
{{ scratch.Key "foo" }}
```

##### `scratch.Get`

Returns the value in the scratchpad at the named key. If the data does not
exist, this will return "nil".

```liquid
{{ scratch.Get "foo" }}
```

##### `scratch.Set`

Saves the given value at the given key. If data already exists at that key, it
is overwritten.

```liquid
{{ scratch.Set "foo" "bar" }}
```

##### `scratch.SetX`

This behaves exactly the same as `Set`, but does not overwrite if the value
already exists.

```liquid
{{ scratch.SetX "foo" "bar" }}
```

##### `scratch.MapSet`

Saves a value in a named key in the map. If data already exists at that key, it
is overwritten.

```liquid
{{ scratch.MapSet "vars" "foo" "bar" }}
```

##### `scratch.MapSetX`

This behaves exactly the same as `MapSet`, but does not overwrite if the value
already exists.

```liquid
{{ scratch.MapSetX "vars" "foo" "bar" }}
```

##### `scratch.MapValues`

Returns a sorted list (by key) of all values in the named map.

```liquid
{{ scratch.MapValues "vars" }}
```

---

#### Helper Functions

Unlike API functions, helper functions do not query remote services. These
functions are useful for parsing data, formatting data, performing math, etc.

##### `base64Decode`

Accepts a base64-encoded string and returns the decoded result, or an error if
the given string is not a valid base64 string.

```liquid
{{ base64Decode "aGVsbG8=" }}
```

renders

```text
hello
```

##### `base64Encode`

Accepts a string and returns a base64-encoded string.

```liquid
{{ base64Encode "hello" }}
```

renders

```text
aGVsbG8=
```

##### `base64URLDecode`

Accepts a base64-encoded URL-safe string and returns the decoded result, or an
error if the given string is not a valid base64 URL-safe string.

```liquid
{{ base64URLDecode "aGVsbG8=" }}
```

renders

```text
hello
```

##### `base64URLEncode`

Accepts a string and returns a base-64 encoded URL-safe string.

```liquid
{{ base64Encode "hello" }}
```

renders

```text
aGVsbG8=
```

##### `byKey`

Accepts a list of pairs returned from a [`tree`](#tree) call and creates a map that groups pairs by their top-level directory.

For example:

```text
groups/elasticsearch/es1
groups/elasticsearch/es2
groups/elasticsearch/es3
services/elasticsearch/check_elasticsearch
services/elasticsearch/check_indexes
```

with the following template

```liquid
{{ range $key, $pairs := tree "groups" | byKey }}{{ $key }}:
{{ range $pair := $pairs }}  {{ .Key }}={{ .Value }}
{{ end }}{{ end }}
```

renders

```text
elasticsearch:
  es1=1
  es2=1
  es3=1
```

Note that the top-most key is stripped from the Key value. Keys that have no
prefix after stripping are removed from the list.

The resulting pairs are keyed as a map, so it is possible to look up a single
value by key:

```liquid
{{ $weights := tree "weights" }}
{{ range service "release.web" }}
  {{ $weight := or (index $weights .Node) 100 }}
  server {{ .Node }} {{ .Address }}:{{ .Port }} weight {{ $weight }}{{ end }}
```

##### `byTag`

Takes the list of services returned by the [`service`](#service) or
[`services`](#services) function and creates a map that groups services by tag.

```liquid
{{ range $tag, $services := service "web" | byTag }}{{ $tag }}
{{ range $services }} server {{ .Name }} {{ .Address }}:{{ .Port }}
{{ end }}{{ end }}
```

##### `byMeta`

Takes a list of services returned by [`service`](#service) and returns a map
that groups services by ServiceMeta values. Multiple service meta keys can be
passed as a comma separated string. `|int` can be added to a meta key to
convert numbers from service meta values to padded numbers in `printf "%05d" %
value` format (useful for sorting as Go Template sorts maps by keys).

**Example**:

If we have the following services registered in Consul:

```json
{
  "Services": [
     {
       "ID": "redis-dev-1",
       "Name": "redis",
       "ServiceMeta": {
         "environment": "dev",
         "shard_number": "1"
       },
       ...
     },
     {
       "ID": "redis-prod-1",
       "Name": "redis",
       "ServiceMeta": {
         "environment": "prod",
         "shard_number": "1"
       },
       ...
     },
     {
       "ID": "redis-prod-2",
       "Name": "redis",
       "ServiceMeta": {
         "environment": "prod",
         "shard_number": "2",
       },
       ...
     }
   ]
}
```

```liquid
{{ service "redis|any" | byMeta "environment,shard_number|int" | toJSON }}
```

The code above will produce a map of services grouped by meta:

```json
{
  "dev_00001": [
    {
      "ID": "redis-dev-1",
      ...
    }
  ],
  "prod_00001": [
    {
      "ID": "redis-prod-1",
      ...
    }
  ],
  "prod_00002": [
    {
      "ID": "redis-prod-2",
      ...
    }
  ]
}
```

##### `contains`

Determines if a needle is within an iterable element.

```liquid
{{ if .Tags | contains "production" }}
# ...
{{ end }}
```

##### `containsAll`

Returns `true` if all needles are within an iterable element, or `false`
otherwise. Returns `true` if the list of needles is empty.

```liquid
{{ if containsAll $requiredTags .Tags }}
# ...
{{ end }}
```

##### `containsAny`

Returns `true` if any needle is within an iterable element, or `false`
otherwise. Returns `false` if the list of needles is empty.

```liquid
{{ if containsAny $acceptableTags .Tags }}
# ...
{{ end }}
```

##### `containsNone`

Returns `true` if no needles are within an iterable element, or `false`
otherwise. Returns `true` if the list of needles is empty.

```liquid
{{ if containsNone $forbiddenTags .Tags }}
# ...
{{ end }}
```

##### `containsNotAll`

Returns `true` if some needle is not within an iterable element, or `false`
otherwise. Returns `false` if the list of needles is empty.

```liquid
{{ if containsNotAll $excludingTags .Tags }}
# ...
{{ end }}
```

##### `env`

Reads the given environment variable accessible to the current process.

```liquid
{{ env "CLUSTER_ID" }}
```

This function can be chained to manipulate the output:

```liquid
{{ env "CLUSTER_ID" | toLower }}
```

Reads the given environment variable and if it does not exist or is blank use a default value, ex `12345`.

```liquid
{{ or (env "CLUSTER_ID") "12345" }}
```

##### `executeTemplate`

Executes and returns a defined template.

```liquid
{{ define "custom" }}my custom template{{ end }}

This is my other template:
{{ executeTemplate "custom" }}

And I can call it multiple times:
{{ executeTemplate "custom" }}

Even with a new context:
{{ executeTemplate "custom" 42 }}

Or save it to a variable:
{{ $var := executeTemplate "custom" }}
```

##### `explode`

Takes the result from a `tree` or `ls` call and converts it into a deeply-nested
map for parsing/traversing.

```liquid
{{ tree "config" | explode }}
```

Note: You will lose any metadata about the key-pair after it has been exploded.
You can also access deeply nested values:

```liquid
{{ with tree "config" | explode }}
{{ .a.b.c }}{{ end }}
```

You will need to have a reasonable format about your data in Consul. Please see
[Go's text/template package][text-template] for more information.


##### `explodeMap`

Takes the value of a map and converts it into a deeply-nested map for parsing/traversing,
using the same logic as `explode`.

```liquid
{{ scratch.MapSet "example", "foo/bar", "a" }}
{{ scratch.MapSet "example", "foo/baz", "b" }}
{{ scratch.Get "example" | explodeMap | toYAML }}
```

##### `indent`

Indents a block of text by prefixing N number of spaces per line.

```liquid
{{ tree "foo" | explode | toYAML | indent 4 }}
```

##### `in`

Determines if a needle is within an iterable element.

```liquid
{{ if in .Tags "production" }}
# ...
{{ end }}
```

##### `loop`

Accepts varying parameters and differs its behavior based on those parameters.

If `loop` is given one integer, it will return a goroutine that begins at zero
and loops up to but not including the given integer:

```liquid
{{ range loop 5 }}
# Comment{{end}}
```

If given two integers, this function will return a goroutine that begins at
the first integer and loops up to but not including the second integer:

```liquid
{{ range $i := loop 5 8 }}
stanza-{{ $i }}{{ end }}
```

which would render:

```text
stanza-5
stanza-6
stanza-7
```

Note: It is not possible to get the index and the element since the function
returns a goroutine, not a slice. In other words, the following is **not
valid**:

```liquid
# Will NOT work!
{{ range $i, $e := loop 5 8 }}
# ...{{ end }}
```

##### `join`

Takes the given list of strings as a pipe and joins them on the provided string:

```liquid
{{ $items | join "," }}
```

##### `trimSpace`

Takes the provided input and trims all whitespace, tabs and newlines:

```liquid
{{ file "/etc/ec2_version" | trimSpace }}
```

##### `parseBool`

Takes the given string and parses it as a boolean:

```liquid
{{ "true" | parseBool }}
```

This can be combined with a key and a conditional check, for example:

```liquid
{{ if key "feature/enabled" | parseBool }}{{ end }}
```

##### `parseFloat`

Takes the given string and parses it as a base-10 float64:

```liquid
{{ "1.2" | parseFloat }}
```

##### `parseInt`

Takes the given string and parses it as a base-10 int64:

```liquid
{{ "1" | parseInt }}
```

This can be combined with other helpers, for example:

```liquid
{{ range $i := loop key "config/pool_size" | parseInt }}
# ...{{ end }}
```

##### `parseJSON`

Takes the given input (usually the value from a key) and parses the result as
JSON:

```liquid
{{ with $d := key "user/info" | parseJSON }}{{ $d.name }}{{ end }}
```

Note: Consul Template evaluates the template multiple times, and on the first
evaluation the value of the key will be empty (because no data has been loaded
yet). This means that templates must guard against empty responses.

##### `parseUint`

Takes the given string and parses it as a base-10 int64:

```liquid
{{ "1" | parseUint }}
```

##### `parseYAML`

Takes the given input (usually the value from a key) and parses the result as
YAML:

```liquid
{{ with $d := key "user/info" | parseYAML }}{{ $d.name }}{{ end }}
```

Note: The same caveats that apply to `parseJSON` apply to `parseYAML`.

##### `plugin`

Takes the name of a plugin and optional payload and executes a Consul Template
plugin.

```liquid
{{ plugin "my-plugin" }}
```

The plugin can take an arbitrary number of string arguments, and can be the
target of a pipeline that produces strings as well. This is most commonly
combined with a JSON filter for customization:

```liquid
{{ tree "foo" | explode | toJSON | plugin "my-plugin" }}
```

Please see the [plugins](#plugins) section for more information about plugins.

##### `regexMatch`

Takes the argument as a regular expression and will return `true` if it matches
on the given string, or `false` otherwise.

```liquid
{{ if "foo.bar" | regexMatch "foo([.a-z]+)" }}
# ...
{{ else }}
# ...
{{ end }}
```

##### `regexReplaceAll`

Takes the argument as a regular expression and replaces all occurrences of the
regex with the given string. As in go, you can use variables like $1 to refer to
subexpressions in the replacement string.

```liquid
{{ "foo.bar" | regexReplaceAll "foo([.a-z]+)" "$1" }}
```

##### `replaceAll`

Takes the argument as a string and replaces all occurrences of the given string
with the given string.

```liquid
{{ "foo.bar" | replaceAll "." "_" }}
```

This function can be chained with other functions as well:

```liquid
{{ service "web" }}{{ .Name | replaceAll ":" "_" }}{{ end }}
```

##### `sha256Hex`

Takes the argument as a string and compute the sha256_hex value

```liquid
{{ "bladibla" | sha256Hex }}
```

##### `split`

Splits the given string on the provided separator:

```liquid
{{ "foo\nbar\n" | split "\n" }}
```

This can be combined with chained and piped with other functions:

```liquid
{{ key "foo" | toUpper | split "\n" | join "," }}
```

##### `timestamp`

Returns the current timestamp as a string (UTC). If no arguments are given, the
result is the current RFC3339 timestamp:

```liquid
{{ timestamp }} // e.g. 1970-01-01T00:00:00Z
```

If the optional parameter is given, it is used to format the timestamp. The
magic reference date **Mon Jan 2 15:04:05 -0700 MST 2006** can be used to format
the date as required:

```liquid
{{ timestamp "2006-01-02" }} // e.g. 1970-01-01
```

See [Go's `time.Format`](http://golang.org/pkg/time/#Time.Format) for more
information.

As a special case, if the optional parameter is `"unix"`, the unix timestamp in
seconds is returned as a string.

```liquid
{{ timestamp "unix" }} // e.g. 0
```

##### `toJSON`

Takes the result from a `tree` or `ls` call and converts it into a JSON object.

```liquid
{{ tree "config" | explode | toJSON }}
```

renders

```javascript
{"admin":{"port":"1234"},"maxconns":"5","minconns":"2"}
```

Note: Consul stores all KV data as strings. Thus true is "true", 1 is "1", etc.

##### `toJSONPretty`

Takes the result from a `tree` or `ls` call and converts it into a
pretty-printed JSON object, indented by two spaces.

```liquid
{{ tree "config" | explode | toJSONPretty }}
```

renders

```javascript
{
  "admin": {
    "port": "1234"
  },
  "maxconns": "5",
  "minconns": "2",
}
```

Note: Consul stores all KV data as strings. Thus true is "true", 1 is "1", etc.

##### `toLower`

Takes the argument as a string and converts it to lowercase.

```liquid
{{ key "user/name" | toLower }}
```

See [Go's `strings.ToLower`](http://golang.org/pkg/strings/#ToLower) for more
information.

##### `toTitle`

Takes the argument as a string and converts it to titlecase.

```liquid
{{ key "user/name" | toTitle }}
```

See [Go's `strings.Title`](http://golang.org/pkg/strings/#Title) for more
information.

##### `toTOML`

Takes the result from a `tree` or `ls` call and converts it into a TOML object.

```liquid
{{ tree "config" | explode | toTOML }}
```

renders

```toml
maxconns = "5"
minconns = "2"

[admin]
  port = "1134"
```

Note: Consul stores all KV data as strings. Thus true is "true", 1 is "1", etc.

##### `toUpper`

Takes the argument as a string and converts it to uppercase.

```liquid
{{ key "user/name" | toUpper }}
```

See [Go's `strings.ToUpper`](http://golang.org/pkg/strings/#ToUpper) for more
information.

##### `toYAML`

Takes the result from a `tree` or `ls` call and converts it into a
pretty-printed YAML object, indented by two spaces.

```liquid
{{ tree "config" | explode | toYAML }}
```

renders

```yaml
admin:
  port: "1234"
maxconns: "5"
minconns: "2"
```

Note: Consul stores all KV data as strings. Thus true is "true", 1 is "1", etc.

##### `sockaddr`

Takes a quote-escaped template string as an argument and passes it on to
[hashicorp/go-sockaddr](https://github.com/hashicorp/go-sockaddr) templating engine.

```liquid
{{ sockaddr "GetPrivateIP" }}
```

See [hashicorp/go-sockaddr documentation](https://godoc.org/github.com/hashicorp/go-sockaddr)
for more information.

---

#### Math Functions

The following functions are available on floats and integer values.

##### `add`

Returns the sum of the two values.

```liquid
{{ add 1 2 }} // 3
```

This can also be used with a pipe function.

```liquid
{{ 1 | add 2 }} // 3
```

##### `subtract`

Returns the difference of the second value from the first.

```liquid
{{ subtract 2 5 }} // 3
```

This can also be used with a pipe function.

```liquid
{{ 5 | subtract 2 }} // 3
```

Please take careful note of the order of arguments.

##### `multiply`

Returns the product of the two values.

```liquid
{{ multiply 2 2 }} // 4
```

This can also be used with a pipe function.

```liquid
{{ 2 | multiply 2 }} // 4
```

##### `divide`

Returns the division of the second value from the first.

```liquid
{{ divide 2 10 }} // 5
```

This can also be used with a pipe function.

```liquid
{{ 10 | divide 2 }} // 5
```

Please take careful note of the order or arguments.

##### `modulo`

Returns the modulo of the second value from the first.

```liquid
{{ modulo 2 5 }} // 1
```

This can also be used with a pipe function.

```liquid
{{ 5 | modulo 2 }} // 1
```

Please take careful note of the order of arguments.

##### `minimum`

Returns the minimum of the two values.

```liquid
{{ minimum 2 5 }} // 2
```

This can also be used with a pipe function.

```liquid
{{ 5 | minimum 2 }} // 2
```

##### `maximum`

Returns the maximum of the two values.

```liquid
{{ maximum 2 5 }} // 2
```

This can also be used with a pipe function.

```liquid
{{ 5 | maximum 2 }} // 2
```

## Plugins

### Authoring Plugins

For some use cases, it may be necessary to write a plugin that offloads work to
another system. This is especially useful for things that may not fit in the
"standard library" of Consul Template, but still need to be shared across
multiple instances.

Consul Template plugins must have the following API:

```shell
$ NAME [INPUT...]
```

- `NAME` - the name of the plugin - this is also the name of the binary, either
  a full path or just the program name.  It will be executed in a shell with the
  inherited `PATH` so e.g. the plugin `cat` will run the first executable `cat`
  that is found on the `PATH`.

- `INPUT` - input from the template. There will be one INPUT for every argument passed
  to the `plugin` function. If the arguments contain whitespace, that whitespace
  will be passed as if the argument were quoted by the shell.

#### Important Notes

- Plugins execute user-provided scripts and pass in potentially sensitive data
  from Consul or Vault. Nothing is validated or protected by Consul Template,
  so all necessary precautions and considerations should be made by template
  authors

- Plugin output must be returned as a string on stdout. Only stdout will be
  parsed for output. Be sure to log all errors, debugging messages onto stderr
  to avoid errors when Consul Template returns the value. Note that output to
  stderr will only be output if the plugin returns a non-zero exit code.

- Always `exit 0` or Consul Template will assume the plugin failed to execute

- Ensure the empty input case is handled correctly (see [Multi-phase execution](#multi-phase-execution))

- Data piped into the plugin is appended after any parameters given explicitly (eg `{{ "sample-data" | plugin "my-plugin" "some-parameter"}}` will call `my-plugin some-parameter sample-data`)

Here is a sample plugin in a few different languages that removes any JSON keys
that start with an underscore and returns the JSON string:

```ruby
#! /usr/bin/env ruby
require "json"

if ARGV.empty?
  puts JSON.fast_generate({})
  Kernel.exit(0)
end

hash = JSON.parse(ARGV.first)
hash.reject! { |k, _| k.start_with?("_")  }
puts JSON.fast_generate(hash)
Kernel.exit(0)
```

```go
func main() {
  arg := []byte(os.Args[1])

  var parsed map[string]interface{}
  if err := json.Unmarshal(arg, &parsed); err != nil {
    fmt.Fprintln(os.Stderr, fmt.Sprintf("err: %s", err))
    os.Exit(1)
  }

  for k, _ := range parsed {
    if string(k[0]) == "_" {
      delete(parsed, k)
    }
  }

  result, err := json.Marshal(parsed)
  if err != nil {
    fmt.Fprintln(os.Stderr, fmt.Sprintf("err: %s", err))
    os.Exit(1)
  }

  fmt.Fprintln(os.Stdout, fmt.Sprintf("%s", result))
  os.Exit(0)
}
```

## Caveats

### Docker Image Use

The Alpine Docker image is configured to support an external volume to render
shared templates to. If mounted you will need to make sure that the
consul-template user in the docker image has write permissions to the
directory. Also if you build your own image using these you need to be sure you
have the permissions correct.

**The consul-template user in docker has a UID of 100 and a GID of 1000.**

This effects the in image directories /consul-template/config, used to add
configuration when using this as a parent image, and /consul-template/data,
exported as a VOLUME as a location to render shared results.

Previously the image initially ran as root in order to ensure the permissions
allowed it. But this ran against docker best practices and security policies.

If you build your own image based on ours you can override these values with
`--build-arg` parameters.

### Dots in Service Names

Using dots `.` in service names will conflict with the use of dots for [TAG
delineation](https://github.com/hashicorp/consul-template#service) in the
template. Dots already [interfere with using
DNS](https://www.consul.io/docs/agent/services.html#service-and-tag-names-with-dns)
for service names, so we recommend avoiding dots wherever possible.

### Once Mode

In Once mode, Consul Template will wait for all dependencies to be rendered. If
a template specifies a dependency (a request) that does not exist in Consul,
once mode will wait until Consul returns data for that dependency. Please note
that "returned data" and "empty data" are not mutually exclusive.

When you query for all healthy services named "foo" (`{{ service "foo" }}`), you
are asking Consul - "give me all the healthy services named foo". If there are
no services named foo, the response is the empty array. This is also the same
response if there are no _healthy_ services named foo.

Consul template processes input templates multiple times, since the first result
could impact later dependencies:

```liquid
{{ range services }}
{{ range service .Name }}
{{ end }}
{{ end }}
```

In this example, we have to process the output of `services` before we can
lookup each `service`, since the inner loops cannot be evaluated until the outer
loop returns a response. Consul Template waits until it gets a response from
Consul for all dependencies before rendering a template. It does not wait until
that response is non-empty though.

**Note:** Once mode implicitly disables any wait/quiescence timers specified in configuration files or passed on the command line.

### Termination on Error

By default Consul Template is highly fault-tolerant. If Consul is unreachable or
a template changes, Consul Template will happily continue running. The only
exception to this rule is if the optional `command` exits non-zero. In this
case, Consul Template will also exit non-zero. The reason for this decision is
so the user can easily configure something like Upstart or God to manage Consul
Template as a service.

If you want Consul Template to continue watching for changes, even if the
optional command argument fails, you can append `|| true` to your command. Note
that `||` is a "shell-ism", not a built-in function. You will also need to run
your command under a shell:

```shell
$ consul-template \
  -template "in.ctmpl:out.file:/bin/bash -c 'service nginx restart || true'"
```

In this example, even if the Nginx restart command returns non-zero, the overall
function will still return an OK exit code; Consul Template will continue to run
as a service. Additionally, if you have complex logic for restarting your
service, you can intelligently choose when you want Consul Template to exit and
when you want it to continue to watch for changes. For these types of complex
scripts, we recommend using a custom sh or bash script instead of putting the
logic directly in the `consul-template` command or configuration file.

### Commands

#### Environment

The current processes environment is used when executing commands with the following additional environment variables:

- `CONSUL_HTTP_ADDR`
- `CONSUL_HTTP_TOKEN`
- `CONSUL_HTTP_AUTH`
- `CONSUL_HTTP_SSL`
- `CONSUL_HTTP_SSL_VERIFY`

These environment variables are exported with their current values when the
command executes. Other Consul tooling reads these environment variables,
providing smooth integration with other Consul tools (like `consul maint` or
`consul lock`). Additionally, exposing these environment variables gives power
users the ability to further customize their command script.

#### Multiple Commands

The command configured for running on template rendering must be a single
command. That is you cannot join multiple commands with `&&`, `;`, `|`, etc.
This is a restriction of how they are executed. **However** you are able to do
this by combining the multiple commands in an explicit shell command using `sh
-c`. This is probably best explained by example.

Say you have a couple scripts you need to run when a template is rendered,
`/opt/foo` and `/opt/bar`, and you only want `/opt/bar` to run if `/opt/foo` is
successful. You can do that with the command...

`command = "sh -c '/opt/foo && /opt/bar'"`

As this is a full shell command you can even use conditionals. So accomplishes the same thing.

`command = "sh -c 'if /opt/foo; then /opt/bar ; fi'"`

Using this method you can run as many shell commands as you need with whatever
logic you need. Though it is suggested that if it gets too long you might want
to wrap it in a shell script, deploy and run that.

### Multi-phase Execution

Consul Template does an n-pass evaluation of templates, accumulating
dependencies on each pass. This is required due to nested dependencies, such as:

```liquid
{{ range services }}
{{ range service .Name }}
  {{ .Address }}
{{ end }}{{ end }}
```

During the first pass, Consul Template does not know any of the services in
Consul, so it has to perform a query. When those results are returned, the
inner-loop is then evaluated with that result, potentially creating more queries
and watches.

Because of this implementation, template functions need a default value that is
an acceptable parameter to a `range` function (or similar), but does not
actually execute the inner loop (which would cause a panic). This is important
to mention because complex templates **must** account for the "empty" case. For
example, the following **will not work**:

```liquid
{{ with index (service "foo") 0 }}
# ...
{{ end }}
```

This will raise an error like:

```text
<index $services 0>: error calling index: index out of range: 0
```

That is because, during the _first_ evaluation of the template, the `service`
key is returning an empty slice. You can account for this in your template like
so:

```liquid
{{ with service "foo" }}
{{ with index . 0 }}
{{ .Node }}{{ end }}{{ end }}
```

This will still add the dependency to the list of watches, but will not
evaluate the inner-if, avoiding the out-of-index error.

## Running and Process Lifecycle

While there are multiple ways to run Consul Template, the most common pattern is
to run Consul Template as a system service. When Consul Template first starts,
it reads any configuration files and templates from disk and loads them into
memory. From that point forward, changes to the files on disk do not propagate
to running process without a reload.

The reason for this behavior is simple and aligns with other tools like haproxy.
A user may want to perform pre-flight validation checks on the configuration or
templates before loading them into the process. Additionally, a user may want to
update configuration and templates simultaneously. Having Consul Template
automatically watch and reload those files on changes is both operationally
dangerous and against some of the paradigms of modern infrastructure. Instead,
Consul Template listens for the `SIGHUP` syscall to trigger a configuration
reload. If you update configuration or templates, simply send `HUP` to the
running Consul Template process and Consul Template will reload all the
configurations and templates from disk.

## Debugging

Consul Template can print verbose debugging output. To set the log level for
Consul Template, use the `-log-level` flag:

```shell
$ consul-template -log-level info ...
```

```text
<timestamp> [INFO] (cli) received redis from Watcher
<timestamp> [INFO] (cli) invoking Runner
# ...
```

You can also specify the level as debug:

```shell
$ consul-template -log-level debug ...
```

```text
<timestamp> [DEBUG] (cli) creating Runner
<timestamp> [DEBUG] (cli) creating Consul API client
<timestamp> [DEBUG] (cli) creating Watcher
<timestamp> [DEBUG] (cli) looping for data
<timestamp> [DEBUG] (watcher) starting watch
<timestamp> [DEBUG] (watcher) all pollers have started, waiting for finish
<timestamp> [DEBUG] (redis) starting poll
<timestamp> [DEBUG] (service redis) querying Consul with &{...}
<timestamp> [DEBUG] (service redis) Consul returned 2 services
<timestamp> [DEBUG] (redis) writing data to channel
<timestamp> [DEBUG] (redis) starting poll
<timestamp> [INFO] (cli) received redis from Watcher
<timestamp> [INFO] (cli) invoking Runner
<timestamp> [DEBUG] (service redis) querying Consul with &{...}
# ...
```

## Telemetry

Consul Template uses the [OpenTelemetry](https://opentelemetry.io/) project for its monitoring engine to collect various runtime metrics. It currently supports metrics exported to stdout, dogstatsd server, and prometheus endpoint.

### Key Metrics

These metrics offer insight into Consul Template and capture subprocess activities. The number of dependencies are aggregated from the configured templates, and metrics are collected around a dependency when it is updated from source. This is useful to correlate any upstream changes to downstream actions originating from Consul Template.

Metrics are monitored around template rendering and execution of template commands. These
metrics indicate the rendering status of a template and how long commands for a template takes
to provide insight on performance of the templates.

| Metric Name | Labels | Description |
|-|:-:|-|
| `consul-template.dependencies` | type=(consul\|vault\|local) | The number of dependencies grouped by types |
| `consul-template.dependencies_received` | type=(consul\|vault\|local), id=dependencyString | A counter of dependencies received from monitoring value changes |
| `consul-template.templates` | | The number of templates configured |
| `consul-template.templates_rendered` | id=templateID, status=(rendered\|would\|quiescence) | A counter of templates rendered |
| `consul-template.runner_actions` | action=(start\|stop\|run) | A count of runner actions |
| `consul-template.commands_exec` | status=(success\|error) | The number of commands executed after rendering templates |
| `consul-template.commands_exec_time` | id=tmplDestination | The execution time (seconds) of a template command |
| `consul-template.vault.token` | status=(configured\|renewed\|expired\|stopped) | A counter of vault token renewal statuses |

### Metric Samples

#### Stdout

```
{"time":"2020-05-05T12:02:16.028883-05:00","updates":[{"name":"consul-template.dependencies{type=consul}","min":2,"max":2,"sum":4,"count":2,"quantiles":[{"q":0.5,"v":2},{"q":0.9,"v":2},{"q":0.99,"v":2}]},{"name":"consul-template.commands_exec_time{destination=out.txt}","min":0.008301234,"max":0.008301234,"sum":0.008301234,"count":1,"quantiles":[{"q":0.5,"v":0.008301234},{"q":0.9,"v":0.008301234},{"q":0.99,"v":0.008301234}]},{"name":"consul-template.runner_actions{action=start}","sum":1},{"name":"consul-template.runner_actions{action=run}","sum":2},{"name":"consul-template.runner_actions{action=stop}","sum":1},{"name":"consul-template.templates_rendered{id=aadcafd7f28f1d9fc5e76ab2e029f844,status=rendered}","sum":1},{"name":"consul-template.dependencies_received{id=kv.block(hello),type=consul}","sum":1},{"name":"consul-template.templates","min":2,"max":2,"sum":2,"count":1,"quantiles":[{"q":0.5,"v":2},{"q":0.9,"v":2},{"q":0.99,"v":2}]},{"name":"consul-template.commands_exec{status=error}","sum":0},{"name":"consul-template.commands_exec{status=success}","sum":1}]}
```

#### DogStatsD

```
2020-05-05 11:57:46.143979 consul-template.runner_actions:1|c|#action:start
consul-template.runner_actions:2|c|#action:run
consul-template.dependencies_received:1|c|#id:kv.block(hello),type:consul
consul-template.dependencies:2|h|#type:consul
consul-template.templates:2|h
consul-template.templates_rendered:1|c|#id:aadcafd7f28f1d9fc5e76ab2e029f844,status:rendered
consul-template.commands_exec:1|c|#status:success
consul-template.commands_exec:0|c|#status:error
consul-template.commands_exec_time:0.011514017|h|#destination:out.txt
```

#### Prometheus

```
$ curl localhost:8888/metrics
# HELP consul_template_commands_exec The number of commands executed with labels status=(success|error)
# TYPE consul_template_commands_exec counter
consul_template_commands_exec{status="error"} 0
consul_template_commands_exec{status="success"} 1
# HELP consul_template_commands_exec_time The execution time (seconds) of a template command. The template destination is used as the identifier
# TYPE consul_template_commands_exec_time histogram
consul_template_commands_exec_time_bucket{destination="out.txt",le="+Inf"} 1
consul_template_commands_exec_time_sum{destination="out.txt"} 0.005063219
consul_template_commands_exec_time_count{destination="out.txt"} 1
# HELP consul_template_dependencies The number of dependencies grouped by types with labels type=(consul|vault|local)
# TYPE consul_template_dependencies histogram
consul_template_dependencies_bucket{type="consul",le="+Inf"} 2
consul_template_dependencies_sum{type="consul"} 4
consul_template_dependencies_count{type="consul"} 2
# HELP consul_template_dependencies_received A counter of dependencies received with labels type=(consul|vault|local) and id=dependencyString
# TYPE consul_template_dependencies_received counter
consul_template_dependencies_received{id="kv.block(hello)",type="consul"} 1
# HELP consul_template_runner_actions A count of runner actions with labels action=(start|stop|run)
# TYPE consul_template_runner_actions counter
consul_template_runner_actions{action="run"} 2
consul_template_runner_actions{action="start"} 1
# HELP consul_template_templates The number of templates configured.
# TYPE consul_template_templates histogram
consul_template_templates_bucket{le="+Inf"} 1
consul_template_templates_sum 2
consul_template_templates_count 1
# HELP consul_template_templates_rendered A counter of templates rendered with labels id=templateID and status=(rendered|would|quiescence)
# TYPE consul_template_templates_rendered counter
consul_template_templates_rendered{id="aadcafd7f28f1d9fc5e76ab2e029f844",status="rendered"} 1
```

## FAQ

**Q: How is this different than confd?**<br>
A: The answer is simple: Service Discovery as a first class citizen. You are also encouraged to read [this Pull Request](https://github.com/kelseyhightower/confd/pull/102) on the project for more background information. We think confd is a great project, but Consul Template fills a missing gap. Additionally, Consul Template has first class integration with [Vault](https://vaultproject.io), making it easy to incorporate secret material like database credentials or API tokens into configuration files.

**Q: How is this different than Puppet/Chef/Ansible/Salt?**<br>
A: Configuration management tools are designed to be used in unison with Consul Template. Instead of rendering a stale configuration file, use your configuration management software to render a dynamic template that will be populated by [Consul][].


## Contributing

To build and install Consul-Template locally, you will need to [install Go][go].

Clone the repository:

```shell
$ git clone https://github.com/hashicorp/consul-template.git
```

To compile the `consul-template` binary for your local machine:

```shell
$ make dev
```

This will compile the `consul-template` binary into `bin/consul-template` as
well as your `$GOPATH` and run the test suite.

If you want to compile a specific binary, set `XC_OS` and `XC_ARCH` or run the
following to generate all binaries:

```shell
$ make build
```

If you want to run the tests, first install [consul](https://www.consul.io/docs/install/index.html) and [vault](https://www.vaultproject.io/docs/install/) locally, then:

```shell
$ make test
```

Or to run a specific test in the suite:

```shell
go test ./... -run SomeTestFunction_name
```

[consul]: https://www.consul.io "Consul by HashiCorp"
[connect]: https://www.consul.io/docs/connect/ "Connect"
[examples]: (https://github.com/hashicorp/consul-template/tree/master/examples) "Consul Template Examples"
[releases]: https://releases.hashicorp.com/consul-template "Consul Template Releases"
[text-template]: https://golang.org/pkg/text/template/ "Go's text/template package"
[vault]: https://www.vaultproject.io "Vault by HashiCorp"
[go]: https://golang.org "Go programming language"
