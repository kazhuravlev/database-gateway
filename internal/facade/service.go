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
	ctxUser    = "c-user"
	keySession = "session"
	keyUserID  = "uid"
)

type Service struct {
	opts Options
}

func New(opts Options) (*Service, error) {
	gob.Register(structs.User{}) //nolint:exhaustruct
	gob.Register(config.UserID(""))

	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("bad configuration: %w", err)
	}

	return &Service{opts: opts}, nil
}

func (s *Service) Run(_ context.Context) error {
	echoInst := echo.New()
	echoInst.HideBanner = true
	echoInst.Use(middleware.Recover())
	echoInst.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Skipper: middleware.DefaultSkipper,
		Format: `${time_rfc3339_nano} ${error} ` +
			`${method} ${uri} ` +
			`${status} ${latency_human} ` + "\n",
		CustomTimeFormat: "2006-01-02 15:04:05.00000",
		CustomTagFunc:    nil,
		Output:           nil,
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
	echoInst.GET("/servers/:id/:qid", s.getQueryResults)

	echoInst.GET("/auth", s.getAuth)
	echoInst.GET("/auth/callback", s.getAuthCallback)
	echoInst.POST("/auth", s.postAuth)

	echoInst.GET("/logout", s.logout)

	echoInst.GET("/*", func(c echo.Context) error {
		return c.Redirect(http.StatusSeeOther, "/")
	})

	echoInst.Logger.Fatal(echoInst.Start(":" + strconv.Itoa(s.opts.port)))

	return nil
}

func (s *Service) getServers(c echo.Context) error {
	user := c.Get(ctxUser).(structs.User) //nolint:forcetypeassert

	servers, err := s.opts.app.GetTargets(c.Request().Context(), user.ID)
	if err != nil {
		s.opts.logger.Error("get targets", slog.String("error", err.Error()))

		return c.String(http.StatusInternalServerError, "the sky was falling")
	}

	return Render(c, http.StatusOK, templates.PageTargetsList(user, servers))
}

func (s *Service) getServer(c echo.Context) error {
	user := c.Get(ctxUser).(structs.User) //nolint:forcetypeassert

	tID := config.TargetID(c.Param("id"))
	srv, err := s.opts.app.GetTargetByID(c.Request().Context(), user.ID, tID)
	if err != nil {
		return fmt.Errorf("get target by id: %w", err)
	}

	formURL := "/servers/" + srv.ID.S()

	return Render(c, http.StatusOK, templates.PageTarget(user, *srv, formURL, ``, nil, nil))
}

func (s *Service) getAuth(c echo.Context) error {
	switch s.opts.app.AuthType() {
	default:
		// TODO: choose a better way to show an error
		return fmt.Errorf("unknown auth type: %s", s.opts.app.AuthType()) //nolint:err113
	case config.AuthTypeConfig:
		return Render(c, http.StatusOK, templates.PageAuth(nil))
	case config.AuthTypeOIDC:
		authURL, err := s.opts.app.InitOIDC(c.Request().Context())
		if err != nil {
			return fmt.Errorf("init oidc: %w", err)
		}

		return c.Redirect(http.StatusSeeOther, authURL)
	}
}

func (s *Service) getAuthCallback(c echo.Context) error {
	if s.opts.app.AuthType() != config.AuthTypeOIDC {
		return errors.New("not available") //nolint:err113
	}

	code := c.Request().URL.Query().Get("code")

	user, expiry, err := s.opts.app.CompleteOIDC(c.Request().Context(), code)
	if err != nil {
		return fmt.Errorf("complete oidc: %w", err)
	}

	sess, err := session.Get(keySession, c)
	if err != nil {
		return fmt.Errorf("have no session: %w", err)
	}

	sess.Options = &sessions.Options{ //nolint:exhaustruct
		Path:     "/",
		MaxAge:   int(time.Until(expiry).Seconds()),
		HttpOnly: true,
	}
	sess.Values[keyUserID] = *user
	if err := sess.Save(c.Request(), c.Response()); err != nil {
		return fmt.Errorf("save session: %w", err)
	}

	return c.Redirect(http.StatusSeeOther, "/")
}

