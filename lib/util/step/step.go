package step

import "time"

type Step struct {
	*time.Ticker
	Current int64
	Value   float64
}

func New(d time.Duration, v float64) *Step {
	return &Step{
		time.NewTicker(d),
		time.Now().Unix(),
		v,
	}
}

func (s *Step) Now() int64 {
	return time.Now().Unix()
}

func (s *Step) Increment(v float64) {
	s.Value = s.Value + v
	s.Current = s.Now()
}

func (s *Step) Decrement(v float64) {
	s.Value = s.Value - v
	s.Current = s.Now()
}
