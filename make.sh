#!/bin/bash

NAME="consul-template"

case $1 in

runbuild)
        echo "BUILDING consul-template"
        # CGO_ENABLED=0 godep go build -a -ldflags '-s' -o run/enterpriseManager
        go get -d ./...
        go build -ldflags '-linkmode external -extldflags "-static" ' -o bin/$NAME
;;

*)
        echo "RUNNING BUILD IN DOCKER CONTAINER"
        docker run --rm -it -v "$(pwd)":/usr/src/myapp -w /usr/src/myapp golang:1.3.3 ./make.sh runbuild

        # cd run
        # docker build --tag ionic/consultemplate .
;;

esac
