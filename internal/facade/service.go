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
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/a-h/templ"

	"github.com/kazhuravlev/database-gateway/internal/config"
	"github.com/kazhuravlev/database-gateway/internal/templates"
	"github.com/kazhuravlev/just"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

const ctxUser = "c-user"

type Service struct {
	opts Options
}

func New(opts Options) (*Service, error) {
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
	echoInst.Use(middleware.BasicAuthWithConfig(middleware.BasicAuthConfig{
		Skipper: func(_ echo.Context) bool {
			return false
		},
		Validator: func(username, password string, c echo.Context) (bool, error) {
			id, err := s.opts.app.AuthUser(c.Request().Context(), username, password)
			if err != nil {
				return false, fmt.Errorf("not authenticated: %w", err)
			}

			user, err := s.opts.app.GetUserByUsername(c.Request().Context(), id)
			if err != nil {
				return false, fmt.Errorf("get user by id: %w", err)
			}

			c.Set(ctxUser, *user)

			return true, nil
		},
		Realm: "",
	}))

	echoInst.GET("/", func(c echo.Context) error {
		return c.Redirect(http.StatusTemporaryRedirect, "/servers")
	})
	echoInst.GET("/servers", s.getServers)
	echoInst.GET("/servers/:id", s.getServer)
	echoInst.POST("/servers/:id", s.runQuery)

	echoInst.GET("/*", func(c echo.Context) error {
		return c.Redirect(http.StatusTemporaryRedirect, "/")
	})

	echoInst.Logger.Fatal(echoInst.Start(":8080"))

	return nil
}

func (s *Service) getServers(c echo.Context) error {
	user := c.Get(ctxUser).(config.User) //nolint:forcetypeassert

	servers, err := s.opts.app.GetTargets(c.Request().Context())
	if err != nil {
		s.opts.logger.Error("get targets", slog.String("error", err.Error()))
		return c.String(http.StatusInternalServerError, "the sky was falling")
	}

	return Render(c, http.StatusOK, templates.PageTargetsList(user, servers))
}

func (s *Service) getServer(c echo.Context) error {
	user := c.Get(ctxUser).(config.User) //nolint:forcetypeassert

	srv, err := s.opts.app.GetTargetByID(c.Request().Context(), c.Param("id"))
	if err != nil {
		return fmt.Errorf("get target by id: %w", err)
	}

	acls := just.SliceFilter(user.Acls, func(acl config.ACL) bool {
		return acl.Target == srv.ID
	})

	return Render(c, http.StatusOK, templates.PageTarget(user, *srv, acls, ``, nil, nil))
}

func (s *Service) runQuery(c echo.Context) error {
	user := c.Get(ctxUser).(config.User) //nolint:forcetypeassert

	srv, err := s.opts.app.GetTargetByID(c.Request().Context(), c.Param("id"))
	if err != nil {
		return fmt.Errorf("get target by id: %w", err)
	}

	params, err := c.FormParams()
	if err != nil {
		return c.Redirect(http.StatusTemporaryRedirect, "/")
	}

	query := params.Get("query")
	format := params.Get("format")

	qTbl, err := s.opts.app.RunQuery(c.Request().Context(), user.Username, srv.ID, query)
	if err != nil {
		return fmt.Errorf("run query: %w", err)
	}

	acls := just.SliceFilter(user.Acls, func(acl config.ACL) bool {
		return acl.Target == srv.ID
	})

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