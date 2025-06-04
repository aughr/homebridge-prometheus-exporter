#!/usr/bin/env bash

docker build . -f Dockerfile.aarc64 -t go-compile-aarc64
docker build . -f Dockerfile.armv7 -t go-compile-armv7