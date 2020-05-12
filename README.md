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
- [Templating Language](docs/templating-language.md)
  - [API Functions](docs/templating-language.md#api-functions)
  - [Scratch](docs/templating-language.md#scratch)
  - [Helper Functions](docs/templating-language.md#helper-functions)
  - [Math Functions](docs/templating-language.md#math-functions)
- [Plugins](#plugins)
  - [Authoring Plugins](#authoring-plugins)
    - [Important Notes](#important-notes)
- [Observability](docs/observability.md)
  - [Logging](docs/observability.md#logging)
  - [Telemetry](docs/observability.md#telemetry)
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
- [Reload Configuration and Templates](#reload-configuration-and-templates)
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

## Reload Configuration and Templates

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
