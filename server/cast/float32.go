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

package cast

import (
	"fmt"
	"math"
	"strconv"

	"github.com/shopspring/decimal"

	"github.com/dolthub/doltgresql/server/functions/framework"
	pgtypes "github.com/dolthub/doltgresql/server/types"
)

// init handles all explicit and implicit casts that are built-in. This comprises only the "From" types.
func init() {
	float32Explicit()
	float32Implicit()
}

// float32Explicit registers all explicit casts. This comprises only the "From" types.
func float32Explicit() {
	framework.MustAddExplicitTypeCast(framework.TypeCast{
		FromType: pgtypes.Float32,
		ToType:   pgtypes.BpChar,
		Function: func(ctx framework.Context, val any, targetType pgtypes.DoltgresType) (any, error) {
			str := strconv.FormatFloat(float64(val.(float32)), 'g', -1, 32)
			return handleCharExplicitCast(str, targetType)
		},
	})
	framework.MustAddExplicitTypeCast(framework.TypeCast{
		FromType: pgtypes.Float32,
		ToType:   pgtypes.Float32,
		Function: func(ctx framework.Context, val any, targetType pgtypes.DoltgresType) (any, error) {
			return val, nil
		},
	})
	framework.MustAddExplicitTypeCast(framework.TypeCast{
		FromType: pgtypes.Float32,
		ToType:   pgtypes.Float64,
		Function: func(ctx framework.Context, val any, targetType pgtypes.DoltgresType) (any, error) {
			return float64(val.(float32)), nil
		},
	})
	framework.MustAddExplicitTypeCast(framework.TypeCast{
		FromType: pgtypes.Float32,
		ToType:   pgtypes.Int16,
		Function: func(ctx framework.Context, valInterface any, targetType pgtypes.DoltgresType) (any, error) {
			val := float32(math.RoundToEven(float64(valInterface.(float32))))
			if val > 32767 || val < -32768 {
				return nil, fmt.Errorf("smallint out of range")
			}
			return int16(val), nil
		},
	})
	framework.MustAddExplicitTypeCast(framework.TypeCast{
		FromType: pgtypes.Float32,
		ToType:   pgtypes.Int32,
		Function: func(ctx framework.Context, valInterface any, targetType pgtypes.DoltgresType) (any, error) {
			val := float32(math.RoundToEven(float64(valInterface.(float32))))
			if val > 2147483647 || val < -2147483648 {
				return nil, fmt.Errorf("integer out of range")
			}
			return int32(val), nil
		},
	})
	framework.MustAddExplicitTypeCast(framework.TypeCast{
		FromType: pgtypes.Float32,
		ToType:   pgtypes.Int64,
		Function: func(ctx framework.Context, valInterface any, targetType pgtypes.DoltgresType) (any, error) {
			val := float32(math.RoundToEven(float64(valInterface.(float32))))
			if val > 9223372036854775807 || val < -9223372036854775808 {
				return nil, fmt.Errorf("bigint out of range")
			}
			return int64(val), nil
		},
	})
	framework.MustAddExplicitTypeCast(framework.TypeCast{
		FromType: pgtypes.Float32,
		ToType:   pgtypes.Numeric,
		Function: func(ctx framework.Context, val any, targetType pgtypes.DoltgresType) (any, error) {
			return decimal.NewFromFloat(float64(val.(float32))), nil
		},
	})
	framework.MustAddExplicitTypeCast(framework.TypeCast{
		FromType: pgtypes.Float32,
		ToType:   pgtypes.Text,
		Function: func(ctx framework.Context, val any, targetType pgtypes.DoltgresType) (any, error) {
			return strconv.FormatFloat(float64(val.(float32)), 'g', -1, 32), nil
		},
	})
	framework.MustAddExplicitTypeCast(framework.TypeCast{
		FromType: pgtypes.Float32,
		ToType:   pgtypes.VarChar,
		Function: func(ctx framework.Context, val any, targetType pgtypes.DoltgresType) (any, error) {
			str := strconv.FormatFloat(float64(val.(float32)), 'g', -1, 32)
			return handleCharExplicitCast(str, targetType)
		},
	})
}

