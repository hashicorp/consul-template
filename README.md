Consul Template
===============

This project provides a convienent way to populate values from [Consul][] into the filesystem using the `consul-template` daemon.

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
| `token`     | | The [Consul API token][Consul ACLs]. |
| `config`    | | The path to a configuration file on disk, relative to the current working directory. Values specified on the CLI take precedence over values specified in the configuration file |
| `wait`      | | The `minimum(:maximum)` to wait before rendering a new template to disk and triggering a command, separated by a colon (`:`). If the optional maximum value is omitted, it is assumed to be 4x the required minimum value. |
| `dry`       | | Dump generated templates to the console. If specified, generated templates are not committed to disk and commands are not invoked. |
| `once`      | | Run Consul Template once and exit (as opposed to the default behavior of daemon). |

### Command Line
The CLI interface supports all of the options detailed above.

Query the nyc1 demo Consul instance, rendering the template on disk at `/tmp/template.ctmpl` to `/tmp/result`, running Consul Template as a service until stopped:

```shell
$ consul-template \
  -consul nyc1.demo.consul.io \
  -template "/tmp/template.ctmpl:/tmp/result"
```

Query a local Consul instance, rendering the template and restarting nginx if the template has changed, once:

```shell
$ consul-template \
  -consul 127.0.0.1:8500 \
  -template "/tmp/template.ctmpl:/var/www/nginx.conf:service nginx restart"
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

### Configuration File
The Consul Template configuration file is written in [HashiCorp Configuration Language (HCL)][HCL]. By proxy, this means the Consul Template configuration file is JSON-compatible. For more information, please see the [HCL specification][HCL].

The Configuration file syntax interface supports all of the options detailed above.

```javascript
consul = "127.0.0.1:8500"
token = "abcd1234"

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

**Commands specified on the command line take precedence over those defined in a config file!**

### Templating Language
Consul Template consumes template files in the [Go Template][] format. If you are not familiar with the syntax, we recommend reading the documentation, but it is similar in appearance to Mustache, Handlebars, or Liquid.

In addition to the [Go-provided template functions][Go Template], Consul Template exposes the following functions:

#### `service`
Query Consul for the service group(s) matching the given pattern. Services are queried using the following syntax:

```liquid
{{service "release.webapp@east-aws:8000"}}
```

The example above is querying Consul for the "webapp" service, with the "release" tag, in the "east-aws" datacenter, using port "8000". The tag, datacenter and port attributes are optional. To query all nodes in the "webapp" service (regardless of tag, datacenter, etc):

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

#### `key`
Query Consul for the value at the given key. If the key cannot be converted to a string-like value, an error will occur. Keys are queried using the following syntax:

```liquid
{{key "service/redis/maxconns@east-aws"}}
```

The example above is querying Consul for the `service/redis/maxconns` in the east-aws datacenter. If you omit the datacenter attribute, the local Consul datacenter will be queried:

```liquid
{{key "service/redis/maxconns"}}
```

The beauty of Consul is that the key-value structure is entirely up to you!

#### `keyPrefix`
Query Consul for all the key-value pairs at the given prefix. If any of the values cannot be converted to a string-like value, an error will occur. KeyPrefixes are queried using the following syntax:

```liquid
{{range keyPrefix "service/redis@east-aws"}}
{{.Key}} {{.Value}}{{end}}
```

If the Consul instance had the correct structure at `service/redis` in the east-aws datacenter, the resulting template could look like:

```text
minconns 2
maxconns 12
```

Like `key`, if you omit the datacenter attribute on `keyPrefix`, the local Consul datacenter will be queried. For more examples of the templating language, see the [Examples](#examples) section below.

### File Permission Caveats
Consul Template uses Go's file modification libraries under the hood. As a caveat, these libraries do not preserve file permissions. If you require specific file permissions on the output file, you can use the optional `command` parameter and `chmod`, for example:

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
    mode {{key "service/haproxy/mode"}}
    {{range keyPrefix "service/haproxy/timeouts"}}
    timeout {{.Key}}{{.Value}}{{end}}

listen http-in
    bind *:8000
    {{range service "release.webapp"}}
    server {{.Name}} {{.Address}}:{{.Port}}{{end}}
```

Save this file to disk as `haproxy.ctmpl` and  run the `consul-template` daemon:

```shell
$ consul-template \
  -consul nyc1.demo.consul.io:80 \
  -template haproxy.ctmpl:/etc/haproxy/haproxy.conf
  -dry
```

You should see the following output:

```text
TODO: Run this command and add the output :)
```

For more information on how to save this result to disk or for the full list of functionality available inside a Consul template file, please consult the API documentation.

### Varnish
Varnish is an common caching engine that can also act as a proxy. You can read more about the Varnish configuration file syntax in the Varnish documentation, but here is an example template for rendering a Varnish configuration file with Consul Template:

```liquid
import directors;
{{range service "consul@nyc1"}}
backend {{.Name}}_{{.ID}} {
    .host = "{{.Address}}";
    .port = "{{.Port}}";"
}{{end}}

sub vcl_init {
  new bar = directors.round_robin();
{{range service "consul@nyc1"}}
  bar.add_backend({{.Name}}_{{.ID}});{{end}}
}

sub vcl_recv {
  set req.backend_hint = bar.backend();
}
```

Save this file to disk as `varnish.ctmpl` and  run the `consul-template` daemon:

```shell
$ consul-template \
  -consul nyc1.demo.consul.io:80 \
  -template varnish.ctmpl:/etc/varnish/varnish.conf
  -dry
```

You should see the following output:

```text
TODO: Run this command and add the output :)
```


Contributing
------------
To hack on Consul Template, you will need a modern [Go][] environment. To compile the `consul-template` binary and run the test suite, simply execute:

```shell
$ make
```

This will compile the `consul-template` bintary into `bin/consul-template` and run the test suite.

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
