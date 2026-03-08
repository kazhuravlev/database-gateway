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
	"encoding/csv"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/gorilla/sessions"
	"github.com/kazhuravlev/database-gateway/internal/app"
	"github.com/kazhuravlev/database-gateway/internal/config"
	"github.com/kazhuravlev/database-gateway/internal/facade/static"
	"github.com/kazhuravlev/database-gateway/internal/facade/templates"
	"github.com/kazhuravlev/database-gateway/internal/structs"
	"github.com/kazhuravlev/database-gateway/internal/uuid6"
	"github.com/kazhuravlev/just"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

const (
	ctxUser      = "c-user"
	keySession   = "session"
	keyUserID    = "uid"
	keyOIDCState = "oidc-state"
)

type Service struct {
	opts Options

	initOIDC           func(ctx context.Context) (string, string, error)
	completeOIDC       func(ctx context.Context, code, expectedState, receivedState string) (*structs.User, time.Time, error)
	buildOIDCLogoutURL func(idTokenHint, postLogoutRedirectURL string) (string, error)
}

func New(opts Options) (*Service, error) {
	gob.Register(structs.User{}) //nolint:exhaustruct
	gob.Register(config.UserID(""))

	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("bad configuration: %w", err)
	}

	return &Service{
		opts:               opts,
		initOIDC:           opts.app.InitOIDC,
		completeOIDC:       opts.app.CompleteOIDC,
		buildOIDCLogoutURL: opts.app.BuildOIDCLogoutURL,
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

	echoInst.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			path := c.Request().URL.Path
			if strings.HasPrefix(path, "/static") {
				return next(c)
			}
			if strings.HasPrefix(path, "/auth") {
				return next(c)
			}
			if strings.HasPrefix(path, "/logout") {
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

	echoInst.StaticFS("/static", static.Files)

	echoInst.GET("/", func(c echo.Context) error {
		return c.Redirect(http.StatusSeeOther, "/servers")
	})
	echoInst.GET("/servers", s.getServers)
	echoInst.GET("/servers/:id", s.getServer)
	echoInst.POST("/servers/:id", s.runQuery)
	echoInst.POST("/servers/:id/bookmarks", s.addBookmark)
	echoInst.POST("/servers/:id/bookmarks/:bid/delete", s.deleteBookmark)
	echoInst.GET("/servers/:id/:qid", s.getQueryResults)
	echoInst.GET("/admin/requests", s.getAdminRequests)
	echoInst.GET("/admin/requests/:qid", s.getAdminRequest)

	echoInst.GET("/auth", s.getAuth)
	echoInst.GET("/auth/callback", s.getAuthCallback)

	echoInst.GET("/logout", s.logout)

	echoInst.GET("/*", func(c echo.Context) error {
		return c.Redirect(http.StatusSeeOther, "/")
	})

	echoInst.Logger.Fatal(echoInst.Start(":" + strconv.Itoa(s.opts.port)))

	return nil
}

func (s *Service) getServers(c echo.Context) error {
	user := c.Get(ctxUser).(structs.User) //nolint:forcetypeassert

	servers, err := s.opts.app.GetTargets(c.Request().Context(), user)
	if err != nil {
		s.opts.logger.Error("get targets", slog.String("error", err.Error()))

		return c.String(http.StatusInternalServerError, "the sky was falling")
	}

	bookmarks, err := s.opts.app.ListAllBookmarks(c.Request().Context(), user.ID)
	if err != nil {
		s.opts.logger.Error("list bookmarks", slog.String("error", err.Error()))

		return c.String(http.StatusInternalServerError, "the sky was falling")
	}

	const maxLastQueries = 50
	recentQueries, err := s.opts.app.ListRecentQueries(c.Request().Context(), user.ID, maxLastQueries)
	if err != nil {
		s.opts.logger.Error("list recent queries", slog.String("error", err.Error()))

		return c.String(http.StatusInternalServerError, "the sky was falling")
	}

	return Render(c, http.StatusOK, templates.PageTargetsList(user, servers, bookmarks, recentQueries))
}

func (s *Service) getServer(c echo.Context) error {
	user := c.Get(ctxUser).(structs.User) //nolint:forcetypeassert

	tID := config.TargetID(c.Param("id"))
	srv, err := s.opts.app.GetTargetByID(c.Request().Context(), user, tID)
	if err != nil {
		return fmt.Errorf("get target by id: %w", err)
	}

	formURL := "/servers/" + srv.ID.S()
	bookmarks, err := s.opts.app.ListBookmarks(c.Request().Context(), user, srv.ID)
	if err != nil {
		return fmt.Errorf("list bookmarks: %w", err)
	}

	return Render(c, http.StatusOK, templates.PageTarget(user, *srv, formURL, ``, bookmarks, nil, nil, nil))
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

	user, expiry, err := s.completeOIDC(c.Request().Context(), code, expectedState, receivedState)
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

	return c.Redirect(http.StatusSeeOther, "/")
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

func (s *Service) runQuery(c echo.Context) error {
	user := c.Get(ctxUser).(structs.User) //nolint:forcetypeassert

	tID := config.TargetID(c.Param("id"))
	srv, err := s.opts.app.GetTargetByID(c.Request().Context(), user, tID)
	if err != nil {
		return fmt.Errorf("get target by id: %w", err)
	}

	params, err := c.FormParams()
	if err != nil {
		return c.Redirect(http.StatusSeeOther, "/")
	}

	query := params.Get("query")
	format := params.Get("format")
	formURL := "/servers/" + srv.ID.S()
	bookmarks, bErr := s.opts.app.ListBookmarks(c.Request().Context(), user, srv.ID)
	if bErr != nil {
		return fmt.Errorf("list bookmarks: %w", bErr)
	}

	queryID, _, err := s.opts.app.RunQuery(c.Request().Context(), user, srv.ID, query)
	if err != nil {
		return Render(c, http.StatusOK, templates.PageTarget(user, *srv, formURL, query, bookmarks, nil, nil, err)) //nolint:err113
	}

	params2 := url.Values{}
	params2.Set("format", format)
	targetURL := fmt.Sprintf("/servers/%s/%s?%s", srv.ID, queryID.S(), params2.Encode())

	return c.Redirect(http.StatusSeeOther, targetURL)
}

func (s *Service) addBookmark(c echo.Context) error {
	user := c.Get(ctxUser).(structs.User) //nolint:forcetypeassert

	tID := config.TargetID(c.Param("id"))
	if err := s.opts.app.AddBookmark(
		c.Request().Context(),
		user,
		tID,
		c.FormValue("title"),
		c.FormValue("query"),
	); err != nil {
		return fmt.Errorf("add bookmark: %w", err)
	}

	return c.Redirect(http.StatusSeeOther, "/servers/"+tID.S())
}

func (s *Service) deleteBookmark(c echo.Context) error {
	user := c.Get(ctxUser).(structs.User) //nolint:forcetypeassert

	tID := config.TargetID(c.Param("id"))
	bookmarkID, err := uuid6.ParseStr(c.Param("bid"))
	if err != nil {
		return fmt.Errorf("parse bookmark id: %w", err)
	}
	if err := s.opts.app.DeleteBookmark(c.Request().Context(), user.ID, bookmarkID); err != nil {
		return fmt.Errorf("delete bookmark: %w", err)
	}

	returnTo := c.FormValue("return_to")
	if returnTo == "" || !strings.HasPrefix(returnTo, "/") || strings.HasPrefix(returnTo, "//") {
		returnTo = "/servers/" + tID.S()
	}

	return c.Redirect(http.StatusSeeOther, returnTo)
}

func (s *Service) getQueryResults(c echo.Context) error { //nolint:cyclop
	ctx := c.Request().Context()
	user := c.Get(ctxUser).(structs.User) //nolint:forcetypeassert

	tID := config.TargetID(c.Param("id"))
	srv, err := s.opts.app.GetTargetByID(ctx, user, tID)
	if err != nil {
		return fmt.Errorf("get target by id: %w", err)
	}

	qID := uuid6.FromStr(c.Param("qid"))
	qRes, err := s.opts.app.GetQueryResults(ctx, user.ID, qID)
	if err != nil {
		return fmt.Errorf("get query results: %w", err)
	}

	params, err := c.FormParams()
	if err != nil {
		return c.Redirect(http.StatusSeeOther, "/")
	}

	format := params.Get("format")
	formURL := "/servers/" + srv.ID.S()
	bookmarks, bErr := s.opts.app.ListBookmarks(ctx, user, srv.ID)
	if bErr != nil {
		return fmt.Errorf("list bookmarks: %w", bErr)
	}

	switch format {
	default:
		correctedURL := c.Request().URL.Path + "?format=html"

		return c.Redirect(http.StatusSeeOther, correctedURL)
	case "html":
		return Render(c, http.StatusOK, templates.PageTarget(user, *srv, formURL, qRes.Query, bookmarks, &qRes.QTable, &qRes.Meta, nil))
	case "json":
		qTbl2 := just.SliceMap(qRes.QTable.Rows, func(row []string) map[string]any {
			m := make(map[string]any, len(qRes.QTable.Headers))
			for i := range qRes.QTable.Headers {
				m[qRes.QTable.Headers[i]] = row[i]
			}

			return m
		})

		resBuf, err := json.Marshal(struct {
			Meta structs.QMeta    `json:"meta"`
			Rows []map[string]any `json:"rows"`
		}{
			Meta: qRes.Meta,
			Rows: qTbl2,
		})
		if err != nil {
			return Render(c, http.StatusOK, templates.PageTarget(user, *srv, formURL, qRes.Query, bookmarks, nil, nil, err))
		}

		c.Response().Header().Set(echo.HeaderContentDisposition, fmt.Sprintf(`%s; filename="%s"`, "attachment", "response.json")) //nolint
		http.ServeContent(c.Response(), c.Request(), "response.json", time.Now(), bytes.NewReader(resBuf))

		return nil
	case "csv":
		var csvBuf bytes.Buffer
		csvWriter := csv.NewWriter(&csvBuf)
		if err := csvWriter.Write(qRes.QTable.Headers); err != nil {
			return Render(c, http.StatusOK, templates.PageTarget(user, *srv, formURL, qRes.Query, bookmarks, nil, nil, err))
		}
		if err := csvWriter.WriteAll(qRes.QTable.Rows); err != nil {
			return Render(c, http.StatusOK, templates.PageTarget(user, *srv, formURL, qRes.Query, bookmarks, nil, nil, err))
		}
		csvWriter.Flush()
		if err := csvWriter.Error(); err != nil {
			return Render(c, http.StatusOK, templates.PageTarget(user, *srv, formURL, qRes.Query, bookmarks, nil, nil, err))
		}

		c.Response().Header().Set(echo.HeaderContentDisposition, fmt.Sprintf(`%s; filename="%s"`, "attachment", "response.csv")) //nolint
		http.ServeContent(c.Response(), c.Request(), "response.csv", time.Now(), bytes.NewReader(csvBuf.Bytes()))

		return nil
	}
}

func (s *Service) getAdminRequests(c echo.Context) error {
	ctx := c.Request().Context()
	user := c.Get(ctxUser).(structs.User) //nolint:forcetypeassert

	page := int64(parsePositiveInt(c.QueryParam("page"), 1))
	const pageSize = int64(50)

	items, hasNext, err := s.opts.app.ListAdminRequests(ctx, user, page, pageSize)
	if err != nil {
		if errors.Is(err, app.ErrForbidden) {
			return c.String(http.StatusForbidden, "forbidden")
		}

		return fmt.Errorf("list admin requests: %w", err)
	}

	return Render(c, http.StatusOK, templates.PageAdminRequests(user, items, int(page), page > 1, hasNext))
}

func (s *Service) getAdminRequest(c echo.Context) error {
	ctx := c.Request().Context()
	user := c.Get(ctxUser).(structs.User) //nolint:forcetypeassert

	qID, err := uuid6.ParseStr(c.Param("qid"))
	if err != nil {
		return c.String(http.StatusNotFound, "not found")
	}

	item, err := s.opts.app.GetAdminQueryResults(ctx, user, qID)
	if err != nil {
		switch {
		case errors.Is(err, app.ErrForbidden):
			return c.String(http.StatusForbidden, "forbidden")
		case errors.Is(err, app.ErrNotFound):
			return c.String(http.StatusNotFound, "not found")
		default:
			return fmt.Errorf("get admin query result: %w", err)
		}
	}

	return Render(c, http.StatusOK, templates.PageAdminRequest(user, *item))
}

func Render(ctx echo.Context, statusCode int, t templ.Component) error {
	buf := templ.GetBuffer()
	defer templ.ReleaseBuffer(buf)

	if err := t.Render(ctx.Request().Context(), buf); err != nil {
		return err
	}

	return ctx.HTML(statusCode, buf.String())
}

func parsePositiveInt(input string, defaultValue int) int {
	if input == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(input)
	if err != nil || value <= 0 {
		return defaultValue
	}

	return value
}
