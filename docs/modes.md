# Consul Template Modes

Consul Template can run in different modes that changes its runtime behavior
and process lifecycle.

- [Once Mode](#once-mode)
- [De-Duplication Mode](#de-duplication-mode)
- [Exec Mode](#exec-mode)

## Once Mode

In Once mode, Consul Template will wait for all dependencies to be rendered. If
a template specifies a dependency (a request) that does not exist in Consul,
once mode will wait until Consul returns data for that dependency. Please note
that "returned data" and "empty data" are not mutually exclusive.

To run in Once mode, include the `-once` flag or enable in the
[configuration file](configuration.md#once-mode).

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

**Note:** Once mode implicitly disables any wait/quiescence timers specified in
configuration files or passed on the command line.

## De-Duplication Mode

Consul Template works by parsing templates to determine what data is needed and
then watching Consul for any changes to that data. This allows Consul Template
to efficiently re-render templates when a change occurs. However, if there are
many instances of Consul Template rendering a common template there is a linear
duplication of work as each instance is querying the same data.

To make this pattern more efficient Consul Template supports work de-duplication
across instances. This can be enabled with the `-dedup` flag or via the top level
[`deduplicate` configuration block](configuration.md#de-duplication-mode). Once enabled,
Consul Template uses leader election on a per-template basis to have only a single
node perform the queries. Results are shared among other instances rendering the
same template by passing compressed data through the Consul K/V store.

Please note that no Vault data will be stored in the compressed template.
Because ACLs around Vault are typically more closely controlled than those ACLs
around Consul's KV, Consul Template will still request the secret from Vault on
each iteration.

When running in de-duplication mode, it is important that local template
functions resolve correctly. For example, you may have a local template function
that relies on the `env` helper like this:

```hcl
{{ key (env "KEY") }}
```

It is crucial that the environment variable `KEY` in this example is consistent
across all machines engaged in de-duplicating this template. If the values are
different, Consul Template will be unable to resolve the template, and you will
not get a successful render.

## Exec Mode

As of version 0.16.0, Consul Template has the ability to maintain an arbitrary
child process (similar to [envconsul](https://github.com/hashicorp/envconsul)).
This mode is most beneficial when running Consul Template in a container or on a
scheduler like [Nomad](https://www.nomadproject.io) or Kubernetes. When
activated, Consul Template will spawn and manage the lifecycle of the child
process.

Configuration options for running Consul Template in exec mode can be found in
the [configuration documentation](configuration.md#exec-mode).

This mode is best-explained through example. Consider a simple application that
reads a configuration file from disk and spawns a server from that
configuration.

```sh
$ consul-template \
    -template "/tmp/config.ctmpl:/tmp/server.conf" \
    -exec "/bin/my-server -config /tmp/server.conf"
```

When Consul Template starts, it will pull the required dependencies and populate
the `/tmp/server.conf`, which the `my-server` binary consumes. After that
template is rendered completely the first time, Consul Template spawns and
manages a child process. When any of the list templates change, Consul Template
will send a configurable reload signal to the child process. Additionally,
Consul Template will proxy any signals it receives to the child process. This
enables a scheduler to control the lifecycle of the process and also eases the
friction of running inside a container.

A common point of confusion is that the command string behaves the same as the
shell; it does not. In the shell, when you run `foo | bar` or `foo > bar`, that
is actually running as a subprocess of your shell (bash, zsh, csh, etc.). When
Consul Template spawns the exec process, it runs outside of your shell. This
behavior is _different_ from when Consul Template executes the template-specific
reload command. If you want the ability to pipe or redirect in the exec command,
you will need to spawn the process in subshell, for example:

```hcl
exec {
  command = "/bin/bash -c 'my-server > /var/log/my-server.log'"
}
```

Note that when spawning like this, most shells do not proxy signals to their
child by default, so your child process will not receive the signals that Consul
Template sends to the shell. You can avoid this by writing a tiny shell wrapper
and executing that instead:

```bash
#!/usr/bin/env bash
trap "kill -TERM $child" SIGTERM

/bin/my-server -config /tmp/server.conf
child=$!
wait "$child"
```

Alternatively, you can use your shell's exec function directly, if it exists:

```bash
#!/usr/bin/env bash
exec /bin/my-server -config /tmp/server.conf > /var/log/my-server.log
```

There are some additional caveats with Exec Mode, which should be considered
carefully before use:

- If the child process dies, the Consul Template process will also die. Consul
  Template **does not supervise the process!** This is generally the
  responsibility of the scheduler or init system.

- The child process must remain in the foreground. This is a requirement for
  Consul Template to manage the process and send signals.

- The exec command will only start after _all_ templates have been rendered at
  least once. One may have multiple templates for a single Consul Template
  process, all of which must be rendered before the process starts. Consider
  something like an nginx or apache configuration where both the process
  configuration file and individual site configuration must be written in order
  for the service to successfully start.

- After the child process is started, any change to any dependent template will
  cause the reload signal to be sent to the child process. If no reload signal
  is provided, Consul Template will kill the process and spawn a new instance.
  The reload signal can be specified and customized via the CLI or configuration
  file.

- When Consul Template is stopped gracefully, it will send the configurable kill
  signal to the child process. The default value is SIGTERM, but it can be
  customized via the CLI or configuration file.

- Consul Template will forward all signals it receives to the child process
  **except** its defined `reload_signal` and `kill_signal`. If you disable these
  signals, Consul Template will forward them to the child process.

- It is not possible to have more than one exec command (although each template
  can still have its own reload command).

- Individual template reload commands still fire independently of the exec
  command.
