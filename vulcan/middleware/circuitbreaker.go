package middleware

import (
	"fmt"
	"time"
)

const (
	CircuitBreakerType = "cbreaker"
	CircuitBreakerID   = "cb1"
)

// CircuitBreaker is a spec for the respective vulcan's middleware that lets vulcan to fallback to
// some default response and trigger some action when an erroneous condition on a location is met.
type CircuitBreaker struct {
	Condition        string
	Fallback         string
	CheckPeriod      time.Duration
	FallbackDuration time.Duration
	RecoveryDuration time.Duration
	OnTripped        string
	OnStandby        string
}

func NewCircuitBreaker(spec CircuitBreaker) Middleware {
	return Middleware{
		Type:     CircuitBreakerType,
		ID:       CircuitBreakerID,
		Priority: DefaultPriority,
		Spec:     spec,
	}
}

func (cb CircuitBreaker) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`{"Condition": "%v", "Fallback": "%v", "CheckPeriod": %v, "FallbackDuration": %v, "RecoveryDuration": %v, "OnTripped": "%v", "OnStandby": "%v"}`,
		cb.Condition, cb.Fallback, cb.CheckPeriod, cb.FallbackDuration, cb.RecoveryDuration, cb.OnTripped, cb.OnStandby)), nil
}

func (cb CircuitBreaker) String() string {
	return fmt.Sprintf("CircuitBreaker(Condition=%v, Fallback=%v, CheckPeriod=%v, FallbackDuration=%v, RecoveryDuration=%v, OnTripped=%v, OnStandby=%v)",
		cb.Condition, cb.Fallback, cb.CheckPeriod, cb.FallbackDuration, cb.RecoveryDuration, cb.OnTripped, cb.OnStandby)
}
