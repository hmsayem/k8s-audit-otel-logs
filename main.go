package main

import (
	"context"
	"encoding/json"
	"fmt"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.25.0"
	"io"
	auditapi "k8s.io/apiserver/pkg/apis/audit/v1"
	"log/slog"
	"os"
	"strings"
)

func main() {
	ctx := context.Background()

	// Create resource.
	res, err := newResource()
	if err != nil {
		panic(err)
	}

	// Create a logger provider.
	// You can pass this instance directly when creating bridges.
	loggerProvider, err := newLoggerProvider(res)
	if err != nil {
		panic(err)
	}

	// Handle shutdown properly so nothing leaks.
	defer func() {
		if err := loggerProvider.Shutdown(ctx); err != nil {
			fmt.Println(err)
		}
	}()

	logger := slog.New(newOtelLogHandler(loggerProvider))

	event, err := readAuditEvent("audit.log")

	logger.LogAttrs(ctx, 8, "Hello World!", getAttributesFromAuditEvent(event)...)

}

func newOtelLogHandler(loggerProvider *log.LoggerProvider) slog.Handler {
	return otelslog.NewHandler("test", otelslog.WithLoggerProvider(loggerProvider))
}

func getAttributesFromAuditEvent(event *auditapi.Event) []slog.Attr {
	attrs := []slog.Attr{
		{
			Key:   "audit.level",
			Value: slog.AnyValue(event.Level),
		},
		{
			Key:   "audit.auditID",
			Value: slog.AnyValue(event.AuditID),
		},
		{
			Key:   "audit.stage",
			Value: slog.AnyValue(event.Stage),
		},
		{
			Key:   "audit.requestURI",
			Value: slog.AnyValue(event.RequestURI),
		},
		{
			Key:   "audit.verb",
			Value: slog.AnyValue(event.Verb),
		},

		{
			Key:   "audit.user.username",
			Value: slog.AnyValue(event.User.Username),
		},
		{
			Key:   "audit.user.uid",
			Value: slog.AnyValue(event.User.UID),
		},
		{
			Key:   "audit.user.groups",
			Value: slog.AnyValue(strings.Join(event.User.Groups, ",")),
		},

		{
			Key:   "audit.sourceIPs",
			Value: slog.AnyValue(strings.Join(event.SourceIPs, ",")),
		},
		{
			Key:   "audit.userAgent",
			Value: slog.AnyValue(event.UserAgent),
		},

		{
			Key:   "audit.objectRef.uid",
			Value: slog.AnyValue(event.ObjectRef.UID),
		},
		{
			Key:   "audit.objectRef.resource",
			Value: slog.AnyValue(event.ObjectRef.Resource),
		},
		{
			Key:   "audit.objectRef.name",
			Value: slog.AnyValue(event.ObjectRef.Name),
		},
		{
			Key:   "audit.objectRef.namespace",
			Value: slog.AnyValue(event.ObjectRef.Namespace),
		},
		{
			Key:   "audit.objectRef.apiGroup",
			Value: slog.AnyValue(event.ObjectRef.APIGroup),
		},

		{
			Key:   "audit.objectRef.apiVersion",
			Value: slog.AnyValue(event.ObjectRef.APIVersion),
		},
		{
			Key:   "audit.objectRef.resourceVersion",
			Value: slog.AnyValue(event.ObjectRef.ResourceVersion),
		},

		{
			Key:   "audit.requestObject",
			Value: slog.AnyValue(event.RequestObject),
		},
		{
			Key:   "audit.responseObject",
			Value: slog.AnyValue(event.ResponseObject),
		},
		{
			Key:   "audit.responseStatus",
			Value: slog.AnyValue(event.ResponseStatus),
		},
		{
			Key:   "audit.requestReceivedTimestamp",
			Value: slog.AnyValue(event.RequestReceivedTimestamp),
		},
		{
			Key:   "audit.stageTimestamp",
			Value: slog.AnyValue(event.StageTimestamp),
		},
	}
	return attrs
}

func readAuditEvent(filePath string) (*auditapi.Event, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	// Read the JSON file
	byteValue, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	var event auditapi.Event
	if err := json.Unmarshal(byteValue, &event); err != nil {
		return nil, err
	}

	return &event, nil
}

func newResource() (*resource.Resource, error) {
	return resource.Merge(resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("test-service"),
			semconv.ServiceVersion("0.1.0"),
		))
}

func newLoggerProvider(res *resource.Resource) (*log.LoggerProvider, error) {
	exporter, err := getStdoutLogExporter()
	if err != nil {
		return nil, err
	}
	processor := log.NewBatchProcessor(exporter, log.WithMaxQueueSize(4), log.WithExportMaxBatchSize(1))
	provider := log.NewLoggerProvider(
		log.WithResource(res),
		log.WithProcessor(processor),
	)
	return provider, nil
}

func getStdoutLogExporter() (log.Exporter, error) {
	return stdoutlog.New(stdoutlog.WithPrettyPrint())
}

func getHTTPlogExporter() (log.Exporter, error) {
	return otlploghttp.New(context.Background())
}
