rwildcard=$(wildcard $1$2) $(foreach d,$(wildcard $1*),$(call rwildcard,$d/,$2))
SOURCES := $(call rwildcard,.,*.go)

.PHONY: all
all: build

start: start_grpc start_envoy

stop: stop_grpc

start_envoy:
	docker run --rm --name envoy -v $$PWD/envoy_config.yaml:/etc/envoy/envoy.yaml:ro --net host envoyproxy/envoy:v1.19.0 -c /etc/envoy/envoy.yaml --log-level debug

start_grpc: grpc_server
	./grpc_server -src-path '/httpbin\?redirect=true' -dst-path /httpgo 2> grpc_server.log &

stop_grpc:
	pkill grpc_server

.PHONY: grpc_server
grpc_server: $(SOURCES)
	go build -o grpc_server .

.PHONY: build
build: grpc_server

.PHONY: clean
clean:
	go clean
	rm -f grpc_server.log envoy.log
