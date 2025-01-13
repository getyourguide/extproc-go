package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("usage: %s <url>", os.Args[0])
	}
	url := os.Args[1]

	r, err := http.Get(url)
	if err != nil {
		log.Fatalf("requesting %s: %w", url, err.Error())
	}
	defer r.Body.Close()

	log.Printf("requesting %s -> %d", url, r.StatusCode)
	if r.StatusCode >= 400 {
		os.Exit(1)
	}
}
