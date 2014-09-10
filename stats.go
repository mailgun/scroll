package scroll

import (
	"fmt"
	"net/http"
	"time"

	"github.com/mailgun/metrics"
)

type AppStats struct {
	metrics metrics.Metrics
}

func NewAppStats(metrics metrics.Metrics) *AppStats {
	return &AppStats{metrics}
}

func (s *AppStats) TrackRequest(metricID string, status int, time time.Duration) {
	s.TrackRequestTime(metricID, time)
	s.TrackTotalRequests(metricID)
	if status != http.StatusOK {
		s.TrackFailedRequests(metricID, status)
	}
}

func (s *AppStats) TrackRequestTime(metricID string, time time.Duration) {
	s.metrics.EmitTimer(fmt.Sprintf("api.%v.time", metricID), time)
}

func (s *AppStats) TrackTotalRequests(metricID string) {
	s.metrics.EmitCounter(fmt.Sprintf("api.%v.count.total", metricID), 1)
}

func (s *AppStats) TrackFailedRequests(metricID string, status int) {
	s.metrics.EmitCounter(fmt.Sprintf("api.%v.count.failed.%v", metricID, status), 1)
}
