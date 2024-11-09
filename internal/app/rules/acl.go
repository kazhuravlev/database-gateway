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

package rules

import (
	"github.com/kazhuravlev/just"
)

const Star = "*"

type ACL struct {
	User   string `json:"user"`   // can be a `userID` or `*` (means all)
	Op     string `json:"op"`     // can be a `specific operation` or `*` (means all)
	Target string `json:"target"` // can be a `specific target id` or `*` (means all)
	Tbl    string `json:"tbl"`    // can be a `specific table` or `*` (means all)
	Allow  bool   `json:"allow"`
}

type ACLs struct {
	source []ACL
}

func New(source []ACL) *ACLs {
	return &ACLs{
		source: source,
	}
}

func (a *ACLs) Allow(filters ...IFilter) bool {
	if len(filters) == 0 || len(a.source) == 0 {
		return false
	}

	res := make([]bool, len(filters))
	for _, acl := range a.source {
		for i, filter := range filters {
			res[i] = filter(acl)
		}

		if just.SliceAll(res, func(v bool) bool { return v }) {
			return acl.Allow
		}
	}

	return false
}
