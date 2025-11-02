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

//nolint:testpackage
package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetVersion(t *testing.T) {
	t.Parallel()

	require.Equal(t, "(devel)", GetVersion())

	t.Run("returns explicitly set version when version variable is set", func(t *testing.T) {
		t.Parallel()

		// Save original value
		original := version
		defer func() { version = original }()

		// Set explicit version
		version = "v1.2.3"
		assert.Equal(t, "v1.2.3", GetVersion())
	})
}
