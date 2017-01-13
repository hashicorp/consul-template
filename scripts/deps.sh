#!/usr/bin/env sh
set -e

echo "--> Installing dependency manager..."
go get -u github.com/kardianos/govendor

echo "--> Removing old dependencies..."
rm -rf vendor/**/*

echo "--> Installing all dependencies..."
govendor init
govendor fetch -v +outside

echo "--> Vendoring..."
govendor add +external
