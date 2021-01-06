FROM golang:1.14.10 as build-env
RUN go env -w GOPROXY=https://goproxy.cn,direct && \
    go env -w CGO_ENABLED=1 && \
    go env -w GOSUMDB="sum.golang.google.cn" && \

WORKDIR /GPUMounter
COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY lib/nvml.h /usr/include/nvml.h
RUN  go build -o GPUMounter-worker cmd/GPUMounter-worker/main.go && chmod +x GPUMounter-worker

FROM ubuntu:18.04
WORKDIR /GPUMounter
COPY --from=build-env /GPUMounter/GPUMounter-worker /GPUMounter/GPUMounter-worker
ENV NVIDIA_VISIBLE_DEVICES=all

CMD ["/bin/bash"]