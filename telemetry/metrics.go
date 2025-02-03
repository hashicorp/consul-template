package telemetry

import (
	"github.com/armon/go-metrics"
)

type CounterMetric struct {
	Names       []string
	Description string
	ConstLabels []metrics.Label
}

func (m *CounterMetric) Add(val float32, labels ...metrics.Label) {
	metrics.IncrCounterWithLabels(m.Names, val, labels)
}

// Counters
var CounterDependenciesReceived = CounterMetric{
	Names:       []string{"dependencies_received"},
	ConstLabels: []metrics.Label{},
	Description: "A counter of dependencies received with labels " +
		"type=(consul|vault|local) and id=dependencyString",
}
var CounterTemplatesRendered = CounterMetric{
	Names:       []string{"templates_rendered"},
	ConstLabels: []metrics.Label{},
	Description: "A counter of templates rendered with labels " +
		"id=templateID and status=(rendered|would|quiescence)",
}

var CounterActions = CounterMetric{
	Names:       []string{"runner_actions"},
	ConstLabels: []metrics.Label{},
	Description: "A count of runner actions with labels action=(start|stop|run)",
}
var CounterCommandExecs = CounterMetric{
	Names:       []string{"commands_exec"},
	ConstLabels: []metrics.Label{},
	Description: "The number of commands executed with labels status=(success|error)",
}

func NewLabel(name string, value string) metrics.Label {
	return metrics.Label{Name: name, Value: value}
}

func InitMetrics() {
	CounterDependenciesReceived.Add(0)
	CounterTemplatesRendered.Add(0)
	CounterActions.Add(0)
	CounterCommandExecs.Add(0)
}
