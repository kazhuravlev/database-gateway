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
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/kazhuravlev/database-gateway/internal/structs"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"

	"github.com/kazhuravlev/database-gateway/internal/facade/static"

	"github.com/a-h/templ"

	"github.com/kazhuravlev/database-gateway/internal/config"
	"github.com/kazhuravlev/database-gateway/internal/facade/templates"
	"github.com/kazhuravlev/just"
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
	gob.Register(structs.User{})
	gob.Register(config.UserID(""))

	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("bad configuration: %w", err)
	}

	return &Service{opts: opts}, nil
}

func (s *Service) Run(ctx context.Context) error {
	echoInst := echo.New()
	echoInst.HideBanner = true
	echoInst.Use(middleware.Recover())
	echoInst.Use(middleware.Logger())
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

	echoInst.GET("/auth", s.getAuth)
	echoInst.GET("/logout", s.logout)

	echoInst.GET("/*", func(c echo.Context) error {
		return c.Redirect(http.StatusSeeOther, "/")
	})

	echoInst.Logger.Fatal(echoInst.Start(":8080"))

	return nil
}

func (s *Service) getServers(c echo.Context) error {
	user := c.Get(ctxUser).(structs.User) //nolint:forcetypeassert

	servers, err := s.opts.app.GetTargets(c.Request().Context())
	if err != nil {
		s.opts.logger.Error("get targets", slog.String("error", err.Error()))
		return c.String(http.StatusInternalServerError, "the sky was falling")
	}

	return Render(c, http.StatusOK, templates.PageTargetsList(user, servers))
}

func (s *Service) getServer(c echo.Context) error {
	user := c.Get(ctxUser).(structs.User) //nolint:forcetypeassert

	tID := config.TargetID(c.Param("id"))
	srv, err := s.opts.app.GetTargetByID(c.Request().Context(), tID)
	if err != nil {
		return fmt.Errorf("get target by id: %w", err)
	}

	acls := s.opts.app.GetACLs(c.Request().Context(), user.ID, tID)

	return Render(c, http.StatusOK, templates.PageTarget(user, *srv, acls, ``, nil, nil))
}

func (s *Service) getAuth(c echo.Context) error {
	switch s.opts.app.AuthType() {
	default:
		return fmt.Errorf("unknown auth type: %s", s.opts.app.AuthType())
	case config.AuthTypeConfig:
		return Render(c, http.StatusOK, templates.PageAuth(nil))
	case config.AuthTypeOIDC:
		panic("implement me")
		return nil
	}
}

func (s *Service) logout(c echo.Context) error {
	sess, err := session.Get(keySession, c)
	if err != nil {
		return c.Redirect(http.StatusSeeOther, "/")
	}

	sess.Options = &sessions.Options{
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
	srv, err := s.opts.app.GetTargetByID(c.Request().Context(), tID)
	if err != nil {
		return fmt.Errorf("get target by id: %w", err)
	}

	params, err := c.FormParams()
	if err != nil {
		return c.Redirect(http.StatusSeeOther, "/")
	}

	query := params.Get("query")
	format := params.Get("format")

	acls := s.opts.app.GetACLs(c.Request().Context(), user.ID, tID)

	qTbl, err := s.opts.app.RunQuery(c.Request().Context(), user.ID, srv.ID, query)
	if err != nil {
		return Render(c, http.StatusOK, templates.PageTarget(user, *srv, acls, query, nil, err)) //nolint:err113
	}

	switch format {
	default:
		return Render(c, http.StatusOK, templates.PageTarget(user, *srv, acls, query, nil, errors.New("unknown format"))) //nolint:err113
	case "html":
		return Render(c, http.StatusOK, templates.PageTarget(user, *srv, acls, query, qTbl, nil))
	case "json":
		qTbl := just.SliceMap(qTbl.Rows, func(row []string) map[string]any {
			m := make(map[string]any, len(qTbl.Headers))
			for i := range qTbl.Headers {
				m[qTbl.Headers[i]] = row[i]
			}

			return m
		})

		resBuf, err := json.Marshal(qTbl)
		if err != nil {
			return Render(c, http.StatusOK, templates.PageTarget(user, *srv, acls, query, nil, err))
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
