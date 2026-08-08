package testutil

import "time"

type FixedClock struct{ Current time.Time }

func (clock FixedClock) Now() time.Time { return clock.Current }
