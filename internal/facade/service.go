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

package facade

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/csv"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/sessions"
	"github.com/kazhuravlev/database-gateway/internal/app"
	"github.com/kazhuravlev/database-gateway/internal/config"
	"github.com/kazhuravlev/database-gateway/internal/facade/ui"
	"github.com/kazhuravlev/database-gateway/internal/structs"
	"github.com/kazhuravlev/database-gateway/internal/uuid6"
	"github.com/kazhuravlev/just"
	"github.com/kazhuravlev/lrpc/ctypes"
	lrpcserver "github.com/kazhuravlev/lrpc/server"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

const (
	ctxUser        = "c-user"
	keySession     = "session"
	keyUserID      = "uid"
	keyOIDCState   = "oidc-state"
	exportNonceLen = 8
)

var (
	errBadTokenFormat    = errors.New("bad token format")
	errBadTokenSignature = errors.New("bad token signature")
	errBadExportFormat   = errors.New("bad export format")
	errTokenExpired      = errors.New("token expired")
	errNoSessionUser     = errors.New("no session user")
)

//nolint:gochecknoglobals
var corsMwForDevelopment = middleware.CORSWithConfig(middleware.CORSConfig{
	Skipper:         nil,
	AllowOrigins:    []string{"localhost", "*"},
	AllowOriginFunc: nil,
	AllowMethods: []string{
		http.MethodOptions,
		http.MethodGet,
		http.MethodHead,
		http.MethodPut,
		http.MethodPatch,
		http.MethodPost,
		http.MethodDelete,
	},
	AllowHeaders:                             []string{},
	AllowCredentials:                         true,
	UnsafeWildcardOriginWithAllowCredentials: true,
	ExposeHeaders:                            nil,
	MaxAge:                                   0,
})

type lrpcContextKey string

const (
	ctxAPITokenUser lrpcContextKey = "api-user"
)

type Service struct {
	opts Options

	initOIDC           func(ctx context.Context) (string, string, error)
	completeOIDC       func(ctx context.Context, code, expectedState, receivedState string) (*structs.User, time.Time, *app.OIDCTokens, error)
	buildOIDCLogoutURL func(idTokenHint, postLogoutRedirectURL string) (string, error)
	authByAccessToken  func(ctx context.Context, token string) (*structs.User, error)
	lrpc               *lrpcserver.Server
}

func New(opts Options) (*Service, error) {
	gob.Register(structs.User{}) //nolint:exhaustruct
	gob.Register(config.UserID(""))

	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("bad configuration: %w", err)
	}

	lrpc, err := lrpcserver.New(lrpcserver.NewOptions(
		lrpcserver.WithLogger(opts.logger.With(slog.String("mod", "lrpc"))),
		lrpcserver.WithName("dbgw"),
	))
	if err != nil {
		return nil, fmt.Errorf("create lrpc server: %w", err)
	}

	return &Service{
		opts:               opts,
		initOIDC:           opts.app.InitOIDC,
		completeOIDC:       opts.app.CompleteOIDC,
		buildOIDCLogoutURL: opts.app.BuildOIDCLogoutURL,
		authByAccessToken:  opts.app.AuthByAccessToken,
		lrpc:               lrpc,
	}, nil
}

