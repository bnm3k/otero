package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// curl -vkL http://127.0.0.1:8081/serviceA
func serviceA(ctx context.Context, port int) {
	serverPort := fmt.Sprintf(":%d", port)
	address := fmt.Sprintf("127.0.0.1%s", serverPort)
	var mux http.ServeMux

	mux.HandleFunc("/serviceA", serviceA_HttpHandler)

	handler := otelhttp.NewHandler(
		&mux,
		"server.http",
		// If you did not set the global propagator as shown in `tracing.go`
		// then you need to provide this one
		// otelhttp.WithPropagators(propagator),
	)
	server := &http.Server{
		Addr:    serverPort,
		Handler: handler,
	}

	log := NewZerolog(ctx)
	log.Info().Str("address", address).Msg("serviceA listening")
	err := server.ListenAndServe()
	if err != nil {
		panic(err)
	}
}

// curl -vkL http://127.0.0.1:8082/serviceB
func serviceB(ctx context.Context, port int) {
	serverPort := fmt.Sprintf(":%d", port)
	address := fmt.Sprintf("127.0.0.1%s", serverPort)
	var mux http.ServeMux
	mux.HandleFunc("/serviceB", serviceB_HttpHandler)

	handler := otelhttp.NewHandler(
		&mux,
		"server.http",
		// If you did not set the global propagator as shown in `tracing.go`
		// then you need to provide this one
		// otelhttp.WithPropagators(propagator),
	)
	server := &http.Server{
		Addr:    serverPort,
		Handler: handler,
	}

	log := NewZerolog(ctx)
	log.Info().Str("address", address).Msg("serviceB listening")
	err := server.ListenAndServe()
	if err != nil {
		panic(err)
	}
}

func serviceA_HttpHandler(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer(tracerName).Start(r.Context(), "serviceA_HttpHandler")
	defer span.End()

	counter, _ := getMeter().Int64Counter(
		"service_a_called_counter",
		sdkmetric.WithDescription("how many time the serviceA handler has been called."),
	)

	counter.Add(
		ctx,
		1,
		[]sdkmetric.AddOption{
			sdkmetric.WithAttributes(
				[]attribute.KeyValue{
					attribute.String("handler_name", "serviceA_HttpHandler"),
					attribute.Int64("req_size", r.ContentLength),
				}...,
			),
		}...,
	)

	log := NewZerolog(ctx)
	log.Info().Msg("serviceA_HttpHandler called")

	// When serviceA is called, it calls serviceB over tcp network.
	// We should still be able to propagate traces over a tcp network.
	cli := &http.Client{
		Transport: otelhttp.NewTransport(
			http.DefaultTransport,
			// If you did not set the global propagator as shown in `tracing.go`
			// then you need to provide this one
			// otelhttp.WithPropagators(propagator),
		),
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost:8082/serviceB", nil)
	if err != nil {
		panic(err)
	}
	resp, err := cli.Do(req)
	if err != nil {
		panic(err)
	}
	log.Info().Int("status_code", resp.StatusCode).Msg("serviceA called serviceB")

	fmt.Fprintf(w, "hello from serviceA")
	// response header contains, `Ot-Tracer-Spanid` & `Ot-Tracer-Traceid` headers that are added by the otel propagator.
	// upstream services can then consume those.
	LogHTTPHeader(log, "request.Header serviceA", r.Header)
	LogHTTPHeader(log, "response.Header serviceA", w.Header())
}

func serviceB_HttpHandler(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer(tracerName).Start(r.Context(), "serviceB_HttpHandler")
	defer span.End()

	counter, _ := getMeter().Int64Counter(
		"serviceB_call_counter",
		sdkmetric.WithDescription("how many time the serviceB handler has been called."),
	)
	counter.Add(ctx, 1)

	log := NewZerolog(ctx)
	log.Info().Msg("serviceB_HttpHandler called")

	answer := add(ctx, 42, 1813)

	fmt.Fprintf(w, "hello from serviceB: Answer is: %d", answer)
	// response header contains, `Ot-Tracer-Spanid` & `Ot-Tracer-Traceid` headers that are added by the otel propagator.
	// upstream services can then consume those.
	LogHTTPHeader(log, "request.Header serviceB", r.Header)
	LogHTTPHeader(log, "response.Header serviceB", w.Header())
}

func add(ctx context.Context, x, y int64) int64 {
	// otel.Tracer("instrumentation/package/name", trace.WithStackTrace(true)) // can also take other opts
	ctx, span := otel.Tracer(tracerName).Start(
		ctx,
		"add",
		// add labels/tags(if any) that are specific to this scope.
		trace.WithAttributes(attribute.String("method", "GET")),
		trace.WithAttributes(attribute.String("endpoint", "/foo/user")),
	)
	defer span.End()

	err := errors.New("oops, 99 problems")
	span.RecordError(err, trace.WithStackTrace(true))

	lz := NewZerolog(ctx)
	lz.Info().Msg("zerolog: add called.")

	return x + y
}
