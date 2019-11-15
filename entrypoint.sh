#!/bin/bash

set -x
set -e
export VAULT_ADDR=https://vault.infra
export VAULT_SKIP_VERIFY=true
export CONSUL_HTTP_ADDR=$CONSUL_ADDR:8500
export SERVICE_ACCOUNT_TOKEN=$(cat /var/run/secrets/kubernetes.io/serviceaccount/token)

export VAULT_PATH=auth/kubernetes/login
if [ $KUBERNETES_SERVICE_HOST == '172.20.0.1' ] || [ $KUBERNETES_SERVICE_HOST == '10.100.0.1' ]
then
  export VAULT_PATH=auth/eks/login
fi

if [ -z "$VAULT_TOKEN" ]; then
    echo "VAULT_TOKEN not set. Exiting ..."
    exit 1
fi

export VAULT_TOKEN=$(vault write $VAULT_PATH role=read jwt=$SERVICE_ACCOUNT_TOKEN | grep -m 1 token | awk '{print $2}')
exec /usr/bin/consul-template "$@"