func (s *Service) postAuth(c echo.Context) error {
	if s.opts.app.AuthType() != config.AuthTypeConfig {
		// TODO: choose a better way to show an error
		return fmt.Errorf("unknown auth type: %s", s.opts.app.AuthType()) //nolint:err113
	}

	ctx := c.Request().Context()

	user, err := s.opts.app.AuthUser(ctx, c.FormValue("username"), c.FormValue("password"))
	if err != nil {
		return Render(c, http.StatusOK, templates.PageAuth(err))
	}

	sess, err := session.Get(keySession, c)
	if err != nil {
		return fmt.Errorf("have no session: %w", err)
	}

	sess.Options = &sessions.Options{ //nolint:exhaustruct
		Path:     "/",
		MaxAge:   int(time.Hour.Seconds()),
		HttpOnly: true,
	}
	sess.Values[keyUserID] = *user
	if err := sess.Save(c.Request(), c.Response()); err != nil {
		return fmt.Errorf("save session: %w", err)
	}

	return c.Redirect(http.StatusSeeOther, "/")
}

func (*Service) logout(c echo.Context) error {
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
	if err := sess.Save(c.Request(), c.Response()); err != nil {
		return fmt.Errorf("save session: %w", err)
	}

	return c.Redirect(http.StatusSeeOther, "/")
}

func (s *Service) runQuery(c echo.Context) error {
	user := c.Get(ctxUser).(structs.User) //nolint:forcetypeassert

	tID := config.TargetID(c.Param("id"))
	srv, err := s.opts.app.GetTargetByID(c.Request().Context(), user.ID, tID)
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

	queryID, _, err := s.opts.app.RunQuery(c.Request().Context(), user.ID, srv.ID, query)
	if err != nil {
		return Render(c, http.StatusOK, templates.PageTarget(user, *srv, formURL, query, nil, err)) //nolint:err113
	}

	params2 := url.Values{}
	params2.Set("format", format)
	targetURL := fmt.Sprintf("/servers/%s/%s?%s", srv.ID, queryID.S(), params2.Encode())

	return c.Redirect(http.StatusSeeOther, targetURL)
}

func (s *Service) getQueryResults(c echo.Context) error {
	user := c.Get(ctxUser).(structs.User) //nolint:forcetypeassert

	tID := config.TargetID(c.Param("id"))
	srv, err := s.opts.app.GetTargetByID(c.Request().Context(), user.ID, tID)
	if err != nil {
		return fmt.Errorf("get target by id: %w", err)
	}

	qID := uuid6.FromStr(c.Param("qid"))
	qRes, err := s.opts.app.GetQueryResults(c.Request().Context(), user.ID, qID)
	if err != nil {
		return fmt.Errorf("get query results: %w", err)
	}

	params, err := c.FormParams()
	if err != nil {
		return c.Redirect(http.StatusSeeOther, "/")
	}

	format := params.Get("format")
	formURL := "/servers/" + srv.ID.S()

	switch format {
	default:
		correctedURL := c.Request().URL.Path + "?format=html"

		return c.Redirect(http.StatusSeeOther, correctedURL)
	case "html":
		return Render(c, http.StatusOK, templates.PageTarget(user, *srv, formURL, qRes.Query, &qRes.QTable, nil))
	case "json":
		qTbl2 := just.SliceMap(qRes.QTable.Rows, func(row []string) map[string]any {
			m := make(map[string]any, len(qRes.QTable.Headers))
			for i := range qRes.QTable.Headers {
				m[qRes.QTable.Headers[i]] = row[i]
			}

			return m
		})

		resBuf, err := json.Marshal(qTbl2)
		if err != nil {
			return Render(c, http.StatusOK, templates.PageTarget(user, *srv, formURL, qRes.Query, nil, err))
		}

		c.Response().Header().Set(echo.HeaderContentDisposition, fmt.Sprintf(`%s; filename="%s"`, "attachment", "response.json")) //nolint
		http.ServeContent(c.Response(), c.Request(), "response.json", time.Now(), bytes.NewReader(resBuf))

		return nil
	}
}

func Render(ctx echo.Context, statusCode int, t templ.Component) error {
	buf := templ.GetBuffer()
	defer templ.ReleaseBuffer(buf)

	if err := t.Render(ctx.Request().Context(), buf); err != nil {
		return err
	}

	return ctx.HTML(statusCode, buf.String())
}