func (s *Service) Run(_ context.Context) error {
	echoInst := echo.New()
	echoInst.HideBanner = true
	echoInst.Use(middleware.Recover())
	echoInst.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		Skipper:        middleware.DefaultSkipper,
		BeforeNextFunc: nil,
		LogValuesFunc: func(_ echo.Context, reqVals middleware.RequestLoggerValues) error {
			logFn := s.opts.logger.Info
			errMsg := ""

			if reqVals.Error != nil {
				logFn = s.opts.logger.Error
				errMsg = reqVals.Error.Error()
			}

			logFn("req",
				slog.Time("start", reqVals.StartTime),
				slog.String("method", reqVals.Method),
				slog.String("uri", reqVals.URI),
				slog.Int("status", reqVals.Status),
				slog.Duration("dur", reqVals.Latency),
				slog.String("err", errMsg),
			)

			return nil
		},
		HandleError:      false,
		LogLatency:       true,
		LogProtocol:      false,
		LogRemoteIP:      false,
		LogHost:          false,
		LogMethod:        true,
		LogURI:           true,
		LogURIPath:       false,
		LogRoutePath:     false,
		LogRequestID:     false,
		LogReferer:       false,
		LogUserAgent:     false,
		LogStatus:        true,
		LogError:         true,
		LogContentLength: false,
		LogResponseSize:  false,
		LogHeaders:       nil,
		LogQueryParams:   nil,
		LogFormValues:    nil,
	}))
	echoInst.Use(session.Middleware(sessions.NewCookieStore([]byte(s.opts.cookieSecret))))

	if s.opts.corsAllowAll {
		echoInst.Use(corsMwForDevelopment)
	}

	echoInst.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			path := c.Request().URL.Path
			if strings.HasPrefix(path, "/auth") {
				return next(c)
			}
			if strings.HasPrefix(path, "/logout") {
				return next(c)
			}
			if strings.HasPrefix(path, "/api/v1/") {
				return next(c)
			}

			sess, err := session.Get(keySession, c)
			if err != nil {
				return fmt.Errorf("have no session: %w", err)
			}

			user, ok := sess.Values[keyUserID]
			if !ok {
				return c.Redirect(http.StatusSeeOther, "/auth")
			}

			strUser, ok := user.(structs.User)
			if !ok {
				return c.Redirect(http.StatusSeeOther, "/logout")
			}

			c.Set(ctxUser, strUser)

			return next(c)
		}
	})

	echoInst.GET("/ui", s.getApp)
	echoInst.GET("/ui/*", s.getApp)

	echoInst.GET("/", func(c echo.Context) error {
		return c.Redirect(http.StatusSeeOther, "/ui")
	})
	echoInst.GET("/auth", s.getAuth)
	echoInst.GET("/auth/callback", s.getAuthCallback)

	echoInst.GET("/logout", s.logout)

	{
		errorMapping := map[error]ctypes.ErrorCode{
			errBadInput:      400,
			app.ErrForbidden: 403,
			app.ErrNotFound:  404,
		}

		lrpcserver.RegisterHandler(s.lrpc, "profile.get.v1", s.lrpcProfileGet, errorMapping)
		lrpcserver.RegisterHandler(s.lrpc, "targets.list.v1", s.lrpcTargetList, errorMapping)
		lrpcserver.RegisterHandler(s.lrpc, "targets.get.v1", s.lrpcTargetGet, errorMapping)
		lrpcserver.RegisterHandler(s.lrpc, "bookmarks.list.v1", s.lrpcBookmarksList, errorMapping)
		lrpcserver.RegisterHandler(s.lrpc, "bookmarks.add.v1", s.lrpcBookmarksAdd, errorMapping)
		lrpcserver.RegisterHandler(s.lrpc, "bookmarks.delete.v1", s.lrpcBookmarksDelete, errorMapping)
		lrpcserver.RegisterHandler(s.lrpc, "queries.list.v1", s.lrpcQueriesList, errorMapping)
		lrpcserver.RegisterHandler(s.lrpc, "admin.requests.list.v1", s.lrpcAdminRequestsList, errorMapping)
		lrpcserver.RegisterHandler(s.lrpc, "query.run.v1", s.lrpcQueryRun, errorMapping)
		lrpcserver.RegisterHandler(s.lrpc, "query-results.get.v1", s.lrpcQueryResultsGet, errorMapping)
		lrpcserver.RegisterHandler(s.lrpc, "query-results.export-link.v1", s.lrpcQueryResultsExportLink, errorMapping)

		var apiGroup *echo.Group
		if s.opts.corsAllowAll {
			//nolint:contextcheck
			apiGroup = echoInst.Group("/api/v1", corsMwForDevelopment, s.withAPIBearerAuth())
		} else {
			//nolint:contextcheck
			apiGroup = echoInst.Group("/api/v1", s.withAPIBearerAuth())
		}

		apiGroup.GET("/schema", echo.WrapHandler(s.lrpc.HTTPHandlerSchema()))
		apiGroup.POST("/:method", echo.WrapHandler(s.lrpc.HTTPHandler()))
	}
	echoInst.GET("/api/v1/query-results/export/:token", s.downloadQueryResultsExport)
	echoInst.GET("/*", func(c echo.Context) error {
		return c.Redirect(http.StatusSeeOther, "/ui")
	})

	echoInst.Logger.Fatal(echoInst.Start(":" + strconv.Itoa(s.opts.port)))

	return nil
}

