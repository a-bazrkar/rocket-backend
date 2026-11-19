package metrics

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestWebsocketCollectorsRegister(t *testing.T) {
	register()
	ObserveWSHandler("ping", time.Millisecond)
	families, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{
		"ws_connections_current":      false,
		"ws_connections_total":        false,
		"ws_frames_received_total":    false,
		"ws_frames_sent_total":        false,
		"ws_send_drops_total":         false,
		"ws_handler_duration_seconds": false,
	}
	for _, family := range families {
		if _, ok := want[family.GetName()]; ok {
			want[family.GetName()] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("missing %s", name)
		}
	}
	if wsSendDrops == nil || wsHandlerDuration == nil {
		t.Fatal("collectors were not created")
	}
	metric := &dto.Metric{}
	if err := wsConnectionsCurrent.Write(metric); err != nil {
		t.Fatal(err)
	}
}
