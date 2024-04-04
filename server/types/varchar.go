// Copyright 2024 Dolthub, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package types

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"reflect"
	"unicode/utf8"

	"github.com/lib/pq/oid"

	"github.com/dolthub/go-mysql-server/sql"
	"github.com/dolthub/go-mysql-server/sql/types"
	"github.com/dolthub/vitess/go/sqltypes"
	"github.com/dolthub/vitess/go/vt/proto/query"
)

const (
	// VarCharMaxLength is the maximum number of characters (not bytes) that a VarChar may contain.
	VarCharMaxLength = 10485760
	// varCharInline is the maximum number of characters (not bytes) that are "guaranteed" to fit inline.
	varCharInline = 16383
)

// VarCharInline is a varchar that has the max inline length automatically set.
// TODO: probably delete me
var VarCharInline = VarCharType{Length: varCharInline}

// VarCharMax is a varchar that has the max length.
// TODO: rename me
var VarCharMax = VarCharType{Length: VarCharMaxLength}

// VarCharType is the extended type implementation of the PostgreSQL varchar.
type VarCharType struct {
	Length uint32
}

var _ DoltgresType = VarCharType{}

// BaseID implements the DoltgresType interface.
func (b VarCharType) BaseID() DoltgresTypeBaseID {
	return DoltgresTypeBaseID(SerializationID_VarChar)
}

// CollationCoercibility implements the DoltgresType interface.
func (b VarCharType) CollationCoercibility(ctx *sql.Context) (collation sql.CollationID, coercibility byte) {
	return sql.Collation_binary, 5
}

// Compare implements the DoltgresType interface.
func (b VarCharType) Compare(v1 any, v2 any) (int, error) {
	if v1 == nil && v2 == nil {
		return 0, nil
	} else if v1 != nil && v2 == nil {
		return 1, nil
	} else if v1 == nil && v2 != nil {
		return -1, nil
	}

	ac, _, err := b.Convert(v1)
	if err != nil {
		return 0, err
	}
	bc, _, err := b.Convert(v2)
	if err != nil {
		return 0, err
	}

	ab := ac.(string)
	bb := bc.(string)
	if ab == bb {
		return 0, nil
	} else if ab < bb {
		return -1, nil
	} else {
		return 1, nil
	}
}

// Convert implements the DoltgresType interface.
func (b VarCharType) Convert(val any) (any, sql.ConvertInRange, error) {
	// TODO: need to check if this always truncates for values that are too large, or if it's just the default behavior
	switch val := val.(type) {
	case string:
		// First we'll do a byte-length check since it's always >= the rune-count check, and it's far faster
		if uint32(len(val)) > b.Length {
			// The byte-length is greater, so now we'll do a rune-count
			if uint32(utf8.RuneCountInString(val)) > b.Length {
				// TODO: figure out if there's a faster way to truncate based on rune count
				startString := val
				for i := uint32(0); i < b.Length; i++ {
					_, size := utf8.DecodeRuneInString(val)
					val = val[size:]
				}
				return startString[:len(startString)-len(val)], sql.InRange, nil
			}
		}
		return val, sql.InRange, nil
	case []byte:
		// First we'll do a byte-length check since it's always >= the rune-count check, and it's far faster
		if uint32(len(val)) > b.Length {
			// The byte-length is greater, so now we'll do a rune-count
			if uint32(utf8.RuneCount(val)) > b.Length {
				// TODO: figure out if there's a faster way to truncate based on rune count
				startBytes := val
				for i := uint32(0); i < b.Length; i++ {
					_, size := utf8.DecodeRune(val)
					val = val[size:]
				}
				return string(startBytes[:len(startBytes)-len(val)]), sql.InRange, nil
			}
		}
		return string(val), sql.InRange, nil
	case nil:
		return nil, sql.InRange, nil
	default:
		return nil, sql.OutOfRange, sql.ErrInvalidType.New(b)
	}
}

// Equals implements the DoltgresType interface.
func (b VarCharType) Equals(otherType sql.Type) bool {
	if otherExtendedType, ok := otherType.(types.ExtendedType); ok {
		return bytes.Equal(MustSerializeType(b), MustSerializeType(otherExtendedType))
	}
	return false
}

