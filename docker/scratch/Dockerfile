FROM scratch
LABEL maintainer "John Eikenberry <jae@zhar.net>"

ADD "https://curl.haxx.se/ca/cacert.pem" "/etc/ssl/certs/ca-certificates.crt"
ADD "./pkg/linux_amd64/consul-template" "/"
ENTRYPOINT ["/consul-template"]
