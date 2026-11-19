package metrics

import (
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	dto "github.com/prometheus/client_model/go"
	"gorm.io/gorm"
)

var (
	httpRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "HTTP requests processed.",
	}, []string{"method", "path", "status"})
	httpDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "http_request_duration_seconds",
		Help: "HTTP request duration.",
	}, []string{"method", "path", "status"})
	gormQueries = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "gorm_queries_total",
		Help: "GORM queries processed.",
	}, []string{"operation"})
	gormDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "gorm_query_duration_seconds",
		Help: "GORM query duration.",
	}, []string{"operation"})
	wsConnectionsCurrent = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "ws_connections_current",
		Help: "Open WebSocket connections.",
	})
	wsConnectionsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "ws_connections_total",
		Help: "WebSocket connections accepted.",
	})
	wsFramesReceived = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "ws_frames_received_total",
		Help: "WebSocket text frames received.",
	})
	wsFramesSent = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "ws_frames_sent_total",
		Help: "WebSocket text frames sent.",
	})
	wsSendDrops = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "ws_send_drops_total",
		Help: "WebSocket frames dropped because a send buffer was full.",
	})
	wsHandlerDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "ws_handler_duration_seconds",
		Help: "WebSocket handler duration.",
	}, []string{"method"})

	registerOnce sync.Once
)

func register() {
	registerOnce.Do(func() {
		prometheus.MustRegister(
			httpRequests, httpDuration, gormQueries, gormDuration,
			wsConnectionsCurrent, wsConnectionsTotal,
			wsFramesReceived, wsFramesSent, wsSendDrops, wsHandlerDuration,
		)
	})
}

// Mount registers collectors, records Gin requests, and serves GET /metrics.
// Call it once, before application routes are registered.
func Mount(engine *gin.Engine) {
	if engine == nil {
		return
	}
	register()
	engine.Use(Gin())
	engine.GET("/metrics", gin.WrapH(promhttp.Handler()))
}

// Gin records one observation after the handler returns.
// The path label is the route template. Unmatched requests use "unmatched".
func Gin() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		path := c.FullPath()
		if path == "" {
			path = "unmatched"
		}
		labels := prometheus.Labels{
			"method": c.Request.Method,
			"path":   path,
			"status": strconv.Itoa(c.Writer.Status()),
		}
		httpRequests.With(labels).Inc()
		httpDuration.With(labels).Observe(time.Since(start).Seconds())
	}
}

// Instrument records GORM query counts and durations on db.
func Instrument(db *gorm.DB) {
	if db == nil {
		return
	}
	register()
	for _, op := range []struct {
		callback string
		name     string
	}{
		{"gorm:query", "query"},
		{"gorm:create", "create"},
		{"gorm:update", "update"},
		{"gorm:delete", "delete"},
		{"gorm:raw", "raw"},
	} {
		processor := processor(op.name)
		before := "metrics:before_" + op.name
		after := "metrics:after_" + op.name
		switch op.name {
		case "query":
			_ = db.Callback().Query().Before(op.callback).Register(before, markStart)
			_ = db.Callback().Query().After(op.callback).Register(after, processor)
		case "create":
			_ = db.Callback().Create().Before(op.callback).Register(before, markStart)
			_ = db.Callback().Create().After(op.callback).Register(after, processor)
		case "update":
			_ = db.Callback().Update().Before(op.callback).Register(before, markStart)
			_ = db.Callback().Update().After(op.callback).Register(after, processor)
		case "delete":
			_ = db.Callback().Delete().Before(op.callback).Register(before, markStart)
			_ = db.Callback().Delete().After(op.callback).Register(after, processor)
		case "raw":
			_ = db.Callback().Raw().Before(op.callback).Register(before, markStart)
			_ = db.Callback().Raw().After(op.callback).Register(after, processor)
		}
	}
}

func markStart(db *gorm.DB) {
	db.InstanceSet("metrics:start", time.Now())
}

func processor(operation string) func(*gorm.DB) {
	return func(db *gorm.DB) {
		start, ok := db.InstanceGet("metrics:start")
		if !ok {
			return
		}
		began, ok := start.(time.Time)
		if !ok {
			return
		}
		gormQueries.WithLabelValues(operation).Inc()
		gormDuration.WithLabelValues(operation).Observe(time.Since(began).Seconds())
	}
}

// WSConnectionOpened records one accepted connection.
func WSConnectionOpened() {
	wsConnectionsCurrent.Inc()
	wsConnectionsTotal.Inc()
}

// WSConnectionClosed records one connection leaving the hub.
func WSConnectionClosed() {
	wsConnectionsCurrent.Dec()
}

// WSFrameReceived records one inbound text frame.
func WSFrameReceived() {
	wsFramesReceived.Inc()
}

// WSFrameSent records one outbound text frame.
func WSFrameSent() {
	wsFramesSent.Inc()
}

// WSSendDropped records one frame that did not fit in a send buffer.
func WSSendDropped() {
	wsSendDrops.Inc()
}

// ObserveWSHandler records handler latency.
// method is the registered name, or "unknown" when the client named something else.
func ObserveWSHandler(method string, d time.Duration) {
	if method == "" {
		method = "unknown"
	}
	wsHandlerDuration.WithLabelValues(method).Observe(d.Seconds())
}

// DroppedSends returns how many outbound frames were dropped.
func DroppedSends() float64 {
	metric := &dto.Metric{}
	if err := wsSendDrops.Write(metric); err != nil || metric.Counter == nil {
		return 0
	}
	return metric.Counter.GetValue()
}

// HandlerMethods returns method labels that have recorded a duration.
func HandlerMethods() []string {
	ch := make(chan prometheus.Metric, 32)
	go func() {
		wsHandlerDuration.Collect(ch)
		close(ch)
	}()
	var out []string
	for item := range ch {
		metric := &dto.Metric{}
		if err := item.Write(metric); err != nil {
			continue
		}
		for _, label := range metric.Label {
			if label.GetName() == "method" {
				out = append(out, label.GetValue())
			}
		}
	}
	return out
}
