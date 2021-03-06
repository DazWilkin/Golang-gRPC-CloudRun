# Golang gRPC Cloud Run Golang

See:

+ https://cloud.google.com/blog/products/compute/serve-cloud-run-requests-with-grpc-not-just-http
+ https://github.com/grpc-ecosystem/grpc-cloud-run-example


## Environment

```bash
WORKDIR="${HOME}/Projects/Golang-gRPC-Cloud-Run"
mkdir ${WORKDIR}
cd ${WORKDIR}
go mod init github.com/$(whoami}/Golang-gRPC-Cloud-Run

```

## Protoc

Used `protoc` 3.11.4:

https://github.com/protocolbuffers/protobuf/releases/download/v3.11.4/protoc-3.11.4-linux-x86_64.zip

```bash
PATH=${PATH}:${PWD}/protoc/bin
```

If not already:

```bash
go get github.com/golang/protobuf/protoc-gen-go
```

Generated Golang protobuf stubs:

```bash
protoc \
--proto_path=. \
--go_out=plugins=grpc:. \
./protos/*.proto
```

Should result in `./protos/calculator.pb`

## Build

### Project

```bash
BILLING=...
PROJECT="golang-grpc-cloudrun"
gcloud projects create ${PROJECT}
gcloud beta billing projects link ${PROJECT} --billing-account=${BILLING}
gcloud services enable cloudbuild.googleapis.com --project=${PROJECT}
```


### Cloud Build

```bash
PROJECT="golang-grpc-cloudrun"
TAG=$(git rev-parse HEAD) && echo ${TAG}
gcloud builds submit . \
--config=./cloudbuild.yaml \
--substitutions=COMMIT_SHA=${TAG} \
--project=${PROJECT}
```

## Run

### Docker Compose

Includes:

+ `grpc-cloudrun-server`
+ `grpc-cloudrun-client`
+ `cadvisor`
+ `opencensus-agent`
+ `prometheus`


```bash
TAG=$(git rev-parse HEAD)
PROJECT=${PROJECT}
docker-compose up
```

### Docker Compose & Cloud Run

```bash
gcloud services enable run.googleapis.com --project=${PROJECT}
```

and:

```bash
PROJECT=${PROJECT}
TAG=$(git rev-parse HEAD)
gcloud run deploy grpc-cloudrun-server \
--image=gcr.io/${PROJECT}/server:${TAG} \
--allow-unauthenticated \
--platform=managed \
--project=${PROJECT} \
--region=us-west1
```


### Standalone

##### Server

```bash
go run server/*.go
2020/03/19 14:14:26 Starting: grpc-cloudrun-server
2020/03/19 14:14:26 Starting: OpenCensus Agent exporter []
2020/03/19 14:14:26 Starting: zPages Listener [:9998]
2020/03/19 14:14:26 Starting gRPC Listener [:50051]
2020/03/19 14:14:26 zPages RPC Stats :9998/debug/rpcz
2020/03/19 14:14:26 zPages Trace Spans :9998/debug/tracez
2020/03/19 14:14:30 [server:Calculate] Started
2020/03/19 14:14:45 [server:Calculate] Started
2020/03/19 14:15:00 [server:Calculate] Started
```

#### Client

```bash
go run client/*.go
2020/03/19 14:14:30 [main] Starting: grpc-cloudrun-server
2020/03/19 14:14:30 Starting: OpenCensus Agent exporter []
2020/03/19 14:14:30 Starting zPages Listener [:9997]
2020/03/19 14:14:30 zPages RPC Stats :9997/deubg/rpcz
2020/03/19 14:14:30 zPages Trace Spans :9997/debug/tracez
2020/03/19 14:14:30 Connecting to gRPC Service [:50051]
2020/03/19 14:14:30 [Calculate] Latency: 1.069814
2020/03/19 14:14:30 [main] 0.850+0.574=1.424
2020/03/19 14:14:45 [Calculate] Latency: 0.606172
2020/03/19 14:14:45 [main] 0.414+0.059=0.473
2020/03/19 14:15:00 [Calculate] Latency: 0.523363
2020/03/19 14:15:00 [main] 0.152+0.753=0.905
```

## Test

Docker Compose exposes the server on `:52051`

```bash
grpcurl \
-plaintext \
-proto protos/calculator.proto \
-d '{"first_operand": 2.0, "second_operand": 3.0, "operation": "ADD"}' \
localhost:52051 \
Calculator.Calculate
{
  "result": 5
}
```

## Monitor

The Docker Compose includes a configured-Prometheus [endpoint](http://localhost:9090)

## Debug

### zPages

zPages endpoints are exposed as ports on the host:

|RPCZ|TraceZ|
|:--:|:----:|
|[X](http://localhost:9995/debug/rpcz)|[X](http://localhost:9995/debug/tracez)|
|[X](http://localhost:9996/debug/rpcz)|[X](http://localhost:9996/debug/tracez)|