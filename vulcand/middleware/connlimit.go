package middleware

import (
	"fmt"

	"github.com/mailgun/scroll/vulcand"
)

const (
	ConnLimitType = "connlimit"
	ConnLimitID   = "cl1"
)

// ConnLimit is a spec for the respective vulcan's middleware that lets to control amount if simultaneous
// connections to locations.
type ConnLimit struct {
	Variable    string `json:"Variable"`
	Connections int    `json:"Connections"`
}

func NewConnLimit(spec ConnLimit) vulcand.Middleware {
	return vulcand.Middleware{
		Type:     ConnLimitType,
		ID:       ConnLimitID,
		Priority: vulcand.DefaultMiddlewarePriority,
		Spec:     spec,
	}
}

func (cl ConnLimit) String() string {
	return fmt.Sprintf("ConnLimit(Variable=%v, Connections=%v)",
		cl.Variable, cl.Connections)
}
