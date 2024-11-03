package main

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
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/k0kubun/pp/v3"
	"github.com/kazhuravlev/database-gateway/internal/config"
	"github.com/kazhuravlev/database-gateway/internal/structs"
	"github.com/kazhuravlev/database-gateway/internal/templates"
	"github.com/kazhuravlev/database-gateway/internal/validator"
	"github.com/kazhuravlev/just"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	_ "github.com/lib/pq"
)

const ctxUser = "k-user"

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := runApp(ctx); err != nil {
		panic(err)
	}
}

type Targets struct {
	id2client map[string]*pgxpool.Pool
}

func Render(ctx echo.Context, statusCode int, t templ.Component) error {
	buf := templ.GetBuffer()
	defer templ.ReleaseBuffer(buf)

	if err := t.Render(ctx.Request().Context(), buf); err != nil {
		return err
	}

	return ctx.HTML(statusCode, buf.String())
}

func runApp(ctx context.Context) error { //nolint:gocognit,gocyclo,cyclop,maintidx
	const configFilename = "config.json"
	cfg, err := just.JsonParseTypeF[config.Config](configFilename)
	if err != nil {
		return fmt.Errorf("parse config: %w", err)
	}

	pp.Println(cfg)

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("validate config: %w", err)
	}

	targets := &Targets{
		id2client: make(map[string]*pgxpool.Pool),
	}

	for i := range cfg.Targets {
		target := &cfg.Targets[i]
		slog.Info("connect to target", slog.String("target", target.ID))
		pgCfg := target.Connection

		urlExample := fmt.Sprintf(
			"postgres://%s:%s@%s:%d/%s?sslmode=%s",
			pgCfg.User,
			pgCfg.Password,
			pgCfg.Host,
			pgCfg.Port,
			pgCfg.DB,
			just.If(pgCfg.UseSSL, "enable", "disable"),
		)
		dbpool, err := pgxpool.New(ctx, urlExample)
		if err != nil {
			return fmt.Errorf("create db pool: %w", err)
		}
		defer dbpool.Close() //nolint:gocritic // this is ok

		targets.id2client[target.ID] = dbpool
	}

	authUser := func(u, p string) (config.User, bool) {
		for _, user := range cfg.Users {
			if user.Username == u && user.Password == p {
				return user, true
			}
		}

		return config.User{}, false //nolint:exhaustruct
	}

	getTarget := func(id string) (config.Target, *pgxpool.Pool, bool) {
		for i := range cfg.Targets {
			target := cfg.Targets[i]
			if target.ID == id {
				return target, targets.id2client[id], true
			}
		}

		return config.Target{}, nil, false //nolint:exhaustruct
	}

	echoInst := echo.New()
	echoInst.HideBanner = true
	echoInst.Use(middleware.BasicAuthWithConfig(middleware.BasicAuthConfig{
		Skipper: func(_ echo.Context) bool {
			return false
		},
		Validator: func(username, password string, c echo.Context) (bool, error) {
			if user, ok := authUser(username, password); ok {
				c.Set(ctxUser, user)

				return true, nil
			}

			return false, nil
		},
		Realm: "",
	}))

	echoInst.GET("/", func(c echo.Context) error {
		return c.Redirect(http.StatusTemporaryRedirect, "/servers")
	})
	echoInst.GET("/servers", func(c echo.Context) error { //nolint:contextcheck
		user := c.Get(ctxUser).(config.User) //nolint:forcetypeassert

		servers := just.SliceMap(cfg.Targets, func(t config.Target) structs.Server {
			return structs.Server{
				ID:     t.ID,
				Type:   t.Type,
				Tables: t.Tables,
			}
		})

		return Render(c, http.StatusOK, templates.PageTargetsList(user, servers))
	})
	echoInst.GET("/servers/:id", func(c echo.Context) error { //nolint:contextcheck
		user := c.Get(ctxUser).(config.User) //nolint:forcetypeassert

		srv, _, ok := getTarget(c.Param("id"))
		if !ok {
			return c.Redirect(http.StatusTemporaryRedirect, "/")
		}

		acls := just.SliceFilter(user.Acls, func(acl config.ACL) bool {
			return acl.Target == srv.ID
		})

		return Render(c, http.StatusOK, templates.PageTarget(user, srv, acls, ``, nil, nil))
	})
	echoInst.POST("/servers/:id", func(c echo.Context) error { //nolint:contextcheck
		user := c.Get(ctxUser).(config.User) //nolint:forcetypeassert

		srv, conn, ok := getTarget(c.Param("id"))
		if !ok {
			return c.Redirect(http.StatusTemporaryRedirect, "/")
		}

		params, err := c.FormParams()
		if err != nil {
			return c.Redirect(http.StatusTemporaryRedirect, "/")
		}

		query := params.Get("query")
		format := params.Get("format")

		acls := just.SliceFilter(user.Acls, func(acl config.ACL) bool {
			return acl.Target == srv.ID
		})

		if err := validator.IsAllowed(srv.Tables, acls, query); err != nil {
			log.Error("err", err.Error())

			return Render(c, http.StatusOK, templates.PageTarget(user, srv, acls, query, nil, err))
		}

		res, err := conn.Query(c.Request().Context(), query)
		if err != nil {
			var connErr *pgconn.ConnectError
			if errors.As(err, &connErr) {
				err2 := errors.New("failed to connect: target not available") //nolint:err113

				return Render(c, http.StatusOK, templates.PageTarget(user, srv, acls, query, nil, err2))
			}

			return Render(c, http.StatusOK, templates.PageTarget(user, srv, acls, query, nil, err))
		}

		rows, err := pgx.CollectRows(res, func(row pgx.CollectableRow) ([]any, error) {
			return row.Values()
		})
		if err != nil {
			return Render(c, http.StatusOK, templates.PageTarget(user, srv, acls, query, nil, err))
		}

		cols := just.SliceMap(res.FieldDescriptions(), func(fd pgconn.FieldDescription) string {
			return fd.Name
		})

		switch format {
		default:
			return Render(c, http.StatusOK, templates.PageTarget(user, srv, acls, query, nil, errors.New("unknown format"))) //nolint:err113
		case "html":
			qTbl := structs.QTable{
				Headers: cols,
				Rows: just.SliceMap(rows, func(row []any) []string {
					return just.SliceMap(row, func(v any) string {
						return fmt.Sprint(v)
					})
				}),
			}

			return Render(c, http.StatusOK, templates.PageTarget(user, srv, acls, query, &qTbl, nil))
		case "json":
			qTbl := just.SliceMap(rows, func(row []any) map[string]any {
				m := make(map[string]any, len(cols))
				for i := range cols {
					m[cols[i]] = row[i]
				}

				return m
			})

			resBuf, err := json.Marshal(qTbl)
			if err != nil {
				return Render(c, http.StatusOK, templates.PageTarget(user, srv, acls, query, nil, err))
			}

			c.Response().Header().Set(echo.HeaderContentDisposition, fmt.Sprintf(`%q; filename="%q"`, "attachment", "response.json"))
			http.ServeContent(c.Response(), c.Request(), "response.json", time.Now(), bytes.NewReader(resBuf))

			return nil
		}
	})

	{ //nolint:gocritic
		echoInst.GET("/*", func(c echo.Context) error {
			return c.Redirect(http.StatusTemporaryRedirect, "/")
		})

		echoInst.Logger.Fatal(echoInst.Start(":8080"))
	}

	return nil
}
