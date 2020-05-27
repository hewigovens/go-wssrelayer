## go-wssrelayer

A utility tool to relay json rpc request (from http post) to remote wss endpoint.

## Build

`go build -o relay cmd/relay.go`

## Usage

`./relay -port=8080 -endpoint=wss://kusama-rpc.polkadot.io/`
