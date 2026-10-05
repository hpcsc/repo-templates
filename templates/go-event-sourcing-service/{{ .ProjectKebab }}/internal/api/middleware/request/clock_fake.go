package request

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func NewFakeClock(t *testing.T, steps ...time.Time) *fakeClock {
	require.NotEmpty(t, steps)

	return &fakeClock{
		t:           t,
		steps:       steps,
		currentStep: 0,
	}
}

// fakeClock advances to next time step after each Now() call, to simulate time progressing which is not under the test control
type fakeClock struct {
	t           *testing.T
	steps       []time.Time
	currentStep int
}

var _ Clock = (*fakeClock)(nil)

func (c *fakeClock) Now() time.Time {
	currentStep := c.currentStep
	require.Less(c.t, currentStep, len(c.steps))
	c.currentStep++
	return c.steps[currentStep]
}

func (c *fakeClock) Since(t time.Time) time.Duration {
	return c.steps[c.currentStep].Sub(t)
}
