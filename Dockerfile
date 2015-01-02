FROM flynn/busybox

ADD ./bin/consul-template /bin/consul-template

ENTRYPOINT ["consul-template"]
