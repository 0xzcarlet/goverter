package quota

import "time"

const JakartaTimezone = "Asia/Jakarta"

var jakartaLocation = mustLoadLocation(JakartaTimezone)

type Summary struct {
	Limit          int
	ReservedCount  int
	CompletedCount int
}

func mustLoadLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

func Location() *time.Location {
	return jakartaLocation
}

func DateForTime(t time.Time) time.Time {
	local := t.In(jakartaLocation)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, jakartaLocation)
}

func (s Summary) ActiveCount() int {
	return s.ReservedCount + s.CompletedCount
}

func (s Summary) Remaining() int {
	remaining := s.Limit - s.ActiveCount()
	if remaining < 0 {
		return 0
	}
	return remaining
}

func (s Summary) Exhausted() bool {
	return s.Remaining() == 0
}
