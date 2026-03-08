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
	"encoding/gob"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gorilla/sessions"
	"github.com/kazhuravlev/database-gateway/internal/app"
	"github.com/kazhuravlev/database-gateway/internal/config"
	"github.com/kazhuravlev/database-gateway/internal/structs"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

var (
	errNotUsed         = errors.New("not used in this test")
	errBrokenDiscovery = errors.New("broken oidc discovery")
)

func TestAuthFlowStoresStateAndCreatesSession(t *testing.T) {
	t.Parallel()

	const (
		state   = "state-123"
		authURL = "https://auth.example.com/application/o/authorize/"
	)
	var completeOIDCCalled bool

	svc := &Service{
		opts: Options{
			logger:       slog.New(slog.DiscardHandler),
			app:          nil,
			cookieSecret: "very-secret-key-for-tests",
			port:         0,
			corsAllowAll: false,
		},
		initOIDC: func(context.Context) (string, string, error) {
			return authURL, state, nil
		},
		completeOIDC: func(_ context.Context, code, expectedState, receivedState string) (*structs.User, time.Time, *app.OIDCTokens, error) {
			completeOIDCCalled = true
			require.Equal(t, "oidc-code-1", code)
			require.Equal(t, state, expectedState)
			require.Equal(t, state, receivedState)

			return &structs.User{
					ID:       config.UserID("alice@example.com"),
					Username: "alice",
					Role:     config.RoleUser,
				}, time.Now().Add(time.Hour), &app.OIDCTokens{
					IDToken:     "",
					AccessToken: "access-token-1",
				}, nil
		},
		buildOIDCLogoutURL: func(_, _ string) (string, error) {
			return "", errNotUsed
		},
		authByAccessToken: nil,
		lrpc:              nil,
	}
	echoInst := authTestEcho(svc)

	reqAuth := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "http://gateway.local/auth", http.NoBody)
	recAuth := httptest.NewRecorder()
	echoInst.ServeHTTP(recAuth, reqAuth)
	require.Equal(t, http.StatusSeeOther, recAuth.Code)
	require.Equal(t, authURL, recAuth.Header().Get(echo.HeaderLocation))

	stateCookie := getSessionCookie(recAuth.Result().Cookies())
	require.NotNil(t, stateCookie)

	reqCallback := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"/auth/callback?code=oidc-code-1&state="+url.QueryEscape(state),
		http.NoBody,
	)
	reqCallback.AddCookie(stateCookie)
	recCallback := httptest.NewRecorder()
	echoInst.ServeHTTP(recCallback, reqCallback)
	require.Equal(t, http.StatusSeeOther, recCallback.Code, recCallback.Body.String())
	require.Equal(t, buildAuthRedirectURL("access-token-1"), recCallback.Header().Get(echo.HeaderLocation))
	require.True(t, completeOIDCCalled)

	userCookie := getSessionCookie(recCallback.Result().Cookies())
	require.NotNil(t, userCookie)
}

