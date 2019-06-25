#!/bin/bash

set -x
set -e
export VAULT_ADDR=https://vault.infra
export VAULT_SKIP_VERIFY=true
export CONSUL_HTTP_ADDR=$CONSUL_ADDR:8500
export SERVICE_ACCOUNT_TOKEN=$(cat /var/run/secrets/kubernetes.io/serviceaccount/token)
export VAULT_TOKEN=$(vault write auth/kubernetes/login role=read jwt=$SERVICE_ACCOUNT_TOKEN | grep -m 1 token | awk '{print $2}')

if [ -z "$VAULT_TOKEN" ]; then
    echo "VAULT_TOKEN not set. Exiting ..."
    exit 1
fi

exec /usr/bin/consul-template "$@"
