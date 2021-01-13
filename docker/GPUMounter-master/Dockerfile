FROM golang:1.14.10 as build-env
RUN go env -w GOPROXY=https://goproxy.io,direct && \
    go env -w CGO_ENABLED=1 && \
    go env -w GOSUMDB="sum.golang.google.cn"

WORKDIR /GPUMounter
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN  go build -o GPUMounter-master cmd/GPUMounter-master/main.go && chmod +x GPUMounter-master

FROM ubuntu:18.04
WORKDIR /GPUMounter
COPY --from=build-env /GPUMounter/GPUMounter-master /GPUMounter/GPUMounter-master

CMD ["/bin/bash"]
