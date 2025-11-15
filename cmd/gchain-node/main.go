package main

import (
	"context"
	"log"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Println("gchain node starting...", ctx.Err())
}
