package main

import (
	"flag"
	"fmt"
	"time"

	wssrelayer "github.com/hewigovens/go-wssrelayer"
)

func main() {
	port := flag.Int("port", 8080, "local http port")
	timeout := flag.Int64("timeout", 4, "request timeout")
	endpoint := flag.String("endpoint", "", "remote wss endpoint to relay")
	flag.Parse()

	if endpoint == nil || *endpoint == "" {
		fmt.Println("endpoint is missing")
		fmt.Println("Usage: ./relay -port=8080 -endpoint=wss://")
		return
	}
	relayer := wssrelayer.WSSRelayer{
		Port:           *port,
		RequestTimeout: time.Duration(*timeout),
	}
	if err := relayer.ConnectAndServe(*endpoint); err != nil {
		fmt.Println(err.Error())
	}
}
