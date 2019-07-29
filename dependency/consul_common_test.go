package dependency

var filtered_meta = []string{"raft_version", "serf_protocol_current",
	"serf_protocol_min", "serf_protocol_max", "version",
}

// filterVersionMeta filters out all version information from the returned
// metadata. It allocates the meta map if it is nil to make the tests backward
// compatible with versions < 1.5.2. Once this is no longer needed the returned
// value can be removed (along with its assignment).
func filterVersionMeta(meta map[string]string) map[string]string {
	if meta == nil {
		return make(map[string]string)
	}
	for _, k := range filtered_meta {
		delete(meta, k)
	}
	return meta
}
