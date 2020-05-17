package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/hewigovens/wsrelayer"
)

func main() {
	port := flag.Int("port", 8080, "local http port")
	timeout := flag.Int64("timeout", 3, "request timeout")
	endpoint := flag.String("endpoint", "", "remote wss endpoint to relay")
	flag.Parse()

	if endpoint == nil || *endpoint == "" {
		fmt.Println("endpoint is missing")
		fmt.Println("Usage: ./relay -port=8080 -endpoint=wss://")
		return
	}
	relayer := wsrelayer.WSRelayer{
		Port:           *port,
		RequestTimeout: time.Duration(*timeout),
	}
	if err := relayer.ConnectAndServe(*endpoint); err != nil {
		fmt.Println(err.Error())
	}
}
