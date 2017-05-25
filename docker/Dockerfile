FROM scratch
LABEL maintainer "Seth Vargo <seth@sethvargo.com>"

ADD "./pkg/linux_amd64/consul-template" "/"
ENTRYPOINT ["/consul-template"]
