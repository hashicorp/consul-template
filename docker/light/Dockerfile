FROM busybox
LABEL maintainer "Seth Vargo <seth@sethvargo.com> (@sethvargo)"

ADD "./pkg/linux_amd64/consul-template" "/"

ENTRYPOINT ["/consul-template"]
