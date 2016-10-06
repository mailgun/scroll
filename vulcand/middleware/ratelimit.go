package middleware

import (
	"fmt"

	"github.com/mailgun/scroll/vulcand"
)

const (
	RateLimitType = "ratelimit"
	RateLimitID   = "rl1"
)

// RateLimit is a spec for the respective vulcan's middleware that lets to apply request rate limits to
// locations.
type RateLimit struct {
	Variable      string `json:"Variable"`
	Requests      int    `json:"Requests"`
	PeriodSeconds int    `json:"PeriodSeconds"`
	Burst         int    `json:"Burst"`
}

func NewRateLimit(spec RateLimit) vulcand.Middleware {
	return vulcand.Middleware{
		Type:     RateLimitType,
		ID:       RateLimitID,
		Priority: vulcand.DefaultMiddlewarePriority,
		Spec:     spec,
	}
}

func (rl RateLimit) String() string {
	return fmt.Sprintf("RateLimit(Variable=%v, Requests=%v, PeriodSeconds=%v, Burst=%v)",
		rl.Variable, rl.Requests, rl.PeriodSeconds, rl.Burst)
}
