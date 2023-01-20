package main

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
)

func setupTracing() (*sdktrace.TracerProvider, error) {
	// There are numerous exporters to choose from.
	// see: https://github.com/open-telemetry/opentelemetry-go/tree/v1.2.0/exporters
	/*
		        Another exporter example, taken from: https://github.com/aspecto-io/opentelemetry-examples/blob/d522230db13780dfd0352ccb7ac63cf021d62108/go/tracing/jaeger.go#L11-L15

		        import "go.opentelemetry.io/otel/exporters/jaeger"
		        exporter, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint("http://localhost:14268/api/traces")))
			if err != nil {
				return nil, err
			}
	*/

	/*
		        Another exporter example, taken from: https://github.com/aspecto-io/opentelemetry-examples/blob/d522230db13780dfd0352ccb7ac63cf021d62108/go/tracing/aspecto.go#L13-L21

		         import "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
		         exp, err := otlptracegrpc.New(
		                           context.Background(),
				           otlptracegrpc.WithEndpoint("collector.aspecto.io:4317"),
				           otlptracegrpc.WithHeaders(map[string]string{
					       "Authorization": "<ADD YOUR TOKEN HERE>",
				       }))
			if err != nil {
				return nil, err
			}
	*/

	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		return nil, err
	}
	// labels/tags that are common to all traces.
	resource := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String("stdout-example"),
		semconv.ServiceVersionKey.String("0.0.1"),
		semconv.DeploymentEnvironmentKey.String("staging"),
		attribute.String("name", "komu"),
	)

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter), // use batch in prod.
		sdktrace.WithResource(resource),
		sdktrace.WithSpanProcessor(loggingSpanProcessor{}),
		// see: https://opentelemetry.io/docs/go/exporting_data/#sampling
		// In prod, you should consider using the TraceIDRatioBased sampler with the ParentBased sampler.
		// sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	/*
	   When the tracer provider is created, we need to set it as the global tracer provider:
	   This ensures that if someone uses the global tracer like;
	       ctx, span := otel.Tracer("my-telemetry-library").Start(r.Context(), "get_user_cart")
	       defer span.End()
	   Then, they will always use our provider-tracer.
	*/
	otel.SetTracerProvider(provider)

	/*
		Alternative ways of providing a propagator:
		  (a)
			propagator := ot.OT{}
			otel.SetTextMapPropagator(propagator)

		  (b)
		    import "go.opentelemetry.io/contrib/propagators/b3"
			otel.SetTextMapPropagator(
			  b3.New(b3.WithInjectEncoding(b3.B3MultipleHeader))
		    )
	*/
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	return provider, nil
}

// loggingSpanProcessor logs at end of a span.
type loggingSpanProcessor struct{}

func (c loggingSpanProcessor) OnEnd(s sdktrace.ReadOnlySpan) {
	// TODO: (komuw) merge s.Attributes() +  s.Resource() + s.Events()[maybe]
	// attrSet := attribute.NewSet(s.Attributes()...)
	// log.Println("\n\n\t onEnd called.",
	// 	"s.Name(): ", s.Name(),
	// 	"TraceID: ", s.SpanContext().TraceID(),
	// 	"SpanID: ", s.SpanContext().SpanID(),
	// 	"duration: ", s.EndTime().Sub(s.StartTime()),
	// 	"s.Attributes(): ", attrSet.Encoded(attribute.DefaultEncoder()),
	// 	"s.Resource(): ", s.Resource(),
	// 	// events is where errorStacktraces(if any) are recorded.
	// 	"s.Events(): ", s.Events(),
	// )
}

func (c loggingSpanProcessor) OnStart(parent context.Context, s sdktrace.ReadWriteSpan) {
	// TODO: also maybe log at the start??
}
func (c loggingSpanProcessor) ForceFlush(ctx context.Context) error { return nil }
func (c loggingSpanProcessor) Shutdown(ctx context.Context) error   { return nil }