func (s *Service) withAPIBearerAuth() echo.MiddlewareFunc { //nolint:contextcheck
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			token := extractBearerToken(c.Request().Header.Get(echo.HeaderAuthorization))
			if token == "" {
				return c.NoContent(http.StatusUnauthorized)
			}

			user, err := s.authByAccessToken(c.Request().Context(), token)
			if err != nil {
				s.opts.logger.Warn("authenticate api token", slog.String("error", err.Error()))

				return c.NoContent(http.StatusUnauthorized)
			}

			ctx := context.WithValue(c.Request().Context(), ctxAPITokenUser, *user)
			c.SetRequest(c.Request().WithContext(ctx))

			return next(c)
		}
	}
}

func extractBearerToken(authHeader string) string {
	if authHeader == "" {
		return ""
	}

	scheme, token, found := strings.Cut(strings.TrimSpace(authHeader), " ")
	if !found {
		return ""
	}
	if !strings.EqualFold(scheme, "Bearer") {
		return ""
	}

	return strings.TrimSpace(token)
}

func (*Service) getApp(c echo.Context) error {
	requestPath := strings.TrimPrefix(c.Request().URL.Path, "/ui")
	requestPath = strings.TrimPrefix(requestPath, "/")

	if requestPath != "" {
		info, err := fs.Stat(ui.DistFS, requestPath)
		if err == nil && !info.IsDir() {
			http.ServeFileFS(c.Response(), c.Request(), ui.DistFS, requestPath)

			return nil
		}
	}

	http.ServeFileFS(c.Response(), c.Request(), ui.DistFS, "index.html")

	return nil
}

func (s *Service) getAuth(c echo.Context) error {
	authURL, state, err := s.initOIDC(c.Request().Context())
	if err != nil {
		return fmt.Errorf("init oidc: %w", err)
	}

	// Store state in session to validate on callback.
	sess, err := session.Get(keySession, c)
	if err != nil {
		return fmt.Errorf("get session: %w", err)
	}

	sess.Options = &sessions.Options{ //nolint:exhaustruct
		Path:     "/",
		MaxAge:   300, // 5 minutes - state is short-lived
		HttpOnly: true,
	}
	sess.Values[keyOIDCState] = state
	if err := sess.Save(c.Request(), c.Response()); err != nil {
		return fmt.Errorf("save session with state: %w", err)
	}

	return c.Redirect(http.StatusSeeOther, authURL)
}

func (s *Service) getAuthCallback(c echo.Context) error {
	sess, err := session.Get(keySession, c)
	if err != nil {
		return fmt.Errorf("get session: %w", err)
	}

	expectedState, ok := sess.Values[keyOIDCState].(string)
	if !ok {
		return errors.New("no state found in session - possible CSRF attack") //nolint:err113
	}

	// Get state and code from callback URL
	receivedState := c.Request().URL.Query().Get("state")
	code := c.Request().URL.Query().Get("code")

	user, expiry, tokens, err := s.completeOIDC(c.Request().Context(), code, expectedState, receivedState)
	if err != nil {
		return fmt.Errorf("complete oidc: %w", err)
	}

	// Clear the one-time state from session and set user session
	sess.Options = &sessions.Options{ //nolint:exhaustruct
		Path:     "/",
		MaxAge:   int(time.Until(expiry).Seconds()),
		HttpOnly: true,
	}
	delete(sess.Values, keyOIDCState) // Remove used state
	sess.Values[keyUserID] = *user
	if err := sess.Save(c.Request(), c.Response()); err != nil {
		return fmt.Errorf("save session: %w", err)
	}

	return c.Redirect(http.StatusSeeOther, buildAuthRedirectURL(tokens.AccessToken))
}

