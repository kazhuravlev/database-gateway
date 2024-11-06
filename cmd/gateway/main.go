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

package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/kazhuravlev/database-gateway/internal/app"
	"github.com/kazhuravlev/database-gateway/internal/facade"

	"github.com/kazhuravlev/database-gateway/internal/config"
	"github.com/kazhuravlev/just"
	_ "github.com/lib/pq"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := runApp(ctx); err != nil {
		panic(err)
	}
}

func runApp(ctx context.Context) error {
	const configFilename = "config.json"
	cfg, err := just.JsonParseTypeF[config.Config](configFilename)
	if err != nil {
		return fmt.Errorf("parse config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("validate config: %w", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		AddSource:   false,
		Level:       nil,
		ReplaceAttr: nil,
	}))

	appInst, err := app.New(app.NewOptions(logger, *cfg))
	if err != nil {
		return fmt.Errorf("create app: %w", err)
	}

	fInst, err := facade.New(facade.NewOptions(logger, appInst, cfg.Facade.CookieSecret, cfg.Facade.Port))
	if err != nil {
		return fmt.Errorf("create facade: %w", err)
	}

	if err := fInst.Run(ctx); err != nil {
		return fmt.Errorf("run facade: %w", err)
	}

	return nil
}
