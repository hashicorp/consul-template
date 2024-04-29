ARG CONSUL_IMAGE_NAME=hashicorp/consul
ARG CONSUL_IMAGE_VERSION=1.17.1

ARG VAULT_IMAGE_NAME=hashicorp/vault
ARG VAULT_IMAGE_VERSION=1.16

ARG NOMAD_IMAGE_NAME=hashicorp/nomad
ARG NOMAD_IMAGE_VERSION=1.7

FROM ${CONSUL_IMAGE_NAME}:${CONSUL_IMAGE_VERSION} as consul
FROM ${VAULT_IMAGE_NAME}:${VAULT_IMAGE_VERSION} as vault
FROM ${NOMAD_IMAGE_NAME}:${NOMAD_IMAGE_VERSION} as nomad
FROM ubuntu:22.04

ARG CONSUL_TEMPLATE_TAG=v0.37.4
ARG GO_VERSION=1.22.1
ARG OS_ARCH=linux-amd64

RUN apt-get update -y -q && apt-get upgrade -y -q && \
    apt-get install tar git wget make rsyslog sudo curl -y && \
    curl -sSL https://get.docker.com/ | sh

RUN  rm -rf /usr/local/go && wget https://go.dev/dl/go${GO_VERSION}.${OS_ARCH}.tar.gz && tar -C /usr/local -xzf go${GO_VERSION}.${OS_ARCH}.tar.gz

RUN git clone https://github.com/hashicorp/consul-template.git
RUN cd consul-template && git fetch --all --tags && git checkout tags/${CONSUL_TEMPLATE_TAG} -b ${CONSUL_TEMPLATE_TAG}-branch

COPY --from=consul /bin/consul /bin/consul
COPY --from=vault /bin/vault /bin/vault
COPY --from=nomad /bin/nomad /bin/nomad

ADD rsyslog.conf /etc/rsyslog.conf
ADD run_tests.sh /run_tests.sh
#ADD run_nomad.sh /run_nomad.sh

RUN chmod +x /run_tests.sh
RUN chmod +x /run_nomad.sh
RUN sed -i '/imklog/s/^/#/' /etc/rsyslog.conf

WORKDIR /consul-template

ENV PATH="${PATH}:/bin:/usr/local/go/bin:/usr/local/go/src"
ENV GOPRIVATE="github.com/hashicorp/*"

CMD ["/run_tests.sh"]
