package metrics

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/metal-toolbox/disko/internal/model"
	rctypes "github.com/metal-toolbox/rivets/condition"
)

const (
	prefix          = model.AppName + "_"
	MetricsEndpoint = "0.0.0.0:9090"
)

var (
	EventsCounter *prometheus.CounterVec

	ConditionRunTimeSummary     *prometheus.SummaryVec
	ActionRuntimeSummary        *prometheus.SummaryVec
	ActionHandlerRunTimeSummary *prometheus.SummaryVec

	DownloadBytes          *prometheus.CounterVec
	DownloadRunTimeSummary *prometheus.SummaryVec
	UploadBytes            *prometheus.CounterVec
	UploadRunTimeSummary   *prometheus.SummaryVec

	StoreQueryErrorCount *prometheus.CounterVec

	NATSErrors *prometheus.CounterVec
)

func init() {
	EventsCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: prefix + "events_received",
			Help: "A counter metric to measure the total count of events received",
		},
		[]string{"valid", "response"}, // valid is true/false, response is ack/nack
	)

	ConditionRunTimeSummary = promauto.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: prefix + "condition_duration_seconds",
			Help: "A summary metric to measure the total time spent in completing each condition",
		},
		[]string{"condition", "state"},
	)

	ActionRuntimeSummary = promauto.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: prefix + "mount_action_runtime_seconds",
			Help: "A summary metric to measure the total time spent in each mount action",
		},
		[]string{"vendor", "state"},
	)

	DownloadBytes = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: prefix + "download_bytes",
			Help: "A counter metric to measure images downloaded in bytes",
		},
		[]string{"component", "vendor"},
	)

	UploadBytes = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: prefix + "upload_bytes",
			Help: "A counter metric to measure images uploaded in bytes",
		},
		[]string{"component", "vendor"},
	)

	StoreQueryErrorCount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: prefix + "store_query_error_count",
			Help: "A counter metric to measure the total count of errors querying the asset store.",
		},
		[]string{"storeKind", "queryKind"},
	)

	NATSErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: prefix + "nats_errors",
			Help: "A count of errors while trying to use NATS.",
		},
		[]string{"operation"},
	)
}

// ListenAndServeMetrics exposes prometheus metrics as /metrics
func ListenAndServe() {
	go func() {
		http.Handle("/metrics", promhttp.Handler())

		server := &http.Server{
			Addr:              MetricsEndpoint,
			ReadHeaderTimeout: 2 * time.Second, // nolint:gomnd // time duration value is clear as is.
		}

		if err := server.ListenAndServe(); err != nil {
			log.Println(err)
		}
	}()
}

// RegisterSpanEvent adds a span event along with the given attributes.
//
// event here is arbitrary and can be in the form of strings like - publishCondition, updateCondition etc
func RegisterSpanEvent(span trace.Span, condition *rctypes.Condition, workerID, serverID, event string, err error) {
	attrs := []attribute.KeyValue{
		attribute.String("workerID", workerID),
		attribute.String("serverID", serverID),
		attribute.String("conditionID", condition.ID.String()),
		attribute.String("conditionKind", string(condition.Kind)),
	}

	if err != nil {
		attrs = append(attrs, attribute.String("error", err.Error()))
	}

	span.AddEvent(event, trace.WithAttributes(attrs...))
}

func NATSError(op string) {
	NATSErrors.WithLabelValues(op).Inc()
}

func RegisterEventCounter(valid bool, response string) {
	EventsCounter.With(
		prometheus.Labels{
			"valid":    strconv.FormatBool(valid),
			"response": response,
		}).Inc()
}

func RegisterConditionMetrics(startTS time.Time, state string) {
	ConditionRunTimeSummary.With(
		prometheus.Labels{
			"condition": string(rctypes.Inventory),
			"state":     state,
		},
	).Observe(time.Since(startTS).Seconds())
}
