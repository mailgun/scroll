package vulcan

import (
	"encoding/json"
	"fmt"
)

type Middleware struct {
	Type     string
	ID       string
	Priority int
	Spec     MiddlewareSpec
}

type MiddlewareSpec interface {
	json.Marshaler
}

func (m Middleware) MarshalJSON() ([]byte, error) {
	spec, _ := json.Marshal(m.Spec)
	return []byte(fmt.Sprintf(`{"Type": "%v", "Id": "%v", "Priority": %v, "Middleware": %v}`,
		m.Type, m.ID, m.Priority, spec)), nil
}

func (m Middleware) String() string {
	return fmt.Sprintf("Middleware(Type=%v, ID=%v, Priority=%v, Spec=%v)",
		m.Type, m.ID, m.Priority, m.Spec)
}
