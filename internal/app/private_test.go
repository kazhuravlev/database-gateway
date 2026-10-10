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

package app //nolint:testpackage

import (
	"context"
	"log/slog"
	"net"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kazhuravlev/database-gateway/internal/config"
	"github.com/stretchr/testify/require"
)

func newTestConnService() *Service {
	return &Service{ //nolint:exhaustruct_v5
		opts: Options{ //nolint:exhaustruct_v5
			logger: slog.Default(),
		},
		connsMu: new(sync.RWMutex),
		conns:   make(map[config.TargetID]*pgxpool.Pool),
	}
}

// freeLocalAddr reserves a local TCP port and immediately releases it, so that
// nothing is listening on it and any connection attempt is refused.
func freeLocalAddr(t *testing.T) (string, int) {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	addr, ok := l.Addr().(*net.TCPAddr)
	require.True(t, ok)

	require.NoError(t, l.Close())

	return addr.IP.String(), addr.Port
}

func TestGetConnection_UnreachableTargetReturnsError(t *testing.T) {
	host, port := freeLocalAddr(t)

	svc := newTestConnService()
	target := config.Target{ //nolint:exhaustruct
		ID: "unreachable-target",
		Connection: config.Connection{
			Host:     host,
			Port:     port,
			User:     "app",
			Password: "some-password",
			DB:       "app",
			UseSSL:   false,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var err error
	require.NotPanics(t, func() {
		_, err = svc.getConnection(ctx, target)
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), string(target.ID))
	require.Contains(t, err.Error(), host)
	require.Contains(t, err.Error(), strconv.Itoa(port))

	svc.connsMu.RLock()
	_, stored := svc.conns[target.ID]
	svc.connsMu.RUnlock()
	require.False(t, stored, "a failed connection must not be cached")
}

func TestGetConnection_ErrorDoesNotLeakPassword(t *testing.T) {
	host, port := freeLocalAddr(t)

	svc := newTestConnService()
	const password = "very-secret-password"
	target := config.Target{ //nolint:exhaustruct
		ID: "unreachable-target",
		Connection: config.Connection{
			Host:     host,
			Port:     port,
			User:     "app",
			Password: password,
			DB:       "app",
			UseSSL:   false,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := svc.getConnection(ctx, target)
	require.Error(t, err)
	require.NotContains(t, err.Error(), password)
}
