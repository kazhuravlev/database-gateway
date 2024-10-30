package main

import (
	"context"
	"fmt"
	"github.com/kazhuravlev/database-gateway/internal/structs"
	"github.com/kazhuravlev/database-gateway/templates"
	"log/slog"
	"net/http"

	"github.com/a-h/templ"
	"github.com/jackc/pgx/v5"
	"github.com/k0kubun/pp/v3"
	"github.com/kazhuravlev/database-gateway/internal/config"
	"github.com/kazhuravlev/just"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "github.com/lib/pq"
)

var ctxUser = "k-user"

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := runApp(ctx); err != nil {
		panic(err)
	}
}

type Targets struct {
	id2client map[string]*pgx.Conn
}

func Render(ctx echo.Context, statusCode int, t templ.Component) error {
	buf := templ.GetBuffer()
	defer templ.ReleaseBuffer(buf)

	if err := t.Render(ctx.Request().Context(), buf); err != nil {
		return err
	}

	return ctx.HTML(statusCode, buf.String())
}

func runApp(ctx context.Context) error {
	const configFilename = "config.json"
	cfg, err := just.JsonParseTypeF[config.Config](configFilename)
	if err != nil {
		return fmt.Errorf("parse config: %w", err)
	}

	pp.Println(cfg)

	targets := &Targets{
		id2client: make(map[string]*pgx.Conn),
	}

	for _, target := range cfg.Targets {
		slog.Info("connect to target", slog.String("target", target.Id))
		c := target.Connection
		urlExample := fmt.Sprintf("postgres://%s:%s@%s:%d/%s", c.User, c.Password, c.Host, c.Port, c.Db)
		conn, err := pgx.Connect(ctx, urlExample)
		if err != nil {
			return fmt.Errorf("connect to (%s): %w", target.Id, err)
		}

		defer conn.Close(context.Background())

		targets.id2client[target.Id] = conn
	}

	authUser := func(u, p string) (config.User, bool) {
		for _, user := range cfg.Users {
			if user.Username == u && user.Password == p {
				return user, true
			}
		}

		return config.User{}, false
	}

	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.BasicAuthWithConfig(middleware.BasicAuthConfig{
		Skipper: func(c echo.Context) bool {
			return false
		},
		Validator: func(username, password string, c echo.Context) (bool, error) {
			if _, ok := authUser(username, password); ok {

				user := structs.User{
					Username: username,
				}

				c.Set(ctxUser, user)
				return true, nil
			}

			return false, nil
		},
		Realm: "",
	}))

	e.GET("/", func(c echo.Context) error {
		return c.Redirect(http.StatusTemporaryRedirect, "/servers")
	})
	e.GET("/servers", func(c echo.Context) error {
		user := c.Get(ctxUser).(structs.User)

		servers := just.SliceMap(cfg.Targets, func(t config.Target) structs.Server {
			return structs.Server{
				ID:     t.Id,
				Type:   t.Type,
				Tables: t.Tables,
			}
		})

		return Render(c, http.StatusOK, templates.PageServersList(user, servers))
	})

	{
		e.GET("/*", func(c echo.Context) error {
			return c.Redirect(http.StatusTemporaryRedirect, "/")
		})

		e.Logger.Fatal(e.Start(":8080"))
	}

	return nil
}
