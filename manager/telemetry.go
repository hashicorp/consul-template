package manager

import (
	"context"

	dep "github.com/hashicorp/consul-template/dependency"
	"github.com/hashicorp/consul-template/telemetry"
	"go.opentelemetry.io/otel/api/metric"
)

// instruments manages the various metric instruments for monitoring the runner
type instruments struct {
	// measureDependencies measures the number of dependencies grouped by type.
	measureDependencies metric.Int64Measure
	// counterDependenciesReceived is a counter of the dependencies received from
	// monitoring clients.
	counterDependenciesReceived metric.Int64Counter
	// measureTemplates measures the number of templates configured. We could use
	// Observer here instead, but since this value is only updated during reload
	// the overhead of Observe is not worthwhile to safely setup with callback
	// function and locks around every usage of Runner.templates.
	measureTemplates metric.Int64Measure
	// counterTemplatesRendered is a counter for the number of templates rendered
	// during a run that did render, would render, or is quiescence.
	counterTemplatesRendered metric.Int64Counter
	// counterActions is a counter of the actions a runner makes. This tracks
	// restarts and runs.
	counterActions metric.Int64Counter
	// counterCommandExecs is a counter for the number of commands executed on
	// each run across all templates.
	counterCommandExecs metric.Int64Counter
	// measureCommandExecTime measures the duration a command is executed until
	// the command exits successfully, is stopped by an error, or is killed.
	measureCommandExecTime metric.Float64Measure
}

func newInstruments(meter metric.Meter) (*instruments, error) {
	if meter == nil {
		meter = telemetry.GlobalMeter()
	}

	deps, err := meter.NewInt64Measure("dependencies",
		metric.WithDescription("The number of dependencies grouped by types "+
			"with labels type=(consul|vault|local)"))
	if err != nil {
		return nil, err
	}

	depsRecv, err := meter.NewInt64Counter("dependencies_received",
		metric.WithDescription("A counter of dependencies received with label "+
			"id=dependencyString"))
	if err != nil {
		return nil, err
	}

	measureTmpls, err := meter.NewInt64Measure("templates",
		metric.WithDescription("The number of templates configured."))
	if err != nil {
		return nil, err
	}

	renderedTmpls, err := meter.NewInt64Counter("templates_rendered",
		metric.WithDescription("A counter of templates rendered with labels "+
			"id=templateID and render=(did|would|quiescence)"))
	if err != nil {
		return nil, err
	}

	actions, err := meter.NewInt64Counter("runner_actions", metric.WithDescription(
		"A count of runner actions with labels action=(start|stop|run)"))
	if err != nil {
		return nil, err
	}

	cmdExecs, err := meter.NewInt64Counter("commands_exec", metric.WithDescription(
		"The number of commands executed with labels status=(success|error)"))
	if err != nil {
		return nil, err
	}

	cmdExecTime, err := meter.NewFloat64Measure("commands_exec_time",
		metric.WithDescription("The execution time (seconds) of a template command. "+
			"The template destination is used as the identifier"))
	if err != nil {
		return nil, err
	}

	return &instruments{
		measureDependencies:         deps,
		counterDependenciesReceived: depsRecv,
		measureTemplates:            measureTmpls,
		counterTemplatesRendered:    renderedTmpls,
		counterActions:              actions,
		counterCommandExecs:         cmdExecs,
		measureCommandExecTime:      cmdExecTime,
	}, nil
}

// recordDependencyCounts is a helper to report on dependencies grouped by type
func (i *instruments) recordDependencyCounts(deps map[string]dep.Dependency) {
	types := make(map[dep.Type]int64)
	for _, dep := range deps {
		types[dep.Type()]++
	}

	for t, count := range types {
		i.measureDependencies.Record(context.Background(), count,
			telemetry.NewLabel("type", t.String()))
	}
}
