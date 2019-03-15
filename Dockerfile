FROM docker-hub.grofer.io/kube/infra/consul-template:mft
RUN wget https://releases.hashicorp.com/vault/0.11.5/vault_0.11.5_linux_amd64.zip && \
    unzip vault_0.11.5_linux_amd64.zip && \
    mv vault /usr/bin/vault && \
    rm vault_0.11.5_linux_amd64.zip
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh
ENTRYPOINT ["./entrypoint.sh"]
