package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	var (
		timeout time.Duration
		url     string
	)

	flag.DurationVar(&timeout, "timeout", 0, "timeout for get requests made by the healthcheck")
	flag.StringVar(&url, "url", "", "url for requests made by the healthcheck")
	flag.Parse()

	if timeout == 0 {
		timeout = 5 * time.Second
	}
	if url == "" {
		log.Fatal("--url is a required argument")
	}

	client := http.Client{Timeout: timeout}
	r, err := client.Get(url)
	if err != nil {
		log.Fatalf("requesting %s: %s", url, err.Error())
	}
	defer r.Body.Close()

	log.Printf("GET %s (%d)", url, r.StatusCode)
	if r.StatusCode >= 400 {
		os.Exit(1)
	}
}
