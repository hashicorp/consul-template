FROM alpine:latest
LABEL maintainer "John Eikenberry <jae@zhar.net>"

RUN apk add --no-cache ca-certificates
ADD "./pkg/linux_amd64/consul-template" "/bin/consul-template"

ENTRYPOINT ["/bin/consul-template"]
