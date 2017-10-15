#!/bin/dumb-init /bin/sh
set -e

# Note above that we run dumb-init as PID 1 in order to reap zombie processes
# as well as forward signals to all processes in its session. Normally, sh
# wouldn't do either of these functions so we'd leak zombies as well as do
# unclean termination of all our sub-processes.

# CONSUL_DATA_DIR is exposed as a volume for possible persistent storage.
# CT_CONFIG_DIR isn't exposed as a volume but you can compose additional config
# files in there if you use this image as a base, or use CT_LOCAL_CONFIG below.
CT_DATA_DIR=/consul-template/config
CT_CONFIG_DIR=/consul-template/config

# You can also set the CT_LOCAL_CONFIG environment variable to pass some
# Consul Template configuration JSON without having to bind any volumes.
if [ -n "$CT_LOCAL_CONFIG" ]; then
  echo "$CT_LOCAL_CONFIG" > "$CT_CONFIG_DIR/local-config.hcl"
fi

# If the user is trying to run consul-template directly with some arguments, then
# pass them to consul-template.
if [ "${1:0:1}" = '-' ]; then
  set -- /bin/consul-template "$@"
fi

# If we are running Consul, make sure it executes as the proper user.
if [ "$1" = '/bin/consul-template' ]; then
  # If the data or config dirs are bind mounted then chown them.
  # Note: This checks for root ownership as that's the most common case.
  if [ "$(stat -c %u /consul-template/data)" != "$(id -u consul-template)" ]; then
    chown consul-template:consul-template /consul-template/data
  fi
  if [ "$(stat -c %u /consul-template/config)" != "$(id -u consul-template)" ]; then
    chown consul-template:consul-template /consul-template/config
  fi

  # Set the configuration directory
  shift
  set -- /bin/consul-template \
    -config="$CT_CONFIG_DIR" \
    "$@"

  # Run under the right user
  set -- gosu consul-template "$@"
fi

exec "$@"
