# envoy-issue-ext-proc-go
 
This repo contains an environment to reproduce an [envoy](https://envoyproxy.io) external processor bug ([#17470](https://github.com/envoyproxy/envoy/issues/17470)).

## The environment
This envoy setup contains one virtual host with two route:
- One that match the prefix `/httpbin`
- and another one that match the prefix `/httpgo`

```bash
# Httpbin response
$ curl -v http://envoy:10001/httpbin
{
  "args": {}, 
  "headers": {
    "Accept": "*/*", 
    "Host": "envoy", 
    "User-Agent": "curl/7.68.0", 
    "X-Amzn-Trace-Id": "Root=1-60fb0473-691ad9c46a89f05e63181efe", 
    "X-Envoy-Expected-Rq-Timeout-Ms": "15000", 
    "X-Envoy-Original-Path": "/httpbin"
  }, 
  "origin": "94.238.233.85", 
  "url": "https://envoy/get"
}

# Httpgo response
$ curl -v http://envoy:10001/httpgo | jq
{
  "args": {},
  "headers": {
    "Accept": [
      "*/*"
    ],
    "Fly-Client-Ip": [
      "94.238.233.85"
    ],
    "Fly-Dispatch-Start": [
      "t=1627063452513762;instance=4753e96d"
    ],
    "Fly-Forwarded-Port": [
      "80"
    ],
    "Fly-Forwarded-Proto": [
      "http"
    ],
    "Fly-Forwarded-Ssl": [
      "off"
    ],
    "Fly-Region": [
      "cdg"
    ],
    "Fly-Request-Id": [
      "01FBA8M0V1R6FN7D4GYT99239A"
    ],
    "Host": [
      "envoy:10001"
    ],
    "User-Agent": [
      "curl/7.68.0"
    ],
    "Via": [
      "1.1 fly.io"
    ],
    "X-Envoy-Expected-Rq-Timeout-Ms": [
      "15000"
    ],
    "X-Envoy-Original-Path": [
      "/httpgo"
    ],
    "X-Forwarded-For": [
      "94.238.233.85, 77.83.142.42"
    ],
    "X-Forwarded-Port": [
      "80"
    ],
    "X-Forwarded-Proto": [
      "http"
    ],
    "X-Forwarded-Ssl": [
      "off"
    ],
    "X-Request-Id": [
      "87727b23-dd05-44d0-be96-460dfff0c1f4"
    ],
    "X-Request-Start": [
      "t=1627063452513523"
    ]
  },
  "origin": "94.238.233.85",
  "url": "http://envoy:10001/get"
}
```

## The external processor

The external processor is a grpc server that alter the request path if it matches a regex. The `start_grpc` make rule
starts the grpc server and configure it to change request path that match the `/httpbin?redirect=true` regex.
.
## Reproducing the issue

```
# Start envoy and the gRPC server
$ make start

# Make a request that should be altered.
$ curl http://envoy:10001/httpbin?redirect=true
{
  "args": {}, 
  "headers": {
    "Accept": "*/*", 
    "Host": "envoy", 
    "User-Agent": "curl/7.68.0", 
    "X-Amzn-Trace-Id": "Root=1-60fb0375-34e64ab97872bada2ac74792", 
    "X-Envoy-Expected-Rq-Timeout-Ms": "15000", 
    "X-Envoy-Original-Path": "/httpgo" // The path has changed
  }, 
  "origin": "94.238.233.85", 
  "url": "https://envoy/get"
}
```
We can see that the external processor changed the path to `/httpgo` but the response is from httpbin and not httpgo.
