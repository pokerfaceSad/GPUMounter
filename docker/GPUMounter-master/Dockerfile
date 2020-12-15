FROM golang:1.14.10 as build-env
RUN go env -w GOPROXY=https://goproxy.cn,direct && \
    go env -w GOSUMDB=off && \
    go env -w CGO_ENABLED=1 && \
    go env -w GOSUMDB="sum.golang.google.cn" && \
    go env -w GO111MODULE=on

WORKDIR /GPUMounter
ENV GOPROXY https://goproxy.io,direct
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN  go build -o GPUMounter-master cmd/GPUMounter-master/main.go && chmod +x GPUMounter-master

FROM ubuntu:16.04
WORKDIR /GPUMounter
COPY --from=build-env /GPUMounter/GPUMounter-master /GPUMounter/GPUMounter-master

CMD ["/bin/bash"]
