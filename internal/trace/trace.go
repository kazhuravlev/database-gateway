package trace

import "time"

type Record struct {
	Start    time.Time     `json:"start"`
	Duration time.Duration `json:"duration"`
	Name     string        `json:"name"`
}

type Trace struct {
	Records []Record `json:"records"`
}

func (t *Trace) Start(name string) func() {
	rec := Record{
		Start:    time.Now(),
		Duration: 0,
		Name:     name,
	}

	return func() {
		rec.Duration = time.Since(rec.Start)
		t.Records = append(t.Records, rec)
	}
}
