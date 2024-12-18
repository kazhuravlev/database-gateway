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

package uuid6

import (
	"bytes"
	"database/sql/driver"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
)

var nilUUID = ulid.ULID{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0} //nolint:gochecknoglobals

type UUID ulid.ULID //nolint:recvcheck

func New() UUID {
	return UUID(ulid.Make())
}

func Nil() UUID {
	return UUID(nilUUID)
}

func (u UUID) S() string {
	return ulid.ULID(u).String()
}

func (u UUID) IsNil() bool {
	return ulid.ULID(u).Compare(nilUUID) == 0
}

func (u UUID) String() string {
	return u.S()
}

// Scan implements the Scanner interface.
func (u *UUID) Scan(value any) error {
	if value == nil {
		return errors.New("bad ulid") //nolint:goerr113
	}

	switch val := value.(type) {
	default:
		return errors.New("bad input type") //nolint:goerr113
	case []byte:
		if err := scanTo(val, u); err != nil {
			return fmt.Errorf("scan uuid6 []byte: %w", err)
		}

		return nil
	case string:
		if err := scanTo([]byte(val), u); err != nil {
			return fmt.Errorf("scan uuid6 str: %w", err)
		}

		return nil
	}
}

func scanTo(buf []byte, id *UUID) error {
	if bytes.Equal(buf, nilUUID.Bytes()) {
		*id = UUID(nilUUID)

		return nil
	}

	uid, err := uuid.Parse(string(buf))
	if err != nil {
		return fmt.Errorf("parse uuid: %w", err)
	}

	*id = FromUUID(uid)

	return nil
}

// Value implements the driver Valuer interface.
func (u UUID) Value() (driver.Value, error) {
	return u.ToUUID().Value() //nolint:wrapcheck
}

func (u UUID) ToUUID() uuid.UUID {
	var id uuid.UUID
	copy(id[:], u[:])

	return id
}

func FromUUID(id uuid.UUID) UUID {
	var id2 ulid.ULID
	copy(id2[:], id[:])

	return UUID(id2)
}

func FromStr(id string) UUID {
	return UUID(ulid.MustParse(id))
}

func ParseStr(in string) (UUID, error) {
	id, err := ulid.Parse(in)
	if err != nil {
		return UUID{}, fmt.Errorf("parse ulid: %w", err)
	}

	return UUID(id), nil
}
