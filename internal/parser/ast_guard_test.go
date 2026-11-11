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

package parser //nolint:testpackage // Exercises the internal AST guard with mutated protobuf messages.

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"

	pg "github.com/pganalyze/pg_query_go/v6"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

// Any upstream AST field/node/enum change needs security review, including fields
// whose default value would not appear in a particular SQL regression fixture.
func TestReviewedASTSchema(t *testing.T) {
	t.Parallel()
	descriptor := protodesc.ToFileDescriptorProto((&pg.ParseResult{}).ProtoReflect().Descriptor().ParentFile())
	data, err := proto.MarshalOptions{Deterministic: true}.Marshal(descriptor)
	require.NoError(t, err)
	require.Equal(t, "0b651af4b438c97f6f68108fec5756ef138059db69252aab308876ed3bbe4360", fmt.Sprintf("%x", sha256.Sum256(data)), "Review AST changes and the allowlist before updating this digest")
}

func TestEveryUnlistedASTFieldIsRejected(t *testing.T) { //nolint:cyclop // Enumerates every protobuf field shape.
	t.Parallel()
	file := (&pg.ParseResult{}).ProtoReflect().Descriptor().ParentFile()
	for name, allowed := range astFields {
		desc := file.Messages().ByName(name)
		require.NotNil(t, desc)
		for fieldName := range strings.FieldsSeq(allowed) {
			require.NotNil(t, desc.Fields().ByName(protoreflect.Name(fieldName)))
		}
		for i := range desc.Fields().Len() {
			field := desc.Fields().Get(i)
			if strings.Contains(" "+allowed+" ", " "+string(field.Name())+" ") {
				continue
			}
			t.Run(string(field.FullName()), func(t *testing.T) {
				t.Parallel()
				message := dynamicpb.NewMessage(desc)
				if field.IsList() {
					list := message.Mutable(field).List()
					list.Append(list.NewElement())
				} else {
					value := message.NewField(field)
					switch field.Kind() { //nolint:exhaustive // Unhandled message fields use their populated default.
					case protoreflect.BoolKind:
						value = protoreflect.ValueOfBool(true)
					case protoreflect.StringKind:
						value = protoreflect.ValueOfString("unreviewed")
					case protoreflect.Int32Kind:
						value = protoreflect.ValueOfInt32(1)
					case protoreflect.Uint32Kind:
						value = protoreflect.ValueOfUint32(1)
					case protoreflect.EnumKind:
						value = protoreflect.ValueOfEnum(1)
					}
					message.Set(field, value)
				}
				require.ErrorIs(t, validateNodeFields(message), ErrNotImplemented)
			})
		}
	}
}

func TestRejectUnknownWireFieldsAndEnums(t *testing.T) {
	t.Parallel()
	tree, err := pg.Parse("SELECT id FROM clients")
	require.NoError(t, err)
	tree.GetStmts()[0].ProtoReflect().SetUnknown([]byte{0xa0, 0x06, 0x01})
	require.ErrorIs(t, validateAST(tree), ErrNotImplemented)
	tree.GetStmts()[0].ProtoReflect().SetUnknown(nil)
	tree.GetStmts()[0].GetStmt().GetSelectStmt().LimitOption = 999
	require.ErrorIs(t, validateAST(tree), ErrNotImplemented)
}

func TestCoverageRejectsOmittedDependencies(t *testing.T) {
	t.Parallel()
	tree, err := parseTree("SELECT c.id FROM clients c JOIN secrets s ON c.id=s.id WHERE s.secret=1")
	require.NoError(t, err)
	require.ErrorIs(t, checkCoverage(tree, []Vector{SelectVec{Tbl: "clients", Target: []string{"id"}}}), ErrNotImplemented)
	require.ErrorIs(t, checkCoverage(tree, []Vector{SelectVec{Tbl: "clients", Target: []string{"id"}}, SelectVec{Tbl: "secrets", Target: []string{"id"}}}), ErrNotImplemented)
}
