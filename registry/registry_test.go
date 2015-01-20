package registry

import (
	"testing"
	"time"

	. "gopkg.in/check.v1"
)

type FakeRegistrationStrategy struct {
	RegistrationCount int
}

func (s *FakeRegistrationStrategy) RegisterApp(registration *AppRegistration) error {
	s.RegistrationCount++
	return nil
}

func (s *FakeRegistrationStrategy) RegisterHandler(registration *HandlerRegistration) error {
	return nil
}

func TestRegistry(t *testing.T) {
	TestingT(t)
}

type RegistrySuite struct {
}

var _ = Suite(&RegistrySuite{})

func (s *RegistrySuite) TestStartRegistersAppAtInterval(c *C) {
	registration := &AppRegistration{}
	strategy := &FakeRegistrationStrategy{}

	heartbeater := NewHeartbeater(registration, strategy)
	heartbeater.Start()
	time.Sleep(30 * time.Millisecond)

	c.Assert(strategy.RegistrationCount, Equals, 3)

	heartbeater.Stop()
	time.Sleep(30 * time.Millisecond)

	c.Assert(strategy.RegistrationCount, Equals, 3)
}
