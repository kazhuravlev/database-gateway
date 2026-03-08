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

package facade //nolint:testpackage

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kazhuravlev/database-gateway/internal/config"
	"github.com/kazhuravlev/database-gateway/internal/structs"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

var errUnexpectedAuthCall = errors.New("unexpected auth call")

func TestExtractBearerToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		header    string
		wantToken string
	}{
		{
			name:      "valid",
			header:    "Bearer abc-123",
			wantToken: "abc-123",
		},
		{
			name:      "valid mixed case scheme",
			header:    "bEaReR abc-123",
			wantToken: "abc-123",
		},
		{
			name:      "missing token",
			header:    "Bearer ",
			wantToken: "",
		},
		{
			name:      "wrong scheme",
			header:    "Basic abc-123",
			wantToken: "",
		},
		{
			name:      "empty",
			header:    "",
			wantToken: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tc.wantToken, extractBearerToken(tc.header))
		})
	}
}

func TestWithAPIBearerAuthRequiresToken(t *testing.T) {
	t.Parallel()

	svc := &Service{
		opts: Options{
			logger:       slog.New(slog.DiscardHandler),
			app:          nil,
			cookieSecret: "",
			port:         0,
			corsAllowAll: false,
		},
		authByAccessToken: func(context.Context, string) (*structs.User, error) {
			t.Fatal("must not be called without bearer token")

			return nil, errUnexpectedAuthCall
		},
		initOIDC:           nil,
		completeOIDC:       nil,
		buildOIDCLogoutURL: nil,
		lrpc:               nil,
	}

	echoInst := echo.New()
	echoInst.GET("/api/v1/ping", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	}, svc.withAPIBearerAuth())

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/ping", http.NoBody)
	rec := httptest.NewRecorder()
	echoInst.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestWithAPIBearerAuthInjectsUserIntoContext(t *testing.T) {
	t.Parallel()

	expectedUser := structs.User{
		ID:       config.UserID("alice@example.com"),
		Username: "alice",
		Role:     config.RoleUser,
	}
	svc := &Service{
		opts: Options{
			logger:       slog.New(slog.DiscardHandler),
			app:          nil,
			cookieSecret: "",
			port:         0,
			corsAllowAll: false,
		},
		authByAccessToken: func(_ context.Context, token string) (*structs.User, error) {
			require.Equal(t, "token-1", token)

			return &expectedUser, nil
		},
		initOIDC:           nil,
		completeOIDC:       nil,
		buildOIDCLogoutURL: nil,
		lrpc:               nil,
	}

	echoInst := echo.New()
	echoInst.GET("/api/v1/ping", func(c echo.Context) error {
		user, err := userFromAPIToken(c.Request().Context())
		require.NoError(t, err)
		require.Equal(t, expectedUser, user)

		return c.NoContent(http.StatusOK)
	}, svc.withAPIBearerAuth())

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/ping", http.NoBody)
	req.Header.Set(echo.HeaderAuthorization, "Bearer token-1")
	rec := httptest.NewRecorder()
	echoInst.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
}
