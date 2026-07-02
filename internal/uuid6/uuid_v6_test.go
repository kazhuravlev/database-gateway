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

package uuid6_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/kazhuravlev/database-gateway/internal/uuid6"
	"github.com/stretchr/testify/require"
)

func TestNil(t *testing.T) {
	t.Parallel()

	id := uuid6.Nil()

	require.True(t, id.IsNil())
	require.Equal(t, "00000000000000000000000000", id.String())
	require.Equal(t, uuid.Nil, id.ToUUID())
}

func TestFromUUID(t *testing.T) {
	t.Parallel()

	original := uuid.MustParse("018f1f60-7a0a-6c20-b9f2-147c3cb8778a")

	id := uuid6.FromUUID(original)

	require.False(t, id.IsNil())
	require.Equal(t, original, id.ToUUID())
}

func TestParseStr(t *testing.T) {
	t.Parallel()

	id, err := uuid6.ParseStr("01HZ7N9R8Y9V1B9KZ6ZQVV8A1S")
	require.NoError(t, err)
	require.Equal(t, "01HZ7N9R8Y9V1B9KZ6ZQVV8A1S", id.String())
}

func TestValue(t *testing.T) {
	t.Parallel()

	original := uuid.MustParse("018f1f60-7a0a-6c20-b9f2-147c3cb8778a")
	id := uuid6.FromUUID(original)

	value, err := id.Value()
	require.NoError(t, err)
	require.Equal(t, original.String(), value)
}

func TestScan(t *testing.T) {
	t.Parallel()

	original := uuid.MustParse("018f1f60-7a0a-6c20-b9f2-147c3cb8778a")

	tests := []struct {
		name  string
		input any
		want  uuid6.UUID
	}{
		{
			name:  "string",
			input: original.String(),
			want:  uuid6.FromUUID(original),
		},
		{
			name:  "bytes",
			input: []byte(original.String()),
			want:  uuid6.FromUUID(original),
		},
		{
			name:  "nil uuid bytes",
			input: uuid.Nil[:],
			want:  uuid6.Nil(),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var got uuid6.UUID
			err := got.Scan(tc.input)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestInvalidInputCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input any
	}{
		{
			name:  "nil scan input",
			input: nil,
		},
		{
			name:  "unsupported scan input",
			input: 42,
		},
		{
			name:  "invalid scan string",
			input: "not-a-uuid",
		},
		{
			name:  "invalid scan bytes",
			input: []byte("not-a-uuid"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var got uuid6.UUID
			err := got.Scan(tc.input)
			require.Error(t, err)
		})
	}

	_, err := uuid6.ParseStr("not-a-ulid")
	require.Error(t, err)
}
