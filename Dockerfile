FROM golang:1.12-stretch AS base

COPY . /go/src/github.com/hashicorp/consul-template
WORKDIR /go/src/github.com/hashicorp/consul-template

RUN go install

FROM debian:stretch-slim
COPY  --from=base /go/bin/consul-template /usr/bin/consul-template


RUN apt update -y && \
    apt install wget -y && \
    apt install unzip -y

RUN wget https://releases.hashicorp.com/vault/0.11.5/vault_0.11.5_linux_amd64.zip && \
    unzip vault_0.11.5_linux_amd64.zip && \
    mv vault /usr/bin/vault && \
    rm vault_0.11.5_linux_amd64.zip
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["./entrypoint.sh"]
CMD ["/usr/bin/consul-template"]
