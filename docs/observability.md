# Observability Options

## Logging

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
