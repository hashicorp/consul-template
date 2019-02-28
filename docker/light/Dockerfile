FROM alpine:latest
LABEL maintainer "Seth Vargo <seth@sethvargo.com> (@sethvargo)"

ADD "./pkg/linux_amd64/consul-template" "/bin/consul-template"

ENTRYPOINT ["/bin/consul-template"]
