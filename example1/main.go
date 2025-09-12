package main

import (
	"log"
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func main() {
	// Initialize OpenTelemetry (traces + metrics).
	shutdown, err := setupOTelSDK()
	if err != nil {
		log.Fatalf("failed to set up OpenTelemetry: %v", err)
	}
	defer shutdown() // flush exporters on exit

	// Basic HTTP server with OTel instrumentation.
	mux := http.NewServeMux()

	// Add routes and attach route tags for tracing.
	handleFunc := func(pattern string, handlerFunc func(http.ResponseWriter, *http.Request)) {
		handler := otelhttp.WithRouteTag(pattern, http.HandlerFunc(handlerFunc))
		mux.Handle(pattern, handler)
	}
	handleFunc("/rolldice/", rolldice)
	handleFunc("/rolldice/{player}", rolldice)

	// Wrap the entire mux so incoming requests create spans/metrics.
	log.Println("Serving on :8080")
	log.Fatal(http.ListenAndServe(":8080", otelhttp.NewHandler(mux, "/")))
}
