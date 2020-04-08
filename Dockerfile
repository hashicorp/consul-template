FROM golang:1.12 AS builder

WORKDIR /go/src/github.com/hashicorp/consul-template
COPY . . 

RUN CGO_ENABLED="0" \
    GOOS=linux \
    go build -a -o "/consul-template" \
    -ldflags "-s -w" 

FROM debian:stretch-slim AS packager

RUN apt update -y && \
    apt install wget -y && \
    apt install unzip -y 

RUN wget https://releases.hashicorp.com/vault/1.0.3/vault_1.0.3_linux_amd64.zip && \
    unzip vault_1.0.3_linux_amd64.zip && \
    mv vault /usr/bin/vault && \
    rm vault_1.0.3_linux_amd64.zip

FROM alpine:latest

COPY  --from=builder /consul-template /bin/consul-template
COPY  --from=packager /usr/bin/vault /usr/bin/vault

COPY entrypoint.sh /entrypoint.sh

RUN chmod +x /entrypoint.sh 

ENTRYPOINT ["./entrypoint.sh"]
CMD ["/bin/consul-template"]