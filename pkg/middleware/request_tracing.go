package middleware

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/grafana/grafana/pkg/infra/tracing"
	"github.com/grafana/grafana/pkg/web"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type contextKey struct{}

var routeOperationNameKey = contextKey{}

// ProvideRouteOperationName creates a named middleware responsible for populating
// the context with the route operation name that can be used later in the request pipeline.
// Implements routing.RegisterNamedMiddleware.
func ProvideRouteOperationName(name string) web.Handler {
	return func(res http.ResponseWriter, req *http.Request, c *web.Context) {
		ctx := context.WithValue(c.Req.Context(), routeOperationNameKey, name)
		c.Req = c.Req.WithContext(ctx)
	}
}

var unnamedHandlers = []struct {
	pathPattern *regexp.Regexp
	handler     string
}{
	{handler: "public-assets", pathPattern: regexp.MustCompile("^/favicon.ico")},
	{handler: "public-assets", pathPattern: regexp.MustCompile("^/public/")},
	{handler: "/metrics", pathPattern: regexp.MustCompile("^/metrics")},
	{handler: "/healthz", pathPattern: regexp.MustCompile("^/healthz")},
	{handler: "/robots.txt", pathPattern: regexp.MustCompile("^/robots.txt$")},
	// bundle all pprof endpoints under the same handler name
	{handler: "/debug/pprof-handlers", pathPattern: regexp.MustCompile("^/debug/pprof")},
}

// routeOperationName receives the route operation name from context, if set.
func routeOperationName(req *http.Request) (string, bool) {
	if val := req.Context().Value(routeOperationNameKey); val != nil {
		op, ok := val.(string)
		return op, ok
	}

	for _, hp := range unnamedHandlers {
		if hp.pathPattern.Match([]byte(req.URL.Path)) {
			return hp.handler, true
		}
	}

	return "", false
}

func RequestTracing(tracer tracing.Tracer) web.Handler {
	return func(res http.ResponseWriter, req *http.Request, c *web.Context) {
		if strings.HasPrefix(c.Req.URL.Path, "/public/") ||
			c.Req.URL.Path == "/robots.txt" ||
			c.Req.URL.Path == "/favicon.ico" {
			c.Next()
			return
		}

		rw := res.(web.ResponseWriter)

		// spanContext := trace.SpanContextFromContext(req.Context())
		// traceID := spanContext.TraceID()
		// fmt.Println("TRACE ID:", traceID)
		var thisCtx context.Context
		wireContext := tracer.Extract(req.Context(), req.Header)
		// for k, v := range req.Header {
		// 	fmt.Println("STRING:", k, v)
		// }
		if len(req.Header["Uber-Trace-Id"]) != 0 {
			traceIDShort := req.Header["Uber-Trace-Id"][0][0:16]
			traceIDString := traceIDShort + "0000000000000000"
			traceID, err := trace.TraceIDFromHex(traceIDString)
			fmt.Println("TRACE ID:", traceIDShort, traceIDString, traceID, err)
			spanID, err := trace.SpanIDFromHex(req.Header["Uber-Trace-Id"][0][17:33])
			fmt.Println("SPAN ID:", spanID, err)
			spanContext := trace.NewSpanContext(trace.SpanContextConfig{TraceID: traceID, SpanID: spanID})
			thisCtx = trace.ContextWithSpanContext(req.Context(), spanContext)
		}

		// tracet.SpanContext
		ctx, span := tracer.Start(thisCtx, fmt.Sprintf("HTTP %s %s", req.Method, req.URL.Path), &wireContext)
		c.Req = req.WithContext(ctx)
		// spanContext := trace.SpanContextFromContext(ctx)
		// traceID := spanContext.TraceID()
		// fmt.Println("TRACE ID:", traceID)
		c.Next()

		// Only call span.End when a route operation name have been set,
		// meaning that not set the span would not be reported.
		if routeOperation, exists := routeOperationName(c.Req); exists {
			defer span.End()
			span.SetName(fmt.Sprintf("HTTP %s %s", req.Method, routeOperation))
		}

		status := rw.Status()

		span.SetAttributes("http.status_code", status, attribute.Int("http.status_code", status))
		span.SetAttributes("http.url", req.RequestURI, attribute.String("http.url", req.RequestURI))
		span.SetAttributes("http.method", req.Method, attribute.String("http.method", req.Method))
		if status >= 400 {
			span.SetStatus(codes.Error, fmt.Sprintf("error with HTTP status code %s", strconv.Itoa(status)))
		}
	}
}
