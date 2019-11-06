package main

import (
	"fmt"
	"log"

	"github.com/valyala/fasthttp"
)

func main() {
	if err := fasthttp.ListenAndServe(":8080", requestHandler); err != nil {
		log.Fatalf("Error in ListenAndServe: %s", err)
	}
}

func requestHandler(ctx *fasthttp.RequestCtx) {
	fmt.Fprintf(ctx, "Dantini")
}
