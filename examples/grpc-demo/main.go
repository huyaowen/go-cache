package main

import (
	"log"

	_ "github.com/coderiser/go-cache/examples/grpc-demo/service/.cache-gen"
)

func main() {
	log.Println("✅ gRPC demo - cache generator working")
	log.Println("Multi-directory test passed!")
}
