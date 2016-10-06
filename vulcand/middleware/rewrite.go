package middleware

import (
	"fmt"

	"github.com/mailgun/scroll/vulcand"
)

const (
	RewriteType = "rewrite"
	RewriteID   = "rw1"
)

// Rewrite is a spec for the respective vulcan's middleware that enables request/response
// alteration.
type Rewrite struct {
	Regexp      string `json:"Regexp"`
	Replacement string `json:"Replacement"`
	RewriteBody bool   `json:"RewriteBody"`
	Redirect    bool   `json:"Redirect"`
}

func NewRewrite(spec Rewrite) vulcand.Middleware {
	return vulcand.Middleware{
		Type:     RewriteType,
		ID:       RewriteID,
		Priority: vulcand.DefaultMiddlewarePriority,
		Spec:     spec,
	}
}

func (rw Rewrite) String() string {
	return fmt.Sprintf("Rewrite(Regexp=%v, Replacement=%v, RewriteBody=%v, Redirect=%v)",
		rw.Regexp, rw.Replacement, rw.RewriteBody, rw.Redirect)
}
