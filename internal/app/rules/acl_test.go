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
	"testing"

	"github.com/stretchr/testify/require"
)

func TestACLAllow_TableDriven(t *testing.T) {
	t.Parallel()

	subjects := []string{UserPrincipal("user1@example.com"), RolePrincipal("user")}

	testCases := []struct {
		name    string
		acls    []ACL
		filters []IFilter
		want    bool
	}{
		{
			name: "no source rules",
			acls: nil,
			filters: []IFilter{
				BySubjects(subjects...),
			},
			want: false,
		},
		{
			name: "no filters",
			acls: []ACL{
				{User: RolePrincipal("user"), Op: Star, Target: Star, Tbl: Star, Allow: true},
			},
			filters: nil,
			want:    false,
		},
		{
			name: "allow by matching role principal",
			acls: []ACL{
				{User: RolePrincipal("user"), Op: "select", Target: "pg-1", Tbl: "public.clients", Allow: true},
			},
			filters: []IFilter{
				BySubjects(subjects...),
				ByOp("select"),
				ByTargetID("pg-1"),
				ByTable("public.clients"),
			},
			want: true,
		},
		{
			name: "allow by star values",
			acls: []ACL{
				{User: Star, Op: Star, Target: Star, Tbl: Star, Allow: true},
			},
			filters: []IFilter{
				BySubjects(subjects...),
				ByOp("delete"),
				ByTargetID("pg-2"),
				ByTable("public.orders"),
			},
			want: true,
		},
		{
			name: "deny when op does not match",
			acls: []ACL{
				{User: RolePrincipal("user"), Op: "select", Target: "pg-1", Tbl: "public.clients", Allow: true},
			},
			filters: []IFilter{
				BySubjects(subjects...),
				ByOp("update"),
				ByTargetID("pg-1"),
				ByTable("public.clients"),
			},
			want: false,
		},
		{
			name: "first matching rule wins",
			acls: []ACL{
				{User: RolePrincipal("user"), Op: Star, Target: Star, Tbl: Star, Allow: false},
				{User: RolePrincipal("user"), Op: Star, Target: Star, Tbl: Star, Allow: true},
			},
			filters: []IFilter{
				BySubjects(subjects...),
				ByOp("select"),
				ByTargetID("pg-1"),
				ByTable("public.clients"),
			},
			want: false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := New(tc.acls).Allow(tc.filters...)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestBySubjects(t *testing.T) {
	t.Parallel()

	matched := ACL{User: RolePrincipal("admin")}
	notMatched := ACL{User: RolePrincipal("user")}
	filter := BySubjects(UserPrincipal("a@example.com"), RolePrincipal("admin"))

	require.True(t, filter(matched))
	require.False(t, filter(notMatched))
}
