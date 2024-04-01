#!/bin/bash

sudo mkdir /sys/fs/cgroup/init && \
for p in `cat /sys/fs/cgroup/cgroup.procs`; do echo $p | sudo tee /sys/fs/cgroup/init/cgroup.procs || true; done

dockerd &
rsyslogd -n &

go test -count=1 -timeout=30s -parallel=20 -tags="${GOTAGS}" `go list ./... | grep -v github.com/hashicorp/consul-template/watch | grep -v github.com/hashicorp/consul-template/version` ${TESTARGS}