func (s *Service) logout(c echo.Context) error {
	sess, err := session.Get(keySession, c)
	if err != nil {
		return fmt.Errorf("get session: %w", err)
	}

	sess.Options = &sessions.Options{ //nolint:exhaustruct
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	}
	delete(sess.Values, keyUserID)
	delete(sess.Values, keyOIDCState)
	if err := sess.Save(c.Request(), c.Response()); err != nil {
		return fmt.Errorf("save session: %w", err)
	}

	postLogoutRedirectURL := fmt.Sprintf("%s://%s/auth", c.Scheme(), c.Request().Host)
	logoutURL, err := s.buildOIDCLogoutURL("", postLogoutRedirectURL)
	if err != nil {
		return c.Redirect(http.StatusSeeOther, "/auth")
	}

	return c.Redirect(http.StatusSeeOther, logoutURL)
}

func buildAuthRedirectURL(accessToken string) string {
	query := url.Values{}
	query.Set("access_token", accessToken)
	query.Set("token_type", "Bearer")

	return "/ui#" + query.Encode()
}

func (s *Service) downloadQueryResultsExport(c echo.Context) error {
	user, err := s.currentExportUser(c)
	if err != nil {
		return c.NoContent(http.StatusUnauthorized)
	}

	claims, err := s.parseQueryResultsExportToken(c.Param("token"))
	if err != nil {
		return c.NoContent(http.StatusNotFound)
	}

	if claims.UserID != user.ID {
		return c.NoContent(http.StatusNotFound)
	}

	qRes, err := s.lookupQueryResultsExport(c, user, claims.QueryResultID)
	if err != nil {
		return err
	}

	return serveExport(c, claims.Format, qRes)
}

func marshalQueryResultsJSON(qRes *app.QueryResults) ([]byte, error) {
	qTbl := just.SliceMap(qRes.QTable.Rows, func(row []string) map[string]any {
		m := make(map[string]any, len(qRes.QTable.Headers))
		for i := range qRes.QTable.Headers {
			m[qRes.QTable.Headers[i]] = row[i]
		}

		return m
	})

	return json.Marshal(struct {
		Meta structs.QMeta    `json:"meta"`
		Rows []map[string]any `json:"rows"`
	}{
		Meta: qRes.Meta,
		Rows: qTbl,
	})
}

func marshalQueryResultsCSV(qRes *app.QueryResults) ([]byte, error) {
	var csvBuf bytes.Buffer
	csvWriter := csv.NewWriter(&csvBuf)
	if err := csvWriter.Write(qRes.QTable.Headers); err != nil {
		return nil, err
	}
	if err := csvWriter.WriteAll(qRes.QTable.Rows); err != nil {
		return nil, err
	}
	csvWriter.Flush()
	if err := csvWriter.Error(); err != nil {
		return nil, err
	}

	return csvBuf.Bytes(), nil
}

type queryResultsExportTokenClaims struct {
	UserID        config.UserID `json:"user_id"`
	QueryResultID uuid6.UUID    `json:"query_result_id"`
	Format        string        `json:"format"`
	ExpiresAt     int64         `json:"expires_at"`
	Nonce         string        `json:"nonce"`
}

