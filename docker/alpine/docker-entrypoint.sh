#!/bin/dumb-init /bin/sh
set -e

# Note above that we run dumb-init as PID 1 in order to reap zombie processes
# as well as forward signals to all processes in its session. Normally, sh
# wouldn't do either of these functions so we'd leak zombies as well as do
# unclean termination of all our sub-processes.

# CT_CONFIG_DIR isn't exposed as a volume but you can compose additional
# config files in there if you use this image as a base, or use CT_LOCAL_CONFIG
# below.
CT_CONFIG_DIR=/consul-template/config

# You can also set the CT_LOCAL_CONFIG environemnt variable to pass some
# Consul Template configuration JSON without having to bind any volumes.
if [ -n "$CT_LOCAL_CONFIG" ]; then
    echo "$CT_LOCAL_CONFIG" > "$CT_CONFIG_DIR/local.json"
fi

# If the user is trying to run Consul Template directly with some arguments, then
# pass them to Consul Template.
if [ "${1:0:1}" = '-' ]; then
    set -- consul-template "$@"
fi

# Look for Consul Template subcommands.
if consul-template --help "$1" 2>&1 | grep -q "consul-template $1"; then
    # We can't use the return code to check for the existence of a subcommand, so
    # we have to use grep to look for a pattern in the help output.
    set -- consul-template "$@"
fi

exec "$@"
