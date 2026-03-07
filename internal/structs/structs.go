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

package structs

import "github.com/kazhuravlev/database-gateway/internal/config"

type Tag struct {
	Name string
	// Color string
}

type Server struct {
	ID          config.TargetID
	Description string
	Tags        []Tag
	Type        string
	Tables      []config.TargetTable
}

type QTable struct {
	Headers []string   `json:"headers"`
	Rows    [][]string `json:"rows"`
}

type QMeta struct {
	ExecutionTimeMS    int64 `json:"execution_time_ms"`
	ParsingTimeMS      int64 `json:"parsing_time_ms"`
	NetworkRoundTripMS int64 `json:"network_round_trip_ms"`
	RowsCount          int   `json:"rows_count,omitempty"`
	ColumnsCount       int   `json:"columns_count,omitempty"`
	VectorsCount       int   `json:"vectors_count,omitempty"`
}

type User struct {
	ID       config.UserID
	Username string
}

type Bookmark struct {
	ID       string
	TargetID config.TargetID
	Title    string
	Query    string
}
