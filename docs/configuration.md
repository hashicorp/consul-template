# Configuring Consul Template

Consul Template can be configured with command line flags and configuration
files. The CLI interface supports most options in the configuration file and
vice-versa. We suggest keeping the usage of CLI flags brief for readability
purposes, and using a configuration file for more complex and longer
configuration.

- [Command Line Flags](#command-line-flags)
- [Configuration File](#configuration-file)
- [Configuration Options](#configuration-options)
  - [Consul Template](#consul-template)
  - [Consul](#consul)
  - [Vault](#vault)
  - [Templates](#templates)
  - [Consul Template Modes](#modes)
    - [Once Mode](#once-mode)
    - [De-Duplication Mode](#de-duplication-mode)
    - [Exec Mode](#exec-mode)

## Command Line Flags

For the full list of options:

```shell
$ consul-template -h
```

Here are a few examples of common integrations on the command line.

Render the template `/tmp/template.ctmpl` to `/tmp/result` on disk:

```shell
$ consul-template \
    -template "/tmp/template.ctmpl:/tmp/result"
```

Render multiple templates in the same process. The optional third argument to
the template is a command that will execute each time the template changes.

```shell
$ consul-template \
    -template "/tmp/nginx.ctmpl:/var/nginx/nginx.conf:nginx -s reload" \
    -template "/tmp/redis.ctmpl:/var/redis/redis.conf:service redis restart" \
    -template "/tmp/haproxy.ctmpl:/var/haproxy/haproxy.conf"
```

Render a template using a custom Consul and Vault address:

```shell
$ consul-template \
    -consul-addr "10.4.4.6:8500" \
    -vault-addr "https://10.5.32.5:8200"
```

Render all templates and then spawn and monitor a child process as a supervisor:

```shell
$ consul-template \
  -template "/tmp/in.ctmpl:/tmp/result" \
  -exec "/sbin/my-server"
```

For more information on supervising, please see the
[Consul Template Exec Mode documentation](modes.md#exec-mode).

## Configuration File

Configuration files are written in the [HashiCorp Configuration Language][hcl].
By proxy, this means the configuration is also JSON compatible.

Instruct Consul Template to use a configuration file with the `-config` flag:

```shell
$ consul-template -config "/my/config.hcl"
```

This argument may be specified multiple times to load multiple configuration
files. The right-most configuration takes the highest precedence. If the path to
a directory is provided (as opposed to the path to a file), all of the files in
the given directory will be merged in
[lexical order](http://golang.org/pkg/path/filepath/#Walk), recursively. Please
note that symbolic links are _not_ followed.

**Commands specified on the CLI take precedence over a config file!**

Note that not all fields listed below are required. If you are not retrieving
secrets from Vault, you do not need to specify a Vault configuration section.
Similarly, if you are not logging to syslog, you do not need to specify a
syslog configuration.

For additional security, tokens may also be read from the environment using the
`CONSUL_TOKEN` or `VAULT_TOKEN` environment variables respectively. It is highly
recommended that you do not put your tokens in plain-text in a configuration
file.

### Example

The example HCL configuration file below connects Consul Template to a Consul
agent and renders a template with a value from [Consul KV][consul-kv]. The
output is written to a file on disk.

```hcl
consul {
  address = "127.0.0.1:8500"

  auth {
    enabled = true
    username = "test"
    password = "test"
  }
}

log_level = "warn"

template {
  contents = "{{key \"hello\"}}"
  destination = "out.txt"
  exec {
    command = "cat out.txt"
  }
}
```

# Configuration Options

This section covers the various options available to configure Consul Template,
data sources, templates, and different run time modes. For simplicity, the
configuration options are written in context of an HCL configuration file,
however they can be written in JSON or most options can be set via CLI flags.

## Consul Template

These options are top level values to configure Consul Template.
They are not required and will fallback to default values when omitted.

```hcl
# This denotes the start of the configuration section for Consul Template.
# All values contained in this section pertain to Vault.

# This is the signal to listen for to trigger a reload event. The default
# value is shown below. Setting this value to the empty string will cause CT
# to not listen for any reload signals.
reload_signal = "SIGHUP"

# This is the signal to listen for to trigger a graceful stop. The default
# value is shown below. Setting this value to the empty string will cause CT
# to not listen for any graceful stop signals.
kill_signal = "SIGINT"

# This is the maximum interval to allow "stale" data. By default, only the
# Consul leader will respond to queries; any requests to a follower will
# forward to the leader. In large clusters with many requests, this is not as
# scalable, so this option allows any follower to respond to a query, so long
# as the last-replicated data is within these bounds. Higher values result in
# less cluster load, but are more likely to have outdated data.
max_stale = "10m"

# This is amount of time in seconds to do a blocking query for.
# Many endpoints in Consul support a feature known as "blocking queries".
# A blocking query is used to wait for a potential change using long polling.
block_query_wait = "60s"

# This is the log level. This is also available as a command line flag.
# Valid options include (in order of verbosity): trace, debug, info, warn, err
log_level = "warn"

# This is the quiescence timers; it defines the minimum and maximum amount of
# time to wait for the cluster to reach a consistent state before rendering a
# template. This is useful to enable in systems that have a lot of flapping,
# because it will reduce the the number of times a template is rendered.
wait {
  min = "5s"
  max = "10s"
}
```

To enable these features, declare the values in the configuration file or
use the corresponding flags, if available.

```hcl
# This is the path to store a PID file which will contain the process ID of the
# Consul Template process. This is useful if you plan to send custom signals
# to the process.
pid_file = "/path/to/pid"

# This block defines the configuration for connecting to a syslog server for
# logging.
syslog {
  # This enables syslog logging. Specifying any other option also enables
  # syslog logging.
  enabled = true

  # This is the name of the syslog facility to log to.
  facility = "LOCAL5"
}
```

## Consul

Enable Consul Template to connect with [Consul][consul] by declaring the
`consul` block. This configures a Consul client to query values from Consul
features, like [Consul Catalog][consul-catalog] and [Consul KV][consul-kv].

```hcl
# This denotes the start of the configuration section for Consul. All values
# contained in this section pertain to Consul.
consul {
  # This block specifies the basic authentication information to pass with the
  # request. For more information on authentication, please see the Consul
  # documentation.
  auth {
    enabled  = true
    username = "test"
    password = "test"
  }

  # This is the address of the Consul agent. By default, this is
  # 127.0.0.1:8500, which is the default bind and port for a local Consul
  # agent. It is not recommended that you communicate directly with a Consul
  # server, and instead communicate with the local Consul agent. There are many
  # reasons for this, most importantly the Consul agent is able to multiplex
  # connections to the Consul server and reduce the number of open HTTP
  # connections. Additionally, it provides a "well-known" IP address for which
  # clients can connect.
  address = "127.0.0.1:8500"

  # This is a Consul Enterprise namespace to use for reading/writing. This can
  # also be set via the CONSUL_NAMESPACE environment variable.
  # BETA: this is to be considered a beta feature as it has had limited testing
  namespace = ""

  # This is the ACL token to use when connecting to Consul. If you did not
  # enable ACLs on your Consul cluster, you do not need to set this option.
  #
  # This option is also available via the environment variable CONSUL_TOKEN.
  # It is highly recommended that you do not put your token in plain-text in a
  # configuration file.
  token = ""

  # This controls the retry behavior when an error is returned from Consul.
  # Consul Template is highly fault tolerant, meaning it does not exit in the
  # face of failure. Instead, it uses exponential back-off and retry functions
  # to wait for the cluster to become available, as is customary in distributed
  # systems.
  retry {
    # This enabled retries. Retries are enabled by default, so this is
    # redundant.
    enabled = true

    # This specifies the number of attempts to make before giving up. Each
    # attempt adds the exponential backoff sleep time. Setting this to
    # zero will implement an unlimited number of retries.
    attempts = 12

    # This is the base amount of time to sleep between retry attempts. Each
    # retry sleeps for an exponent of 2 longer than this base. For 5 retries,
    # the sleep times would be: 250ms, 500ms, 1s, 2s, then 4s.
    backoff = "250ms"

    # This is the maximum amount of time to sleep between retry attempts.
    # When max_backoff is set to zero, there is no upper limit to the
    # exponential sleep between retry attempts.
    # If max_backoff is set to 10s and backoff is set to 1s, sleep times
    # would be: 1s, 2s, 4s, 8s, 10s, 10s, ...
    max_backoff = "1m"
  }

  # This block configures the SSL options for connecting to the Consul server.
  ssl {
    # This enables SSL. Specifying any option for SSL will also enable it.
    enabled = true

    # This enables SSL peer verification. The default value is "true", which
    # will check the global CA chain to make sure the given certificates are
    # valid. If you are using a self-signed certificate that you have not added
    # to the CA chain, you may want to disable SSL verification. However, please
    # understand this is a potential security vulnerability.
    verify = false

    # This is the path to the certificate to use to authenticate. If just a
    # certificate is provided, it is assumed to contain both the certificate and
    # the key to convert to an X509 certificate. If both the certificate and
    # key are specified, Consul Template will automatically combine them into an
    # X509 certificate for you.
    cert = "/path/to/client/cert"
    key  = "/path/to/client/key"

    # This is the path to the certificate authority to use as a CA. This is
    # useful for self-signed certificates or for organizations using their own
    # internal certificate authority.
    ca_cert = "/path/to/ca"

    # This is the path to a directory of PEM-encoded CA cert files. If both
    # `ca_cert` and `ca_path` is specified, `ca_cert` is preferred.
    ca_path = "path/to/certs/"

    # This sets the SNI server name to use for validation.
    server_name = "my-server.com"
  }
}
```

## Vault

Enable Consul Template to connect with [Vault][vault] by declaring the `vault`
block. This configures a Vault client to query secrets from Vault to render
secret data into templates.

```hcl
# This denotes the start of the configuration section for Vault. All values
# contained in this section pertain to Vault.
vault {
  # This is the address of the Vault leader. The protocol (http(s)) portion
  # of the address is required.
  address = "https://vault.service.consul:8200"

  # This is a Vault Enterprise namespace to use for reading/writing secrets.
  #
  # This value can also be specified via the environment variable VAULT_NAMESPACE.
  namespace = ""

  # This is the token to use when communicating with the Vault server.
  # Like other tools that integrate with Vault, Consul Template makes the
  # assumption that you provide it with a Vault token; it does not have the
  # incorporated logic to generate tokens via Vault's auth methods.
  #
  # This value can also be specified via the environment variable VAULT_TOKEN.
  # It is highly recommended that you do not put your token in plain-text in a
  # configuration file.
  #
  # When using a token from Vault Agent, the vault_agent_token_file setting
  # should be used instead, as that will take precedence over this field.
  token = ""

  # This tells Consul Template to load the Vault token from the contents of a file.
  # If this field is specified:
  # - by default Consul Template will not try to renew the Vault token, if you want it
  # to renew you will need to specify renew_token = true as below.
  # - Consul Template will periodically stat the file and update the token if it has
  # changed.
  # vault_agent_token_file = "/tmp/vault/agent/token"

  # This tells Consul Template that the provided token is actually a wrapped
  # token that should be unwrapped using Vault's cubbyhole response wrapping
  # before being used. Please see Vault's cubbyhole response wrapping
  # documentation for more information.
  unwrap_token = true

  # The default lease duration Consul Template will use on a Vault secret that 
  # does not have a lease duration. This is used to calculate the sleep duration
  # for rechecking a Vault secret value. This field is optional and will default to
  # 5 minutes.
  default_lease_duration = "60s"

  # This option tells Consul Template to automatically renew the Vault token
  # given. If you are unfamiliar with Vault's architecture, Vault requires
  # tokens be renewed at some regular interval or they will be revoked. Consul
  # Template will automatically renew the token at half the lease duration of
  # the token. The default value is true, but this option can be disabled if
  # you want to renew the Vault token using an out-of-band process.
  #
  # Note that secrets specified in a template (using {{secret}} for example)
  # are always renewed, even if this option is set to false. This option only
  # applies to the top-level Vault token itself.
  renew_token = true

  # This section details the retry options for connecting to Vault. Please see
  # the retry options in the Consul section for more information (they are the
  # same).
  retry {
    # ...
  }

  # This section details the SSL options for connecting to the Vault server.
  # Please see the SSL options in the Consul section for more information (they
  # are the same).
  ssl {
    # ...
  }
}
```

## Templates

A `template` block defines the configuration for a template. Unlike other
blocks, this block may be specified multiple times to configure multiple
templates. It is also possible to configure templates via the CLI directly.

```hcl
template {
  # This is the source file on disk to use as the input template. This is often
  # called the "Consul Template template". This option is required if not using
  # the `contents` option.
  source = "/path/on/disk/to/template.ctmpl"

  # This is the destination path on disk where the source template will render.
  # If the parent directories do not exist, Consul Template will attempt to
  # create them, unless create_dest_dirs is false.
  destination = "/path/on/disk/where/template/will/render.txt"

  # This options tells Consul Template to create the parent directories of the
  # destination path if they do not exist. The default value is true.
  create_dest_dirs = true

  # This option allows embedding the contents of a template in the configuration
  # file rather then supplying the `source` path to the template file. This is
  # useful for short templates. This option is mutually exclusive with the
  # `source` option.
  contents = "{{ keyOrDefault \"service/redis/maxconns@east-aws\" \"5\" }}"

  # This is the optional command to run when the template is rendered. The
  # command will only run if the resulting template changes. The command must
  # return within 30s (configurable), and it must have a successful exit code.
  # Consul Template is not a replacement for a process monitor or init system.
  # Please see the Commands section in the README for more.
  command = "restart service foo"

  # This is the maximum amount of time to wait for the optional command to
  # return. If you set the timeout to 0s the command is run in the background
  # without monitoring it for errors. If also using Once, consul-template can
  # exit before the command is finished. Default is 30s.
  command_timeout = "60s"

  # Exit with an error when accessing a struct or map field/key that does not
  # exist. The default behavior will print "<no value>" when accessing a field
  # that does not exist. It is highly recommended you set this to "true" when
  # retrieving secrets from Vault.
  error_on_missing_key = false

  # This is the permission to render the file. If this option is left
  # unspecified, Consul Template will attempt to match the permissions of the
  # file that already exists at the destination path. If no file exists at that
  # path, the permissions are 0644.
  perms = 0600

  # This option backs up the previously rendered template at the destination
  # path before writing a new one. It keeps exactly one backup. This option is
  # useful for preventing accidental changes to the data without having a
  # rollback strategy.
  backup = true

  # These are the delimiters to use in the template. The default is "{{" and
  # "}}", but for some templates, it may be easier to use a different delimiter
  # that does not conflict with the output file itself.
  left_delimiter  = "{{"
  right_delimiter = "}}"

  # These are functions that are not permitted in the template. If a template
  # includes one of these functions, it will exit with an error.
  function_denylist = []

  # If a sandbox path is provided, any path provided to the `file` function is
  # checked that it falls within the sandbox path. Relative paths that try to
  # traverse outside the sandbox path will exit with an error.
  sandbox_path = ""

  # This is the `minimum(:maximum)` to wait before rendering a new template to
  # disk and triggering a command, separated by a colon (`:`). If the optional
  # maximum value is omitted, it is assumed to be 4x the required minimum value.
  # This is a numeric time with a unit suffix ("5s"). There is no default value.
  # The wait value for a template takes precedence over any globally-configured
  # wait.
  wait {
    min = "2s"
    max = "10s"
  }
}
```

## Modes

Configure Consul Template to run in various modes with the following options.
Visit [documentation on modes](modes.md) for more context on each mode.

### Once Mode

Configure Consul Template to execute each template exactly once and exits with
the flag `-once` or in the configuration file.

```hcl
once = true
```

### De-Duplication Mode

This block defines the configuration for running Consul Template in
de-duplication mode. Please see the
[de-duplication mode documentation](modes.md#de-duplication-mode) for more
information on how de-duplication mode operates and the caveats of this mode.

```hcl
deduplicate {
  # This enables de-duplication mode. Specifying any other options also enables
  # de-duplication mode.
  enabled = true

  # This is the prefix to the path in Consul's KV store where de-duplication
  # templates will be pre-rendered and stored.
  prefix = "consul-template/dedup/"
}
```

### Exec Mode

This block defines the configuration for running Consul Template in exec mode.
Please see the [exec mode documentation](modes.md#exec-mode) for more information on
how exec mode operates and the caveats of this mode.

```hcl
exec {
  # This is the command to exec as a child process. There can be only one
  # command per Consul Template process.
  command = "/usr/bin/app"

  # This is a random splay to wait before killing the command. The default
  # value is 0 (no wait), but large clusters should consider setting a splay
  # value to prevent all child processes from reloading at the same time when
  # data changes occur. When this value is set to non-zero, Consul Template
  # will wait a random period of time up to the splay value before reloading
  # or killing the child process. This can be used to prevent the thundering
  # herd problem on applications that do not gracefully reload.
  splay = "5s"

  env {
    # This specifies if the child process should not inherit the parent
    # process's environment. By default, the child will have full access to the
    # environment variables of the parent. Setting this to true will send only
    # the values specified in `custom_env` to the child process.
    pristine = false

    # This specifies additional custom environment variables in the form shown
    # below to inject into the child's runtime environment. If a custom
    # environment variable shares its name with a system environment variable,
    # the custom environment variable takes precedence. Even if pristine,
    # allowlist, or denylist is specified, all values in this option
    # are given to the child process.
    custom = ["PATH=$PATH:/etc/myapp/bin"]

    # This specifies a list of environment variables to exclusively include in
    # the list of environment variables exposed to the child process. If
    # specified, only those environment variables matching the given patterns
    # are exposed to the child process. These strings are matched using Go's
    # glob function, so wildcards are permitted.
    allowlist = ["CONSUL_*"]

    # This specifies a list of environment variables to exclusively prohibit in
    # the list of environment variables exposed to the child process. If
    # specified, any environment variables matching the given patterns will not
    # be exposed to the child process, even if they are in the allowlist. The
    # values in this option take precedence over the values in the allowlist.
    # These strings are matched using Go's glob function, so wildcards are
    # permitted.
    denylist = ["VAULT_*"]
  }

  # This defines the signal that will be sent to the child process when a
  # change occurs in a watched template. The signal will only be sent after the
  # process is started, and the process will only be started after all
  # dependent templates have been rendered at least once. The default value is
  # nil, which tells Consul Template to stop the child process and spawn a new
  # one instead of sending it a signal. This is useful for legacy applications
  # or applications that cannot properly reload their configuration without a
  # full reload.
  reload_signal = ""

  # This defines the signal sent to the child process when Consul Template is
  # gracefully shutting down. The application should begin a graceful cleanup.
  # If the application does not terminate before the `kill_timeout`, it will
  # be terminated (effectively "kill -9"). The default value is "SIGINT".
  kill_signal = "SIGINT"

  # This defines the amount of time to wait for the child process to gracefully
  # terminate when Consul Template exits. After this specified time, the child
  # process will be force-killed (effectively "kill -9"). The default value is
  # "30s".
  kill_timeout = "2s"
}
```

[hcl]: https://github.com/hashicorp/hcl "HashiCorp Configuration Language (hcl)"
[consul]: https://www.consul.io "Consul by HashiCorp"
[consul-catalog]: https://www.consul.io/docs/commands/catalog.html "Consul Catalog"
[consul-kv]: https://www.consul.io/docs/agent/kv.html "Consul KV"
[vault]: https://www.vaultproject.io/ "Vault by HashiCorp"
