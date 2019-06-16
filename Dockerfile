#
# Goyamp Dockerfile
#
# USAGE
##
# Run one of the Examples
#
# $ docker run --rm -u $(id -u):$(id -g) -v "$PWD":/work docker.io/birchb1024/goyamp:0.0.2 examples/argv.yaml
#
# Run a file
#
# $ docker run --rm -u $(id -u):$(id -g) -v "$PWD":/work docker.io/birchb1024/goyamp:0.0.2 /work/path/to/your.yaml
# 
# Run from stdin:
#
#   $ echo 'argv' | docker run -i --rm docker.io/birchb1024/goyamp:0.0.2 - Hello World
#
# Debug in the container: (debug on stderr)
#
# $ docker run -it --rm -u $(id -u):$(id -g) -v "$PWD":/work --entrypoint=/bin/bash docker.io/birchb1024/goyamp:0.0.2 
#
# BUILD:
#
# $ git clone https://github.com/birchb1024/goyamp.git
# $ cd goyamp
# $ docker build -t goyamp .
#
#
FROM golang:1.12-stretch as builder
#
# Build
#
RUN mkdir -p /build
ADD . /build
WORKDIR /build
RUN go get gopkg.in/yaml.v3 
RUN go build -o goyamp
#
# Install the application
#
FROM debian:stretch-slim
RUN addgroup --gid 1000 goyampers
RUN adduser --home /goyamp --uid 1000 --gid 1000 --gecos "" goyamper --disabled-password
COPY --from=builder /build/goyamp /goyamp/bin/
ADD examples /goyamp/examples/
ADD test /goyamp/test/

RUN mkdir -p /goyamp/bin /work
USER goyamper
WORKDIR /work
ENTRYPOINT ["/goyamp/bin/goyamp"] 
