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

package rules_test

import (
	"testing"

	"github.com/kazhuravlev/database-gateway/internal/app/rules"
	"github.com/stretchr/testify/require"
)

func TestACLAllow_TableDriven(t *testing.T) {
	t.Parallel()

	subjects := []string{rules.UserPrincipal("user1@example.com"), rules.RolePrincipal("user")}

	testCases := []struct {
		name    string
		acls    []rules.ACL
		filters []rules.IFilter
		want    bool
	}{
		{
			name: "no source rules",
			acls: nil,
			filters: []rules.IFilter{
				rules.BySubjects(subjects...),
			},
			want: false,
		},
		{
			name: "no filters",
			acls: []rules.ACL{
				{User: rules.RolePrincipal("user"), Op: rules.Star, Target: rules.Star, Tbl: rules.Star, Allow: true},
			},
			filters: nil,
			want:    false,
		},
		{
			name: "allow by matching role principal",
			acls: []rules.ACL{
				{User: rules.RolePrincipal("user"), Op: "select", Target: "pg-1", Tbl: "public.clients", Allow: true},
			},
			filters: []rules.IFilter{
				rules.BySubjects(subjects...),
				rules.ByOp("select"),
				rules.ByTargetID("pg-1"),
				rules.ByTable("public.clients"),
			},
			want: true,
		},
		{
			name: "allow by star values",
			acls: []rules.ACL{
				{User: rules.Star, Op: rules.Star, Target: rules.Star, Tbl: rules.Star, Allow: true},
			},
			filters: []rules.IFilter{
				rules.BySubjects(subjects...),
				rules.ByOp("delete"),
				rules.ByTargetID("pg-2"),
				rules.ByTable("public.orders"),
			},
			want: true,
		},
		{
			name: "deny when op does not match",
			acls: []rules.ACL{
				{User: rules.RolePrincipal("user"), Op: "select", Target: "pg-1", Tbl: "public.clients", Allow: true},
			},
			filters: []rules.IFilter{
				rules.BySubjects(subjects...),
				rules.ByOp("update"),
				rules.ByTargetID("pg-1"),
				rules.ByTable("public.clients"),
			},
			want: false,
		},
		{
			name: "first matching rule wins",
			acls: []rules.ACL{
				{User: rules.RolePrincipal("user"), Op: rules.Star, Target: rules.Star, Tbl: rules.Star, Allow: false},
				{User: rules.RolePrincipal("user"), Op: rules.Star, Target: rules.Star, Tbl: rules.Star, Allow: true},
			},
			filters: []rules.IFilter{
				rules.BySubjects(subjects...),
				rules.ByOp("select"),
				rules.ByTargetID("pg-1"),
				rules.ByTable("public.clients"),
			},
			want: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := rules.New(tc.acls).Allow(tc.filters...)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestBySubjects(t *testing.T) {
	t.Parallel()

	matched := rules.ACL{
		User:   rules.RolePrincipal("admin"),
		Op:     rules.Star,
		Target: rules.Star,
		Tbl:    rules.Star,
		Allow:  true,
	}
	notMatched := rules.ACL{
		User:   rules.RolePrincipal("user"),
		Op:     rules.Star,
		Target: rules.Star,
		Tbl:    rules.Star,
		Allow:  true,
	}
	filter := rules.BySubjects(rules.UserPrincipal("a@example.com"), rules.RolePrincipal("admin"))

	require.True(t, filter(matched))
	require.False(t, filter(notMatched))
}
