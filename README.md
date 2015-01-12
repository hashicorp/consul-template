Consul Template
===============
[![Latest Version](http://img.shields.io/github/release/hashicorp/consul-template.svg?style=flat-square)][release]
[![Build Status](http://img.shields.io/travis/hashicorp/consul-template.svg?style=flat-square)][travis]
[![Go Documentation](http://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)][godocs]

[release]: https://github.com/hashicorp/consul-template/releases
[travis]: http://travis-ci.org/hashicorp/consul-template
[godocs]: http://godoc.org/github.com/hashicorp/consul-template

This project provides a convenient way to populate values from [Consul][] into the filesystem using the `consul-template` daemon.

The daemon `consul-template` queries a [Consul][] instance and updates any number of specified templates on the filesystem. As an added bonus, `consul-template` can optionally run arbitrary commands when the update process completes. See the [Examples](#examples) section for some scenarios were this functionality might prove useful.


Installation
------------
You can download a released `consul-template` artifact from [the Consul Template release page][Releases] on GitHub. If you wish to compile from source, you will need to have buildtools and [Go][] installed:

```shell
$ git clone https://github.com/hashicorp/consul-template.git
$ cd consul-template
$ make
```

This process will create `bin/consul-template` which make be invoked as a binary.


Usage
-----
### Options
| Option | Required | Description |
| ------ | -------- |------------ |
| `consul`    | _(required)_ | The location of the Consul instance to query (may be an IP address or FQDN) with port. |
| `template`  | _(required)_ | The input template, output path, and optional command separated by a colon (`:`). This option is additive and may be specified multiple times for multiple templates. |
| `ssl`       | | Use HTTPS while talking to Consul. Requires the Consul server to be configured to serve secure connections.
| `ssl_no_verify` | | Ignore certificate warnings. Only used if `ssl` is enabled.
| `token`     | | The [Consul API token][Consul ACLs]. |
| `config`    | | The path to a configuration file or directory on disk, relative to the current working directory. Values specified on the CLI take precedence over values specified in the configuration file |
| `wait`      | | The `minimum(:maximum)` to wait before rendering a new template to disk and triggering a command, separated by a colon (`:`). If the optional maximum value is omitted, it is assumed to be 4x the required minimum value. |
| `retry`     | | The amount of time to wait if Consul returns an error when communicating with the API. |
| `dry`       | | Dump generated templates to the console. If specified, generated templates are not committed to disk and commands are not invoked. _(CLI-only)_ |
| `once`      | | Run Consul Template once and exit (as opposed to the default behavior of daemon). _(CLI-only)_ |

### Command Line
The CLI interface supports all of the options detailed above.

Query the nyc1 demo Consul instance, rendering the template on disk at `/tmp/template.ctmpl` to `/tmp/result`, running Consul Template as a service until stopped:

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

### Configuration File(s)
The Consul Template configuration files are written in [HashiCorp Configuration Language (HCL)][HCL]. By proxy, this means the Consul Template configuration file is JSON-compatible. For more information, please see the [HCL specification][HCL].

The Configuration file syntax interface supports all of the options detailed above, unless otherwise noted in the table.

```javascript
consul = "127.0.0.1:8500"
token = "abcd1234"
retry = "10s"

template {
  source = "/path/on/disk/to/template"
  destination = "/path/on/disk/where/template/will/render"
  command = "optional command to run when the template is updated"
}

template {
  // Multiple definitions are supported
}
```

Query the nyc1 demo Consul instance, rendering the template on disk at `/tmp/template.ctmpl` to `/tmp/result`, running Consul Template as a service until stopped:

```javascript
consul = "nyc1.demo.consul.io"

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

##### `file`
Read and render the contents of a local file on disk. If the file cannot be read, an error will occur. Files are read using the following syntax:

```liquid
{{file "/path/to/local/file"}}
```

This example will render the entire contents of the file at `/path/to/local/file` into the template.

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
{{service "release.webapp@east-aws:8000"}}
```

The example above is querying Consul for healthy "webapp" services, with the "release" tag, in the "east-aws" datacenter, using port "8000". The tag, datacenter and port attributes are optional. To query all nodes of the "webapp" service (regardless of tag and port) for the current datacenter:

```liquid
{{service "webapp"}}
```

The function returns a `[]*Service` struct which can be used for ranging in a template:

```liquid
{{range service "webapp@datacenter"}}
server {{.Name}} {{.Address}}:{{.Port}}{{end}}
```

which would produce something like:

```text
server nyc_web_01 123.456.789.10:8080
server nyc_web_02 456.789.101.213:8080
```

By default only healthy services are returned.
If you want to get all services, in specific healths, then you can specify a comma-separated list of health check statuses.
Currently supported are `"any"`, `"passing"`, `"warning"` and `"critical"`.

```liquid
{{service "webapp" "any"}}
{{service "webapp" "passing"}}
{{service "webapp" "passing, warning, critical"}}
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

- - -

#### Helper Functions

##### `byTag`
Takes the list of services returned by the [`service`](#service) function and creates a map that groups services by tag.

```liquid
{{range $tag, $services := service "webapp" | byTag}}{{$tag}}
{{range $services}} server {{.Name}} {{.Address}}:{{.Port}}
{{end}}{{end}}
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

##### `parseJSON`
Takes the given input (usually the value from a key) and parses the result as JSON:

```liquid
{{with $d := key "user/info" | parseJSON}}{{$d.name}}{{end}}
```

Alternatively you can read data from a local JSON file:

```liquid
{{with $d := file "/path/to/local/data.json" | parseJSON}}{{$d.some_key}}{{end}}
```

##### `regexReplaceAll`
Takes the argument as a regular expression and replaces all occurences of the regex with the given string. As in go, you can use variables like $1 to refer to subexpressions in the replacement string.

```liquid
{{"foo.bar" | regexReplaceAll "foo([.a-z]+)", "$1"}}
```

##### `replaceAll`
Takes the argument as a string and replaces all occurences of the given string with the given string.

```liquid
{{"foo.bar" | replaceAll ".", "_"}}
```

This function can be chained with other functions as well:

```liquid
{{service "webapp"}}{{.Name | replaceAll ":", "_"}}{{end}}
```

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

### File Permissions
Consul Template uses Go's file modification libraries under the hood. If a file at the destination path already exists, Consul Template will do its best to preserve the existing file permissions. For non-existent files, Go will default to the system default. If you require specific file permissions on the output file, you can use the optional `command` parameter and `chmod`, for example:

```bash
consul-template \
  -template "/tmp/nginx.ctmpl:/var/nginx/nginx.conf:chmod 644 /var/nginx/nginx.conf && sudo restart nginx"
```

```javascript
template {
  source = "/tmp/nginx.ctmpl"
  destination = "/var/nginx/nginx.conf"
  command = "chmod 644 /var/nginx/nginx.conf && sudo restart nginx"
}
```


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
    bind *:8000{{range service "release.webapp"}}
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
    .port = "{{.Port}}";"
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
{{range $t, $s := service "web" | byTag}}
# "{{$t}}" api providers.
<Proxy balancer://{{$t}}>
{{range $s}}  BalancerMember http://{{.Address}}:{{.Port}}
{{end}} ProxySet lbmethod=bybusyness
</Proxy>
Redirect permanent /api/{{$t}} /api/{{$t}}/
ProxyPass /api/{{$t}}/ balancer://{{$t}}/
ProxyPassReverse /api/{{$t}}/ balancer://{{$t}}/
{{end}}
```

Just like the previous examples, save this file to disk and run the `consul-template` daemon:

```shell
$ consul-template \
  -consul <YOUR.CONSUL.ADDRESS> \
  -template httpd.ctmpl:/etc/httpd/sites-available/balancer.conf
```

Debugging
---------
Consul Template can print verbose debugging output. To set the log level for Consul Template, use the `CONSUL_TEMPLATE_LOG` environment variable:

```shell
$ CONSUL_TEMPLATE_LOG=info consul-template ...
```

```text
<timestamp> [INFO] (cli) received redis from Watcher
<timestamp> [INFO] (cli) invoking Runner
# ...
```

You can also specify the level as debug:

```shell
$ CONSUL_TEMPLATE_LOG=debug consul-template ...
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
A: The answer is simple: Service Discovery as a first class citizen. You are
also encouraged to read [this Pull Request](https://github.com/kelseyhightower/confd/pull/102) on the project for more background information. We think confd is a
great project, but Consul Template fills a missing gap.

**Q: How is this different than Puppet/Chef/Ansible/Salt?**<br>
A: Configuration management tools are designed to be used in unison with Consul
Template. Instead of rendering a stale configuration file, use your
configuration management software to render a dynamic template that will be
populated by [Consul][].


Contributing
------------
To hack on Consul Template, you will need a modern [Go][] environment. To compile the `consul-template` binary and run the test suite, simply execute:

```shell
$ make
```

This will compile the `consul-template` binary into `bin/consul-template` and run the test suite.

If you just want to run the tests:

```shell
$ make
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