// FormatSerializedValue implements the DoltgresType interface.
func (b VarCharType) FormatSerializedValue(val []byte) (string, error) {
	deserialized, err := b.DeserializeValue(val)
	if err != nil {
		return "", err
	}
	return b.FormatValue(deserialized)
}

// FormatValue implements the DoltgresType interface.
func (b VarCharType) FormatValue(val any) (string, error) {
	if val == nil {
		return "", nil
	}
	converted, _, err := b.Convert(val)
	if err != nil {
		return "", err
	}
	return converted.(string), nil
}

// MaxSerializedWidth implements the DoltgresType interface.
func (b VarCharType) MaxSerializedWidth() types.ExtendedTypeSerializedWidth {
	if b.Length <= varCharInline {
		return types.ExtendedTypeSerializedWidth_64K
	} else {
		return types.ExtendedTypeSerializedWidth_Unbounded
	}
}

// MaxTextResponseByteLength implements the DoltgresType interface.
func (b VarCharType) MaxTextResponseByteLength(ctx *sql.Context) uint32 {
	return b.Length * 4
}

// OID implements the DoltgresType interface.
func (b VarCharType) OID() uint32 {
	return uint32(oid.T_varchar)
}

// Promote implements the DoltgresType interface.
func (b VarCharType) Promote() sql.Type {
	return VarCharMax
}

// SerializedCompare implements the DoltgresType interface.
func (b VarCharType) SerializedCompare(v1 []byte, v2 []byte) (int, error) {
	if len(v1) == 0 && len(v2) == 0 {
		return 0, nil
	} else if len(v1) > 0 && len(v2) == 0 {
		return 1, nil
	} else if len(v1) == 0 && len(v2) > 0 {
		return -1, nil
	}

	//TODO: can we byte-compare unicode strings like this?
	return bytes.Compare(v1, v2), nil
}

// SerializeType implements the DoltgresType interface.
func (b VarCharType) SerializeType() ([]byte, error) {
	t := make([]byte, 6)
	copy(t, SerializationID_VarChar.ToByteSlice())
	binary.LittleEndian.PutUint32(t[2:], b.Length)
	return t, nil
}

// SQL implements the DoltgresType interface.
func (b VarCharType) SQL(ctx *sql.Context, dest []byte, v any) (sqltypes.Value, error) {
	if v == nil {
		return sqltypes.NULL, nil
	}
	value, err := b.FormatValue(v)
	if err != nil {
		return sqltypes.Value{}, err
	}
	return sqltypes.MakeTrusted(sqltypes.Text, types.AppendAndSliceBytes(dest, []byte(value))), nil
}

// String implements the DoltgresType interface.
func (b VarCharType) String() string {
	if b.Length == VarCharMaxLength {
		return "varchar"
	}
	return fmt.Sprintf("varchar(%d)", b.Length)
}

// ToArrayType implements the DoltgresType interface.
func (b VarCharType) ToArrayType() DoltgresArrayType {
	return createArrayTypeWithFuncs(b, SerializationID_VarCharArray, oid.T__varchar, arrayContainerFunctions{
		SQL: varCharArraySQL,
	})
}

// Type implements the DoltgresType interface.
func (b VarCharType) Type() query.Type {
	return sqltypes.Text
}

// ValueType implements the DoltgresType interface.
func (b VarCharType) ValueType() reflect.Type {
	return reflect.TypeOf("")
}

// Zero implements the DoltgresType interface.
func (b VarCharType) Zero() any {
	return ""
}

// SerializeValue implements the DoltgresType interface.
func (b VarCharType) SerializeValue(val any) ([]byte, error) {
	if val == nil {
		return nil, nil
	}
	converted, _, err := b.Convert(val)
	if err != nil {
		return nil, err
	}
	return []byte(converted.(string)), nil
}

// DeserializeValue implements the DoltgresType interface.
func (b VarCharType) DeserializeValue(val []byte) (any, error) {
	if len(val) == 0 {
		return nil, nil
	}
	return string(val), nil
}
