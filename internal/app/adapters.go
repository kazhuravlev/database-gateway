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

package app

import (
	"bytes"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/kazhuravlev/database-gateway/internal/config"
	"github.com/kazhuravlev/database-gateway/internal/structs"
	"github.com/kazhuravlev/just"
)

func adaptPgType(val any) string {
	switch val := val.(type) {
	default:
		return fmt.Sprint(val)
	case pgtype.Numeric:
		// TODO: is that really best solution?
		res, err := val.MarshalJSON()
		if err != nil {
			return "--bad payload--"
		}

		return string(bytes.Trim(res, `"`))
	}
}

func adaptTarget(target config.Target) structs.Server { //nolint:gocritic
	return structs.Server{
		ID:          target.ID,
		Description: target.Description,
		Tags:        adaptTags(target.Tags),
		Type:        target.Type,
		Tables:      target.Tables,
	}
}

func adaptTags(tags []string) []structs.Tag {
	return just.SliceMap(tags, func(t string) structs.Tag {
		return structs.Tag{
			Name: t,
			// Color: colorful.HappyColor().Hex(),
		}
	})
}
