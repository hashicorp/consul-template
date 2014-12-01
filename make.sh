#!/bin/bash

NAME="consul-template"

case $1 in

runbuild)
        echo "BUILDING consul-template"
        go get -d ./...
        go build -ldflags '-linkmode external -extldflags "-static" ' -o bin/$NAME
;;

*)
        echo "RUNNING BUILD IN DOCKER CONTAINER"
        docker run --rm -it -v "$(pwd)":/usr/src/myapp -w /usr/src/myapp golang:1.3.3 ./make.sh runbuild

;;

esac
