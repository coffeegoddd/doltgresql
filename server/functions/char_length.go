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

package functions

import (
	"github.com/dolthub/go-mysql-server/sql"

	"github.com/dolthub/doltgresql/server/functions/framework"
	pgtypes "github.com/dolthub/doltgresql/server/types"
)

// initCharLength registers the functions to the catalog.
func initCharLength() {
	framework.RegisterFunction(char_length_varchar)
	// Register alias
	character_length_varchar := char_length_varchar
	character_length_varchar.Name = "character_length"
	framework.RegisterFunction(character_length_varchar)
}

// char_length_varchar represents the PostgreSQL function of the same name, taking the same parameters.
var char_length_varchar = framework.Function1{
	Name:       "char_length",
	Return:     pgtypes.Int32,
	Parameters: []pgtypes.DoltgresType{pgtypes.VarChar},
	Callable: func(ctx *sql.Context, val1 any) (any, error) {
		if val1 == nil {
			return nil, nil
		}
		return int32(len([]rune(val1.(string)))), nil
	},
}
