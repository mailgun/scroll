package middleware

import "fmt"

const (
	DefaultPriority = 1
)

type T struct {
	Type     string         `json:"Type"`
	ID       string         `json:"Id"`
	Priority int            `json:"Priority"`
	Spec     MiddlewareSpec `json:"Middleware"`
}

type MiddlewareSpec interface{}

func (m T) String() string {
	return fmt.Sprintf("Middleware(Type=%v, ID=%v, Priority=%v, Spec=%v)",
		m.Type, m.ID, m.Priority, m.Spec)
}
