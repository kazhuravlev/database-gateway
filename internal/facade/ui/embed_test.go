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

package ui //nolint:testpackage

import (
	"fmt"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubDistFSPanicIncludesOperationAndPath(t *testing.T) {
	t.Parallel()

	const dir = "../dist"

	panicValue := requirePanic(t, func() {
		subDistFS(fstest.MapFS{}, dir)
	})

	message := fmt.Sprint(panicValue)
	assert.Contains(t, message, "create embedded ui fs")
	assert.Contains(t, message, dir)
}

func requirePanic(t *testing.T, fn func()) any {
	t.Helper()

	var panicValue any
	func() {
		defer func() {
			panicValue = recover()
		}()

		fn()
	}()

	require.NotNil(t, panicValue)

	return panicValue
}
