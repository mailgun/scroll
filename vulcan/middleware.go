package vulcan

import "fmt"

type Middleware struct {
	Type     string
	ID       string
	Priority int
	Spec     MiddlewareSpec
}

type MiddlewareSpec interface {
	Format() string
}

func (m Middleware) Format() string {
	return fmt.Sprintf(`{"Type": "%v", "Id": "%v", "Priority": %v, "Middleware": %v}`,
		m.Type, m.ID, m.Priority, m.Spec.Format())
}

func (m Middleware) String() string {
	return fmt.Sprintf("Middleware(Type=%v, ID=%v, Priority=%v, Spec=%v)",
		m.Type, m.ID, m.Priority, m.Spec)
}
