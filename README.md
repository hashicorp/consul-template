Consul Template
===============
[![Latest Version](http://img.shields.io/github/release/hashicorp/consul-template.svg?style=flat-square)][release]
[![Build Status](http://img.shields.io/travis/hashicorp/consul-template.svg?style=flat-square)][travis]
[![Go Documentation](http://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)][godocs]

[release]: https://github.com/hashicorp/consul-template/releases
[travis]: http://travis-ci.org/hashicorp/consul-template
[godocs]: http://godoc.org/github.com/hashicorp/consul-template

This project provides a convenient way to populate values from [Consul][] into the filesystem using the `consul-template` daemon.

The daemon `consul-template` queries a [Consul][] instance and updates any number of specified templates on the filesystem. As an added bonus, `consul-template` can optionally run arbitrary commands when the update process completes. See the [Examples](#examples) section for some scenarios where this functionality might prove useful.

**The documentation in this README corresponds to the master branch of Consul Template. It may contain unreleased features or different APIs than the most recently released version. Please see the Git tag that corresponds to your version of Consul Template for the proper documentation.**


Installation
------------
You can download a released `consul-template` artifact from [the Consul Template release page][Releases] on GitHub. If you wish to compile from source, please see the instructions in the [Contributing][#Contributing] section.


Usage
-----
### Options
|       Option      | Description |
| ----------------- |------------ |
| `auth`            | The basic authentication username (and optional password), separated by a colon. There is no default value.
| `consul`*         | The location of the Consul instance to query (may be an IP address or FQDN) with port.
| `max-stale`       | The maximum staleness of a query. If specified, Consul will distribute work among all servers instead of just the leader. The default value is 1s.
| `ssl`             | Use HTTPS while talking to Consul. Requires the Consul server to be configured to serve secure connections. The default value is false.
| `ssl-verify`      | Verify certificates when connecting via SSL. This requires the use of `-ssl`. The default value is true.
| `ssl-cert`        | Path to an SSL client certificate to use to authenticate to the consul server. Useful if the consul server "verify_incoming" option is set.
| `ssl-ca-cert`     | Path to a CA certificate file, containing one or more CA certificates to use to validate the certificate sent by the consul server to us. This is a handy alternative to setting ```--ssl-verify=false``` if you are using your own CA.
| `syslog`          | Send log output to syslog (in addition to stdout and stderr). The default value is false.
| `syslog-facility` | The facility to use when sending to syslog. This requires the use of `-syslog`. The default value is `LOCAL0`.
| `token`           | The [Consul API token][Consul ACLs]. There is no default value.
| `template`*       | The input template, output path, and optional command separated by a colon (`:`). This option is additive and may be specified multiple times for multiple templates.
| `wait`            | The `minimum(:maximum)` to wait before rendering a new template to disk and triggering a command, separated by a colon (`:`). If the optional maximum value is omitted, it is assumed to be 4x the required minimum value. There is no default value.
| `retry`           | The amount of time to wait if Consul returns an error when communicating with the API. The default value is 5 seconds.
| `config`          | The path to a configuration file or directory of configuration files on disk, relative to the current working directory. Values specified on the CLI take precedence over values specified in the configuration file. There is no default value.
| `log-level`       | The log level for output. This applies to the stdout/stderr logging as well as syslog logging (if enabled). Valid values are "debug", "info", "warn", and "err". The default value is "warn".
| `pid-file`        | The path on disk to write Consul Template's PID file
| `dry`             | Dump generated templates to the console. If specified, generated templates are not committed to disk and commands are not invoked. _(CLI-only)_
| `once`            | Run Consul Template once and exit (as opposed to the default behavior of daemon). _(CLI-only)_
| `version`         | Output version information and quit. _(CLI-only)_

\* = Required parameter

### Command Line
The CLI interface supports all of the options detailed above.

Query the nyc3 demo Consul instance, rendering the template on disk at `/tmp/template.ctmpl` to `/tmp/result`, running Consul Template as a service until stopped:

```shell
$ consul-template \
  -consul demo.consul.io \
  -template "/tmp/template.ctmpl:/tmp/result"
```

Query a local Consul instance, rendering the template and restarting nginx if the template has changed, once, polling 30s if Consul is unavailable:

```shell
$ consul-template \
  -consul 127.0.0.1:8500 \
  -template "/tmp/template.ctmpl:/var/www/nginx.conf:service nginx restart" \
  -retry 30s \
  -once
```

Query a Consul instance, rendering multiple templates and commands as a service until stopped:

```shell
$ consul-template \
  -consul my.consul.internal:6124 \
  -template "/tmp/nginx.ctmpl:/var/nginx/nginx.conf:service nginx restart" \
  -template "/tmp/redis.ctmpl:/var/redis/redis.conf:service redis restart" \
  -template "/tmp/haproxy.ctmpl:/var/haproxy/haproxy.conf"
```

Query a Consul instance that requires authentication, dumping the templates to stdout instead of writing to disk. In this example, the second and third parameters to the `-template` option are required but ignored. The file will not be written to disk and the optional command will not be executed:

```shell
$ consul-template \
  -consul my.consul.internal:6124 \
  -template "/tmp/template.ctmpl:/tmp/result:service nginx restart"
  -dry
```

Query a Consul that uses custom SSL certificates:

```shell
$ consul-template \
  -consul 127.0.0.1:8543 \
  -ssl \
  -ssl-cert /path/to/client/cert.pem \
  -ssl-ca-cert /path/to/ca/cert.pem \
  -template "/tmp/template.ctmpl:/tmp/result" \
  -dry \
  -once
```

### Configuration File(s)
The Consul Template configuration files are written in [HashiCorp Configuration Language (HCL)][HCL]. By proxy, this means the Consul Template configuration file is JSON-compatible. For more information, please see the [HCL specification][HCL].

The Configuration file syntax interface supports all of the options detailed above, unless otherwise noted in the table.

```javascript
consul = "127.0.0.1:8500"
token = "abcd1234" // May also be specified via the envvar CONSUL_TOKEN
retry = "10s"
max_stale = "10m"
log_level = "warn"
pid_file = "/path/to/pid"

vault {
  address = "https://vault.service.consul:8200"
  token = "abcd1234" // May also be specified via the envvar VAULT_TOKEN
  renew = true
  ssl {
    enabled = true
    verify = true
    cert = "/path/to/client/cert.pem"
    ca_cert = "/path/to/ca/cert.pem"
  }
}

auth {
  enabled = true
  username = "test"
  password = "test"
}

ssl {
  enabled = true
  verify = false
  cert = "/path/to/client/cert.pem"
  ca_cert = "/path/to/ca/cert.pem"
}

syslog {
  enabled = true
  facility = "LOCAL5"
}

template {
  source = "/path/on/disk/to/template"
  destination = "/path/on/disk/where/template/will/render"
  command = "optional command to run when the template is updated"
  perms = 0600
  backup = true
}

template {
  // Multiple template definitions are supported
}
```

Note: Not all fields are required. For example, if you are not using Vault secrets, you do not need to specify a vault configuration. Similarly, if you are not logging to syslog, you do not need to specify a syslog configuration.

For additional security, token may also be read from the environment using the `CONSUL_TOKEN` or `VAULT_TOKEN` environment variables respectively. It is highly recommended that you do not put your tokens in plain-text in a configuration file.

Query the nyc3 demo Consul instance, rendering the template on disk at `/tmp/template.ctmpl` to `/tmp/result`, running Consul Template as a service until stopped:

```javascript
consul = "nyc3.demo.consul.io"

template {
  source = "/tmp/template.ctmpl"
  destination  = "/tmp/result"
}
```

If a directory is given instead of a file, all files in the directory (recursively) will be merged in [lexical order](http://golang.org/pkg/path/filepath/#Walk). So if multiple files declare a "consul" key for instance, the last one will be used.

**Commands specified on the command line take precedence over those defined in a config file!**

### Templating Language
Consul Template consumes template files in the [Go Template][] format. If you are not familiar with the syntax, we recommend reading the documentation, but it is similar in appearance to Mustache, Handlebars, or Liquid.

In addition to the [Go-provided template functions][Go Template], Consul Template exposes the following functions:

#### API Functions

##### `datacenters`
Query Consul for all datacenters in the catalog. Datacenters are queried using the following syntax:

```liquid
{{datacenters}}
```

##### `file`
Read and output the contents of a local file on disk. If the file cannot be read, an error will occur. Files are read using the following syntax:

```liquid
{{file "/path/to/local/file"}}
```

This example will out the entire contents of the file at `/path/to/local/file` into the template. Note: this does not process nested templates.

##### `key`
Query Consul for the value at the given key. If the key cannot be converted to a string-like value, an error will occur. Keys are queried using the following syntax:

```liquid
{{key "service/redis/maxconns@east-aws"}}
```

The example above is querying Consul for the `service/redis/maxconns` in the east-aws datacenter. If you omit the datacenter attribute, the local Consul datacenter will be queried:

```liquid
{{key "service/redis/maxconns"}}
```

The beauty of Consul is that the key-value structure is entirely up to you!

##### `key_or_default`
Query Consul for the value at the given key. If no key exists at the given path, the default value will be used instead. The existing constraints and usage for keys apply:

```liquid
{{key_or_default "service/redis/maxconns@east-aws" "5"}}
```

Please note that Consul Template uses a multi-phase evaluation. During the first phase of evaluation, Consul Template will have no data from Consul and thus will _always_ fall back to the default value. Subsequent reads from Consul will pull in the real value from Consul (if the key exists) on the next template pass. This is important because it means that Consul Template will never "block" the rendering of a template due to a missing key from a `key_or_default`. Even if the key exists, if Consul has not yet returned data for the key, the
default value will be used instead.

##### `ls`
Query Consul for all top-level key-value pairs at the given prefix. If any of the values cannot be converted to a string-like value, an error will occur:

```liquid
{{range ls "service/redis@east-aws"}}
{{.Key}} {{.Value}}{{end}}
```

If the Consul instance had the correct structure at `service/redis` in the east-aws datacenter, the resulting template could look like:

```text
minconns 2
maxconns 12
```

If you omit the datacenter attribute on `ls`, the local Consul datacenter will be queried.

##### `node`
Query Consul for a single node in the catalog.

```liquid
{{node "node1"}}
```

When called without any arguments then the node for the current agent is returned.

```liquid
{{node}}
```

You can specify an optional parameter to the `nodes` call to specify the datacenter:

```liquid
{{node "node1" "@east-aws"}}
```

If the node specified is not found then `nil` is returned. If the node is found then you are provided with information about the node and a list of services that it provides.

```liquid
{{with node}}{{.Node.Node}} ({{.Node.Address}}){{range .Services}}
  {{.Service}} {{.Port}} ({{.Tags | join ","}}){{end}}
{{end}}
```

##### `nodes`
Query Consul for all nodes in the catalog. Nodes are queried using the following syntax:

```liquid
{{nodes}}
```

This example will query Consul's default datacenter. You can specify an optional parameter to the `nodes` call to specify the datacenter:

```liquid
{{nodes "@east-aws"}}
```

This will query Consul for all nodes in the east-aws datacenter.

##### `service`
Query Consul for the service group(s) matching the given pattern. Services are queried using the following syntax:

```liquid
{{service "release.web@east-aws"}}
```

The example above is querying Consul for healthy "web" services, in the "east-aws" datacenter. The tag and datacenter attributes are optional. To query all nodes of the "web" service (regardless of tag) for the current datacenter:

```liquid
{{service "web"}}
```

The function returns a `[]*Service` struct which can be used for ranging in a template:

```liquid
{{range service "web@datacenter"}}
server {{.Name}} {{.Address}}:{{.Port}}{{end}}
```

which would produce something like:

```text
server nyc_web_01 123.456.789.10:8080
server nyc_web_02 456.789.101.213:8080
```

By default only healthy services are returned. If you want to get all services, you can pass the "any" option:

```liquid
{{service "web" "any"}}
```

This will return all services registered to the agent, regardless of their status.

If you want to filter services by a specific health or health(s), you can specify a comma-separated list of health check statuses:

```liquid
{{service "web" "passing, warning"}}
```

This will returns services which are deemed "passing" or "warning" according to their node and service-level checks defined in Consul. Please note that the comma implies an "or", not an "and".

Specifying more than one status filter while "any" is used will return an error, since "any" is the superset of all status filters.

There is an architectural difference between the following:

```liquid
{{service "web"}}
{{service "web" "passing"}}
```

The former will return all services which Consul considers "healthy" and passing. The latter will return all services registered with the Consul agent and perform client-side filtering. As a general rule, you should not use the "passing" argument alone if you want only healthy services - simply omit the second argument instead. However, the extra argument is useful if you want "passing or warning" services like:

```liquid
{{service "web" "passing, warning"}}
```

The service's status is also exposed if you need to do additional filtering:

```liquid
{{range service "web" "any"}}
{{if eq .Status "critical"}}
// Critical state!{{end}}
{{if eq .Status "passing"}}
// Ok{{end}}
```

To put a service into maintenance mode in Consul around executing the command, simply wrap your command in a `consul maint` call:

```shell
#!/bin/sh
set -e
consul maint -enable -service web -reason "Consul Template updated"
service nginx reload
consul maint -disable -service web
```

Alternatively, if you do not have the Consul agent installed, you can make the API requests directly (advanced):

```shell
#!/bin/sh
set -e
curl -X PUT "http://$CONSUL_HTTP_ADDR/v1/agent/service/maintenance/web?enable=true&reason=Consul+Template+Updated"
service nginx reload
curl -X PUT "http://$CONSUL_HTTP_ADDR/v1/agent/service/maintenance/web?enable=false"
```

##### `services`
Query Consul for all services in the catalog. Services are queried using the following syntax:

```liquid
{{services}}
```

This example will query Consul's default datacenter. You can specify an optional parameter to the `services` call to specify the datacenter:

```liquid
{{services "@east-aws"}}
```

Please be advised: the `services` function is different than `service`, which accepts more parameters and queries the **health** endpoints for a list of services. This endpoint queries the Consul **catalog** which returns a map of services to its list of tags. For example:

```liquid
{{range services}}
{{.Name}}
{{range .Tags}}
  {{.}}{{end}}
{{end}}
```

##### `tree`
Query Consul for all key-value pairs at the given prefix. If any of the values cannot be converted to a string-like value, an error will occur:

```liquid
{{range tree "service/redis@east-aws"}}
{{.Key}} {{.Value}}{{end}}
```

If the Consul instance had the correct structure at `service/redis` in the east-aws datacenter, the resulting template could look like:

```text
minconns 2
maxconns 12
nested/config/value "value"
```

Unlike `ls`, `tree` returns **all** keys under the prefix, just like the Unix `tree` command.

If you omit the datacenter attribute on `tree`, the local Consul datacenter will be queried.

##### `vault`
Query [Vault](https://vaultproject.io) for the secret data at the given path. If the path does not exist or if the configured vault token does not have permission to read the path, an error will be returned.  If the path exists, but the key does not exist, `<no value>` will be returned.

```liquid
{{with vault "secret/passwords"}}{{.Data.password}}{{end}}
```

The following fields are available:

- `LeaseID` - the unique lease identifier
- `LeaseDuration` - the number of seconds the lease is valid
- `Renewable` - if the secret is renewable
- `Data` - the raw data - this is a `map[string]interface{}`, so it can be queried using Go's templating "dot notation"

If the map key has dots "." in it, you need to access the value using the `index` function:

```liquid
{{index .Data "my.key.with.dots"}}
```

Please always consider the security implications of having the contents of a secret in plain-text on disk. If an attacker is able to get access to the file, they will have access to plain-text secrets.

- - -

#### Helper Functions

##### `byKey`
Takes the list of key pairs returned from a [`tree`](#tree) function and creates a map that groups pairs by their top-level directory. For example, if the Consul KV store contained the following structure:

```text
groups/elasticsearch/es1
groups/elasticsearch/es2
groups/elasticsearch/es3
services/elasticsearch/check_elasticsearch
services/elasticsearch/check_indexes
```

With the following template:

```liquid
{{range $key, $pairs := tree "groups" | byKey}}{{$key}}:
{{range $pair := $pairs}}  {{.Key}}={{.Value}}
{{end}}{{end}}
```

The result would be:

```text
elasticsearch:
  es1=1
  es2=1
  es3=1
```

Note that the top-most key is stripped from the Key value. Keys that have no prefix after stripping are removed from the list.

The resulting pairs are keyed as a map, so it is possible to look up a single value by key:

```liquid
{{$weights := tree "weights"}}
{{range service "release.web"}}
  {{$weight := or (index $weights .Node) 100}}
  server {{.Node}} {{.Address}}:{{.Port}} weight {{$weight}}{{end}}
```

##### `byTag`
Takes the list of services returned by the [`service`](#service) or [`services`](#services) function and creates a map that groups services by tag.

```liquid
{{range $tag, $services := service "web" | byTag}}{{$tag}}
{{range $services}} server {{.Name}} {{.Address}}:{{.Port}}
{{end}}{{end}}
```

##### `contains`
Determines if a needle is within an iterable element.

```liquid
{{ if .Tags | contains "production" }}
# ...
{{ end }}
```

##### `env`
Reads the given environment variable accessible to the current process.

```liquid
{{env "CLUSTER_ID"}}
```

This function can be chained to manipulate the output:

```liquid
{{env "CLUSTER_ID" | toLower}}
```

##### `explode`
Takes the result from a `tree` or `ls` call and converts it into a deeply-nested map for parsing/traversing.

```liquid
{{ tree "config" | explode }}
```

Note: You will lose any metadata about the keypair after it has been exploded.

You can also access deeply nested values:

```liquid
{{ with tree "config" | explode }}
{{.a.b.c}}{{ end }}
```

Note: You will need to have a reasonable format about your data in Consul. Please see Golang's text/template package for more information.

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
{{range loop 5}}
# Comment{{end}}
```

If given two integers, this function will return a goroutine that begins at
the first integer and loops up to but not including the second integer:

```liquid
{{range $i := loop 5 8}}
stanza-{{$i}}{{end}}
```

which would render:

```text
stanza-5
stanza-6
stanza-7
```

Note: It is not possible to get the index and the element since the function
returns a goroutine, not a slice. In other words, the following is **not valid**:

```liquid
# Will NOT work!
{{range $i, $e := loop 5 8}}
# ...{{end}}
```

##### `join`
Takes the given list of strings as a pipe and joins them on the provided string:

```liquid
{{$items | join ","}}
```

##### `parseBool`
Takes the given string and parses it as a boolean:

```liquid
{{"true" | parseBool}}
```

This can be combined with a key and a conditional check, for example:

```liquid
{{if key "feature/enabled" | parseBool}}{{end}}
```

##### `parseFloat`
Takes the given string and parses it as a base-10 float64:

```liquid
{{"1.2" | parseFloat}}
```

##### `parseInt`
Takes the given string and parses it as a base-10 int64:

```liquid
{{"1" | parseInt}}
```

This can be combined with other helpers, for example:

```liquid
{{range $i := loop key "config/pool_size" | parseInt}}
# ...{{end}}
```

##### `parseJSON`
Takes the given input (usually the value from a key) and parses the result as JSON:

```liquid
{{with $d := key "user/info" | parseJSON}}{{$d.name}}{{end}}
```

Note: Consul Template evaluates the template multiple times, and on the first evaluation the value of the key will be empty (because no data has been loaded yet). This means that templates must guard against empty responses. For example:

```liquid
{{with $d := key "user/info" | parseJSON}}
{{if $d}}
...
{{end}}
{{end}}
```

It just works for simple keys. But fails if you want to iterate over keys or use `index` function. Wrapping code that access object with `{{ if $d }}...{{end}}` is good enough.

Alternatively you can read data from a local JSON file:

```liquid
{{with $d := file "/path/to/local/data.json" | parseJSON}}{{$d.some_key}}{{end}}
```

##### `parseUint`
Takes the given string and parses it as a base-10 int64:

```liquid
{{"1" | parseUint}}
```

See `parseInt` for examples.

##### `plugin`
Takes the name of a plugin and optional payload and executes a Consul Template plugin.

```liquid
{{plugin "my-plugin"}}
```

This is most commonly combined with a JSON filter for customization:

```liquid
{{tree "foo" | explode | toJSON | plugin "my-plugin"}}
```

Please see the [plugins](#plugins) section for more information about plugins.

##### `regexMatch`
Takes the argument as a regular expression and will return `true` if it matches on the given string, or `false` otherwise.

```liquid
{{"foo.bar" | regexMatch "foo([.a-z]+)"}}
```

##### `regexReplaceAll`
Takes the argument as a regular expression and replaces all occurences of the regex with the given string. As in go, you can use variables like $1 to refer to subexpressions in the replacement string.

```liquid
{{"foo.bar" | regexReplaceAll "foo([.a-z]+)" "$1"}}
```

##### `replaceAll`
Takes the argument as a string and replaces all occurences of the given string with the given string.

```liquid
{{"foo.bar" | replaceAll "." "_"}}
```

This function can be chained with other functions as well:

```liquid
{{service "web"}}{{.Name | replaceAll ":" "_"}}{{end}}
```

##### `split`
Splits the given string on the provided separator:

```liquid
{{"foo\nbar\n" | split "\n"}}
```

This can be combined with chained and piped with other functions:

```liquid
{{key "foo" | toUpper | split "\n" | join ","}}
```

##### `timestamp`
Returns the current timestamp as a string (UTC). If no arguments are given, the result is the current RFC3339 timestamp:

```liquid
{{timestamp}} // e.g. 1970-01-01T00:00:00Z
```

If the optional parameter is given, it is used to format the timestamp. The magic reference date **Mon Jan 2 15:04:05 -0700 MST 2006** can be used to format the date as required:

```liquid
{{timestamp "2006-01-02"}} // e.g. 1970-01-01
```

See Go's [time.Format()](http://golang.org/pkg/time/#Time.Format) for more information.

As a special case, if the optional parameter is `"unix"`, the unix timestamp in seconds is returned as a string.

```liquid
{{timestamp "unix"}} // e.g. 0
```


##### `toJSON`
Takes the result from a `tree` or `ls` call and converts it into a JSON object.

```liquid
{{ tree "config" | explode | toJSON }} // e.g. {"admin":{"port":1234},"maxconns":5,"minconns":2}
```

Note: This functionality should be considered final. If you need to manipulate keys, combine values, or perform mutations, that should be done _outside_ of Consul. In order to keep the API scope limited, we likely will not accept Pull Requests that focus on customizing the `toJSON` functionality.

##### `toJSONPretty`
Takes the result from a `tree` or `ls` call and converts it into a pretty-printed JSON object, indented by two spaces.

```liquid
{{ tree "config" | explode | toJSONPretty }}
/*
{
  "admin": {
    "port": 1234
  },
  "maxconns": 5,
  "minconns": 2,
}
*/
```

Note: This functionality should be considered final. If you need to manipulate keys, combine values, or perform mutations, that should be done _outside_ of Consul. In order to keep the API scope limited, we likely will not accept Pull Requests that focus on customizing the `toJSONPretty` functionality.

##### `toLower`
Takes the argument as a string and converts it to lowercase.

```liquid
{{key "user/name" | toLower}}
```

See Go's [strings.ToLower()](http://golang.org/pkg/strings/#ToLower) for more information.

##### `toTitle`
Takes the argument as a string and converts it to titlecase.

```liquid
{{key "user/name" | toTitle}}
```

See Go's [strings.Title()](http://golang.org/pkg/strings/#Title) for more information.

##### `toUpper`
Takes the argument as a string and converts it to uppercase.

```liquid
{{key "user/name" | toUpper}}
```

See Go's [strings.ToUpper()](http://golang.org/pkg/strings/#ToUpper) for more information.

##### `toYAML`
Takes the result from a `tree` or `ls` call and converts it into a pretty-printed YAML object, indented by two spaces.

```liquid
{{ tree "config" | explode | toYAML }}
/*
admin:
  port: 1234
maxconns: 5
minconns: 2
*/
```

Note: This functionality should be considered final. If you need to manipulate keys, combine values, or perform mutations, that should be done _outside_ of Consul. In order to keep the API scope limited, we likely will not accept Pull Requests that focus on customizing the `toYAML` functionality.

- - -

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
{{ 5 | subtract 2 }}
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

Plugins
-------
### Authoring Plugins
For some use cases, it may be necessary to write a plugin that offloads work to another system. This is especially useful for things that may not fit in the "standard library" of Consul Template, but still need to be shared across multiple instances.

Consul Template plugins must have the following API:

```shell
$ NAME [INPUT...]
```

- `NAME` - the name of the plugin - this is also the name of the binary, either a full path or just the program name.  It will be executed in a shell with the inherited `PATH` so e.g. the plugin `cat` will run the first executable `cat` that is found on the `PATH`.
- `INPUT` - input from the template - this wil always be JSON if provided

#### Important Notes

- Plugins execute user-provided scripts and pass in potentially sensitive data
  from Consul or Vault. Nothing is validated or protected by Consul Template,
  so all necessary precautions and considerations should be made by template
  authors
- Plugin output must be returned as a string on stdout. Only stdout will be
  parsed for output. Be sure to log all errors, debugging messages onto stderr
  to avoid errors when Consul Template returns the value.
- Always `exit 0` or Consul Template will assume the plugin failed to execute

Here is a sample plugin in a few different languages that removes any JSON keys that start with an underscore and returns the JSON string:

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

Caveats
-------
### Termination on Error
By default Consul Template is highly fault-tolerant. If Consul is unreachable or a template changes, Consul Template will happily continue running. The only exception to this rule is if the optional `command` exits non-zero. In this case, Consul Template will also exit non-zero. The reason for this decision is so the user can easily configure something like Upstart or God to manage Consul Template as a service.

If you want Consul Template to continue watching for changes, even if the optional command argument fails, you can append `|| true` to your command. For example:

```shell
$ consul-template \
  -template "in.ctmpl:out.file:service nginx restart || true"
```

In this example, even if the Nginx restart command returns non-zero, the overall function will still return an OK exit code; Consul Template will continue to run as a service. Additionally, if you have complex logic for restarting your service, you can intelligently choose when you want Consul Template to exit and when you want it to continue to watch for changes. For these types of complex scripts, we recommend using a custom sh or bash script instead of putting the logic directly in the `consul-template` command or configuration file.

### Command Environment
The current processes environment is used when executing commands with the following additional environment variables:

- `CONSUL_HTTP_ADDR`
- `CONSUL_HTTP_TOKEN`
- `CONSUL_HTTP_AUTH`
- `CONSUL_HTTP_SSL`
- `CONSUL_HTTP_SSL_VERIFY`

These environment variables are exported with their current values when the command executes. Other Consul tooling reads these environment variables, providing smooth integration with other Consul tools (like `consul maint` or `consul lock`). Additionally, exposing these environment variables gives power users the ability to further customize their command script.

### Multi-phase Execution
Consul Template does an n-pass evaluation of templates, accumulating dependencies on each pass. This is required due to nested dependencies, such as:

```liquid
{{range services}}
{{range service .Name}}
  {{.Address}}
{{end}}{{end}}
```

During the first pass, Consul Template does not know any of the services in Consul, so it has to perform a query. When those results are returned, the inner-loop is then evaluated with that result, potentially creating more queries and watches.

Because of this implementation, template functions need a default value that is an acceptable parameter to a `range` function (or similar), but does not actually execute the inner loop (which would cause a panic). This is important to mention because complex templates **must** account for the "empty" case. For example, the following **will not work**:

```liquid
{{with index (service "foo") 0}}
# ...
{{end}}
```

This will raise an error like:

```text
<index $services 0>: error calling index: index out of range: 0
```

That is because, during the _first_ evaluation of the template, the `service` key is returning an empty slice. You can account for this in your template like so:

```liquid
{{if service "foo"}}
{{with index (service "foo") 0}}
{{.Node}}
{{ end }}
{{ end }}
```

This will still add the dependency to the list of watches, but Go will not evaluate the inner-if, avoiding the out-of-index error.


Examples
--------
### HAProxy
HAProxy is a very common load balancer. You can read more about the HAProxy configuration file syntax in the HAProxy documentation, but here is an example template for rendering an HAProxy configuration file with Consul Template:

```liquid
global
    daemon
    maxconn {{key "service/haproxy/maxconn"}}

defaults
    mode {{key "service/haproxy/mode"}}{{range ls "service/haproxy/timeouts"}}
    timeout {{.Key}} {{.Value}}{{end}}

listen http-in
    bind *:8000{{range service "release.web"}}
    server {{.Node}} {{.Address}}:{{.Port}}{{end}}
```

Save this file to disk as `haproxy.ctmpl` and  run the `consul-template` daemon:

```shell
$ consul-template \
  -consul demo.consul.io \
  -template haproxy.ctmpl:/etc/haproxy/haproxy.conf
  -dry
```

Depending on the state of the demo Consul instance, you could see the following output:

```text
global
    daemon
    maxconn 4

defaults
    mode default
    timeout 5

listen http-in
    bind *:8000
    server nyc3-worker-2 104.131.109.224:80
    server nyc3-worker-3 104.131.59.59:80
    server nyc3-worker-1 104.131.86.92:80
```

For more information on how to save this result to disk or for the full list of functionality available inside a Consul template file, please consult the API documentation.

### Varnish
Varnish is an common caching engine that can also act as a proxy. You can read more about the Varnish configuration file syntax in the Varnish documentation, but here is an example template for rendering a Varnish configuration file with Consul Template:

```liquid
import directors;
{{range service "consul"}}
backend {{.Name}}_{{.ID}} {
    .host = "{{.Address}}";
    .port = "{{.Port}}";
}{{end}}

sub vcl_init {
  new bar = directors.round_robin();
{{range service "consul"}}
  bar.add_backend({{.Name}}_{{.ID}});{{end}}
}

sub vcl_recv {
  set req.backend_hint = bar.backend();
}
```

Save this file to disk as `varnish.ctmpl` and  run the `consul-template` daemon:

```shell
$ consul-template \
  -consul demo.consul.io \
  -template varnish.ctmpl:/etc/varnish/varnish.conf \
  -dry
```

You should see the following output:

```text
import directors;

backend consul_consul {
    .host = "104.131.109.106";
    .port = "8300";"
}

sub vcl_init {
  new bar = directors.round_robin();

  bar.add_backend(consul_consul);
}

sub vcl_recv {
  set req.backend_hint = bar.backend();
}
```

### Apache httpd
Apache httpd is a popular web server. You can read more about the Apache httpd configuration file syntax in the Apache httpd documentation, but here is an example template for rendering part of an Apache httpd configuration file that is responsible for configuring a reverse proxy with dynamic end points based on service tags with Consul Template:

```liquid
{{range $tag, $service := service "web" | byTag}}
# "{{$tag}}" api providers.
<Proxy balancer://{{$tag}}>
{{range $service}}  BalancerMember http://{{.Address}}:{{.Port}}
{{end}} ProxySet lbmethod=bybusyness
</Proxy>
Redirect permanent /api/{{$tag}} /api/{{$tag}}/
ProxyPass /api/{{$tag}}/ balancer://{{$tag}}/
ProxyPassReverse /api/{{$tag}}/ balancer://{{$tag}}/
{{end}}
```

Just like the previous examples, save this file to disk and run the `consul-template` daemon:

```shell
$ consul-template \
  -consul demo.consul.io \
  -template httpd.ctmpl:/etc/httpd/sites-available/balancer.conf
```

You should see output similar to the following:

```text
# "frontend" api providers.
<Proxy balancer://frontend>
  BalancerMember http://104.131.109.106:8080
  BalancerMember http://104.131.109.113:8081
  ProxySet lbmethod=bybusyness
</Proxy>
Redirect permanent /api/frontend /api/frontend/
ProxyPass /api/frontend/ balancer://frontend/
ProxyPassReverse /api/frontend/ balancer://frontend/

# "api" api providers.
<Proxy balancer://api>
  BalancerMember http://104.131.108.11:8500
  ProxySet lbmethod=bybusyness
</Proxy>
Redirect permanent /api/api /api/api/
ProxyPass /api/api/ balancer://api/
ProxyPassReverse /api/api/ balancer://api/
```

### Querying all services
As of Consul Template 0.6.0, it is possible to have a complex dependency graph with dependent services. As such, it is possible to query and watch all services in Consul:

```liquid
{{range services}}# {{.Name}}{{range service .Name}}
{{.Address}}{{end}}

{{end}}
```

Just like the previous examples, save this file to disk and run the `consul-template` daemon:

```shell
$ consul-template \
  -consul demo.consul.io \
  -template everything.ctmpl:/tmp/inventory
```

You should see output similar to the following:

```text
# consul
104.131.121.232

# redis
104.131.86.92
104.131.109.224
104.131.59.59

# web
104.131.86.92
104.131.109.224
104.131.59.59
```

Debugging
---------
Consul Template can print verbose debugging output. To set the log level for Consul Template, use the `-log-level` flag:

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


FAQ
---
**Q: How is this different than confd?**<br>
A: The answer is simple: Service Discovery as a first class citizen. You are also encouraged to read [this Pull Request](https://github.com/kelseyhightower/confd/pull/102) on the project for more background information. We think confd is a great project, but Consul Template fills a missing gap.

**Q: How is this different than Puppet/Chef/Ansible/Salt?**<br>
A: Configuration management tools are designed to be used in unison with Consul Template. Instead of rendering a stale configuration file, use your configuration management software to render a dynamic template that will be populated by [Consul][].


Contributing
------------
To build and install Consul Template locally, you will need a modern [Go][] (Go 1.5+) environment.

First, clone the repo:

```shell
$ git clone https://github.com/hashicorp/consul-template.git
```

Next, download/update all the dependencies:

```shell
$ make updatedeps
```

To compile the `consul-template` binary and run the test suite:

```shell
$ make dev
```

This will compile the `consul-template` binary into `bin/consul-template` as well as your `$GOPATH` and run the test suite.

If you just want to run the tests:

```shell
$ make test
```

Or to run a specific test in the suite:

```shell
go test ./... -run SomeTestFunction_name
```

Submit Pull Requests and Issues to the [Consul Template project on GitHub][Consul Template].


[Consul]: http://consul.io/ "Service discovery and configuration made easy"
[Releases]: https://github.com/hashicorp/consul-template/releases "Consul Template releases page"
[HCL]: https://github.com/hashicorp/hcl "HashiCorp Configuration Language (HCL)"
[Go]: http://golang.org "Go the language"
[Consul ACLs]: http://www.consul.io/docs/internals/acl.html "Consul ACLs"
[Go Template]: http://golang.org/pkg/text/template/ "Go Template"
[Consul Template]: https://github.com/hashicorp/consul-template "Consul Template on GitHub"
