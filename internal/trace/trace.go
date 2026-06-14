// Database Gateway provides access to servers with ACL for safe and restricted database interactions.
// Copyright (C) 2024  Kirill Zhuravlev
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package trace

import (
	"time"
)

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