// float32Implicit registers all implicit casts. This comprises only the "From" types.
func float32Implicit() {
	framework.MustAddImplicitTypeCast(framework.TypeCast{
		FromType: pgtypes.Float32,
		ToType:   pgtypes.BpChar,
		Function: func(ctx framework.Context, val any, targetType pgtypes.DoltgresType) (any, error) {
			str := strconv.FormatFloat(float64(val.(float32)), 'g', -1, 32)
			return handleCharImplicitCast(str, targetType)
		},
	})
	framework.MustAddImplicitTypeCast(framework.TypeCast{
		FromType: pgtypes.Float32,
		ToType:   pgtypes.Float32,
		Function: func(ctx framework.Context, val any, targetType pgtypes.DoltgresType) (any, error) {
			return val, nil
		},
	})
	framework.MustAddImplicitTypeCast(framework.TypeCast{
		FromType: pgtypes.Float32,
		ToType:   pgtypes.Float64,
		Function: func(ctx framework.Context, val any, targetType pgtypes.DoltgresType) (any, error) {
			return float64(val.(float32)), nil
		},
	})
	framework.MustAddImplicitTypeCast(framework.TypeCast{
		FromType: pgtypes.Float32,
		ToType:   pgtypes.Int16,
		Function: func(ctx framework.Context, valInterface any, targetType pgtypes.DoltgresType) (any, error) {
			val := float32(math.RoundToEven(float64(valInterface.(float32))))
			if val > 32767 || val < -32768 {
				return nil, fmt.Errorf("smallint out of range")
			}
			return int16(val), nil
		},
	})
	framework.MustAddImplicitTypeCast(framework.TypeCast{
		FromType: pgtypes.Float32,
		ToType:   pgtypes.Int32,
		Function: func(ctx framework.Context, valInterface any, targetType pgtypes.DoltgresType) (any, error) {
			val := float32(math.RoundToEven(float64(valInterface.(float32))))
			if val > 2147483647 || val < -2147483648 {
				return nil, fmt.Errorf("integer out of range")
			}
			return int32(val), nil
		},
	})
	framework.MustAddImplicitTypeCast(framework.TypeCast{
		FromType: pgtypes.Float32,
		ToType:   pgtypes.Int64,
		Function: func(ctx framework.Context, valInterface any, targetType pgtypes.DoltgresType) (any, error) {
			val := float32(math.RoundToEven(float64(valInterface.(float32))))
			if val > 9223372036854775807 || val < -9223372036854775808 {
				return nil, fmt.Errorf("bigint out of range")
			}
			return int64(val), nil
		},
	})
	framework.MustAddImplicitTypeCast(framework.TypeCast{
		FromType: pgtypes.Float32,
		ToType:   pgtypes.Numeric,
		Function: func(ctx framework.Context, val any, targetType pgtypes.DoltgresType) (any, error) {
			return decimal.NewFromFloat(float64(val.(float32))), nil
		},
	})
	framework.MustAddImplicitTypeCast(framework.TypeCast{
		FromType: pgtypes.Float32,
		ToType:   pgtypes.Text,
		Function: func(ctx framework.Context, val any, targetType pgtypes.DoltgresType) (any, error) {
			return strconv.FormatFloat(float64(val.(float32)), 'g', -1, 32), nil
		},
	})
	framework.MustAddImplicitTypeCast(framework.TypeCast{
		FromType: pgtypes.Float32,
		ToType:   pgtypes.VarChar,
		Function: func(ctx framework.Context, val any, targetType pgtypes.DoltgresType) (any, error) {
			str := strconv.FormatFloat(float64(val.(float32)), 'g', -1, 32)
			return handleCharImplicitCast(str, targetType)
		},
	})
}
