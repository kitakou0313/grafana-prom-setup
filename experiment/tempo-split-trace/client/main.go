package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

// client sends a plain, uninstrumented HTTP request to service-a. It creates
// no span and propagates no trace context, so the trace starts at service-a.
func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	serviceAURL := getenv("SERVICE_A_URL", "http://localhost:8080/handle")

	resp, err := http.Get(serviceAURL)
	if err != nil {
		log.Fatalf("request to service-a failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("failed to read response: %v", err)
	}

	fmt.Print(string(body))
}