func TestAuthCallbackRequiresStateInSession(t *testing.T) {
	t.Parallel()

	svc := &Service{
		opts: Options{
			logger:       slog.New(slog.DiscardHandler),
			app:          nil,
			cookieSecret: "very-secret-key-for-tests",
			port:         0,
			corsAllowAll: false,
		},
		initOIDC: func(context.Context) (string, string, error) {
			return "", "", errNotUsed
		},
		completeOIDC: func(context.Context, string, string, string) (*structs.User, time.Time, *app.OIDCTokens, error) {
			return nil, time.Time{}, nil, errNotUsed
		},
		buildOIDCLogoutURL: func(string, string) (string, error) {
			return "", errNotUsed
		},
		authByAccessToken: nil,
		lrpc:              nil,
	}
	echoInst := authTestEcho(svc)

	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"/auth/callback?code=oidc-code-1&state=any",
		http.NoBody,
	)
	rec := httptest.NewRecorder()
	echoInst.ServeHTTP(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestLogoutRedirectsToOIDCEndSession(t *testing.T) {
	t.Parallel()

	const (
		state     = "state-123"
		authURL   = "https://auth.example.com/authorize"
		logoutURL = "https://auth.example.com/end-session?post_logout_redirect_uri=http%3A%2F%2Fgateway.local%2Fauth"
	)
	var gotPostLogoutRedirectURL string

	svc := &Service{
		opts: Options{
			logger:       slog.New(slog.DiscardHandler),
			app:          nil,
			cookieSecret: "very-secret-key-for-tests",
			port:         0,
			corsAllowAll: false,
		},
		initOIDC: func(context.Context) (string, string, error) {
			return authURL, state, nil
		},
		completeOIDC: func(context.Context, string, string, string) (*structs.User, time.Time, *app.OIDCTokens, error) {
			return &structs.User{
					ID:       config.UserID("alice@example.com"),
					Username: "alice",
					Role:     config.RoleUser,
				}, time.Now().Add(time.Hour), &app.OIDCTokens{
					IDToken:     "",
					AccessToken: "access-token-1",
				}, nil
		},
		buildOIDCLogoutURL: func(_ string, postLogoutRedirectURL string) (string, error) {
			gotPostLogoutRedirectURL = postLogoutRedirectURL

			return logoutURL, nil
		},
		authByAccessToken: nil,
		lrpc:              nil,
	}
	echoInst := authTestEcho(svc)

	reqAuth := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "http://gateway.local/auth", http.NoBody)
	recAuth := httptest.NewRecorder()
	echoInst.ServeHTTP(recAuth, reqAuth)
	stateCookie := getSessionCookie(recAuth.Result().Cookies())
	require.NotNil(t, stateCookie)

	reqCallback := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"/auth/callback?code=oidc-code-1&state="+url.QueryEscape(state),
		http.NoBody,
	)
	reqCallback.AddCookie(stateCookie)
	recCallback := httptest.NewRecorder()
	echoInst.ServeHTTP(recCallback, reqCallback)
	userCookie := getSessionCookie(recCallback.Result().Cookies())
	require.NotNil(t, userCookie)

	reqLogout := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "http://gateway.local/logout", http.NoBody)
	reqLogout.AddCookie(userCookie)
	recLogout := httptest.NewRecorder()
	echoInst.ServeHTTP(recLogout, reqLogout)

	require.Equal(t, http.StatusSeeOther, recLogout.Code)
	require.Equal(t, logoutURL, recLogout.Header().Get(echo.HeaderLocation))
	require.Equal(t, "http://gateway.local/auth", gotPostLogoutRedirectURL)
}

func TestLogoutFallbackToAuthWhenOIDCLogoutURLFails(t *testing.T) {
	t.Parallel()

	svc := &Service{
		opts: Options{
			logger:       slog.New(slog.DiscardHandler),
			app:          nil,
			cookieSecret: "very-secret-key-for-tests",
			port:         0,
			corsAllowAll: false,
		},
		initOIDC: func(context.Context) (string, string, error) {
			return "", "", errNotUsed
		},
		completeOIDC: func(context.Context, string, string, string) (*structs.User, time.Time, *app.OIDCTokens, error) {
			return nil, time.Time{}, nil, errNotUsed
		},
		buildOIDCLogoutURL: func(string, string) (string, error) {
			return "", errBrokenDiscovery
		},
		authByAccessToken: nil,
		lrpc:              nil,
	}
	echoInst := authTestEcho(svc)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "http://gateway.local/logout", http.NoBody)
	rec := httptest.NewRecorder()
	echoInst.ServeHTTP(rec, req)

	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Equal(t, "/auth", rec.Header().Get(echo.HeaderLocation))
}

func TestBuildAuthRedirectURL(t *testing.T) {
	t.Parallel()

	require.Equal(
		t,
		"/ui#access_token=token%2Bwith%2Fchars&token_type=Bearer",
		buildAuthRedirectURL("token+with/chars"),
	)
}

func authTestEcho(svc *Service) *echo.Echo {
	gob.Register(structs.User{}) //nolint:exhaustruct
	gob.Register(config.UserID(""))

	echoInst := echo.New()
	echoInst.HTTPErrorHandler = func(err error, c echo.Context) {
		_ = c.String(http.StatusInternalServerError, err.Error())
	}
	echoInst.Use(session.Middleware(sessions.NewCookieStore([]byte(svc.opts.cookieSecret))))
	echoInst.GET("/auth", svc.getAuth)
	echoInst.GET("/auth/callback", svc.getAuthCallback)
	echoInst.GET("/logout", svc.logout)

	return echoInst
}

func getSessionCookie(cookies []*http.Cookie) *http.Cookie {
	for _, cookie := range cookies {
		if cookie.Name == keySession {
			return cookie
		}
	}

	return nil
}