func (s *Service) buildQueryResultsExportToken(
	userID config.UserID,
	queryResultID uuid6.UUID,
	format string,
	expiresAt time.Time,
) (string, error) {
	nonce := make([]byte, exportNonceLen)
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}

	claims := queryResultsExportTokenClaims{
		UserID:        userID,
		QueryResultID: queryResultID,
		Format:        format,
		ExpiresAt:     expiresAt.Unix(),
		Nonce:         base64.RawURLEncoding.EncodeToString(nonce),
	}

	payload, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("marshal claims: %w", err)
	}

	sig := s.signQueryResultsExportPayload(payload)

	return base64.RawURLEncoding.EncodeToString(payload) + "." +
		base64.RawURLEncoding.EncodeToString(sig), nil
}

func (s *Service) parseQueryResultsExportToken(rawToken string) (*queryResultsExportTokenClaims, error) {
	rawToken = strings.TrimSpace(rawToken)
	payloadPart, sigPart, ok := strings.Cut(rawToken, ".")
	if !ok {
		return nil, errBadTokenFormat
	}

	payload, err := base64.RawURLEncoding.DecodeString(payloadPart)
	if err != nil {
		return nil, fmt.Errorf("decode payload: %w", err)
	}

	gotSig, err := base64.RawURLEncoding.DecodeString(sigPart)
	if err != nil {
		return nil, fmt.Errorf("decode signature: %w", err)
	}

	wantSig := s.signQueryResultsExportPayload(payload)
	if !hmac.Equal(gotSig, wantSig) {
		return nil, errBadTokenSignature
	}

	var claims queryResultsExportTokenClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("unmarshal claims: %w", err)
	}

	if !isSupportedExportFormat(claims.Format) {
		return nil, errBadExportFormat
	}
	if time.Now().Unix() > claims.ExpiresAt {
		return nil, errTokenExpired
	}

	return &claims, nil
}

func (s *Service) signQueryResultsExportPayload(payload []byte) []byte {
	mac := hmac.New(sha256.New, []byte(s.opts.cookieSecret))
	_, _ = mac.Write(payload)

	return mac.Sum(nil)
}

func (s *Service) currentExportUser(c echo.Context) (*structs.User, error) {
	token := extractBearerToken(c.Request().Header.Get(echo.HeaderAuthorization))
	if token != "" {
		return s.authByAccessToken(c.Request().Context(), token)
	}

	sess, err := session.Get(keySession, c)
	if err != nil {
		return nil, err
	}

	user, ok := sess.Values[keyUserID].(structs.User)
	if !ok {
		return nil, errNoSessionUser
	}

	return &user, nil
}

func (s *Service) lookupQueryResultsExport(
	c echo.Context,
	user *structs.User,
	queryResultID uuid6.UUID,
) (*app.QueryResults, error) {
	qRes, err := s.opts.app.GetQueryResults(c.Request().Context(), *user, queryResultID)
	if err != nil {
		switch {
		case errors.Is(err, app.ErrNotFound):
			return nil, c.NoContent(http.StatusNotFound)
		case errors.Is(err, app.ErrForbidden):
			return nil, c.NoContent(http.StatusForbidden)
		default:
			return nil, fmt.Errorf("get query results: %w", err)
		}
	}

	return qRes, nil
}

func serveExport(c echo.Context, format string, qRes *app.QueryResults) error {
	var (
		payload  []byte
		filename string
		err      error
	)

	switch format {
	case "json":
		filename = "response.json"
		payload, err = marshalQueryResultsJSON(qRes)
		if err != nil {
			return fmt.Errorf("marshal query results json: %w", err)
		}
	case "csv":
		filename = "response.csv"
		payload, err = marshalQueryResultsCSV(qRes)
		if err != nil {
			return fmt.Errorf("marshal query results csv: %w", err)
		}
	default:
		return c.NoContent(http.StatusNotFound)
	}

	c.Response().Header().Set(echo.HeaderContentDisposition, fmt.Sprintf(`%s; filename="%s"`, "attachment", filename)) //nolint
	http.ServeContent(c.Response(), c.Request(), filename, time.Now(), bytes.NewReader(payload))

	return nil
}

func isSupportedExportFormat(format string) bool {
	switch format {
	case "json", "csv":
		return true
	default:
		return false
	}
}
