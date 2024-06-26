// Copyright 2023 Dolthub, Inc.
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

package _go

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net"
	"os"
	"testing"
	"time"

	"github.com/dolthub/dolt/go/libraries/utils/svcs"
	"github.com/dolthub/go-mysql-server/sql"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	dserver "github.com/dolthub/doltgresql/server"
	"github.com/dolthub/doltgresql/servercfg"
)

// runOnPostgres is a debug setting to redirect the test framework to a local running postgres server,
// rather than starting a doltgres server.
const runOnPostgres = false

// ScriptTest defines a consistent structure for testing queries.
type ScriptTest struct {
	// Name of the script.
	Name string
	// The database to create and use. If not provided, then it defaults to "postgres".
	Database string
	// The SQL statements to execute as setup, in order. Results are not checked, but statements must not error.
	SetUpScript []string
	// The set of assertions to make after setup, in order
	Assertions []ScriptTestAssertion
	// When using RunScripts, setting this on one (or more) tests causes RunScripts to ignore all tests that have this
	// set to false (which is the default value). This allows a developer to easily "focus" on a specific test without
	// having to comment out other tests, pull it into a different function, etc. In addition, CI ensures that this is
	// false before passing, meaning this prevents the commented-out situation where the developer forgets to uncomment
	// their code.
	Focus bool
	// Skip is used to completely skip a test including setup
	Skip bool
}

// ScriptTestAssertion are the assertions upon which the script executes its main "testing" logic.
type ScriptTestAssertion struct {
	Query       string
	Expected    []sql.Row
	ExpectedErr string

	BindVars []any

	// SkipResultsCheck is used to skip assertions on the expected rows returned from a query. For now, this is
	// included as some messages do not have a full logical implementation. Skipping the results check allows us to
	// force the test client to not send of those messages.
	SkipResultsCheck bool

	// Skip is used to completely skip a test, not execute its query at all, and record it as a skipped test
	// in the test suite results.
	Skip bool

	// ExpectedTag is used to check the command tag returned from the server.
	// This is checked only if no Expected is defined
	ExpectedTag string
}

// RunScript runs the given script.
func RunScript(t *testing.T, script ScriptTest, normalizeRows bool) {
	scriptDatabase := script.Database
	if len(scriptDatabase) == 0 {
		scriptDatabase = "postgres"
	}

	var ctx context.Context
	var conn *pgx.Conn

	if runOnPostgres {
		var err error
		ctx = context.Background()
		conn, err = pgx.Connect(ctx, fmt.Sprintf("postgres://postgres:password@127.0.0.1:%d/%s?sslmode=disable", 5432, scriptDatabase))
		require.NoError(t, err)
		defer func() {
			_ = conn.Close(ctx)
		}()
	} else {
		var controller *svcs.Controller
		ctx, conn, controller = CreateServer(t, scriptDatabase)
		defer func() {
			_ = conn.Close(ctx)
			controller.Stop()
			err := controller.WaitForStop()
			require.NoError(t, err)
		}()
	}

	t.Run(script.Name, func(t *testing.T) {
		runScript(t, ctx, script, conn, normalizeRows)
	})
}

// runScript runs the script given on the postgres connection provided
func runScript(t *testing.T, ctx context.Context, script ScriptTest, conn *pgx.Conn, normalizeRows bool) {
	if script.Skip {
		t.Skip("Skip has been set in the script")
	}

	// Run the setup
	for _, query := range script.SetUpScript {
		_, err := conn.Exec(ctx, query)
		require.NoError(t, err)
	}

	// Run the assertions
	for _, assertion := range script.Assertions {
		t.Run(assertion.Query, func(t *testing.T) {
			if assertion.Skip {
				t.Skip("Skip has been set in the assertion")
			}
			// If we're skipping the results check, then we call Execute, as it uses a simplified message model.
			if assertion.SkipResultsCheck || assertion.ExpectedErr != "" {
				_, err := conn.Exec(ctx, assertion.Query, assertion.BindVars...)
				if assertion.ExpectedErr != "" {
					require.Error(t, err)
					assert.Contains(t, err.Error(), assertion.ExpectedErr)
				} else {
					require.NoError(t, err)
				}
			} else if assertion.ExpectedTag != "" {
				// check for command tag
				commandTag, err := conn.Exec(ctx, assertion.Query)
				require.NoError(t, err)
				assert.Equal(t, assertion.ExpectedTag, commandTag.String())
			} else {
				rows, err := conn.Query(ctx, assertion.Query, assertion.BindVars...)
				require.NoError(t, err)
				readRows, err := ReadRows(rows, normalizeRows)
				require.NoError(t, err)
				if normalizeRows {
					assert.Equal(t, NormalizeRows(assertion.Expected), readRows)
				} else {
					assert.Equal(t, assertion.Expected, readRows)
				}
			}
		})
	}
}

// RunScripts runs the given collection of scripts. This normalizes all rows before comparing them.
func RunScripts(t *testing.T, scripts []ScriptTest) {
	runScripts(t, scripts, true)
}

// RunScriptsWithoutNormalization runs the given collection of scripts, without normalizing any rows.
func RunScriptsWithoutNormalization(t *testing.T, scripts []ScriptTest) {
	runScripts(t, scripts, false)
}

// runScripts is the implementation of both RunScripts and RunScriptsWithoutNormalization.
func runScripts(t *testing.T, scripts []ScriptTest, normalizeRows bool) {
	// First, we'll run through the scripts to check for the Focus variable. If it's true, then append it to the new slice.
	focusScripts := make([]ScriptTest, 0, len(scripts))
	for _, script := range scripts {
		if script.Focus {
			// If this is running in GitHub Actions, then we'll panic, because someone forgot to disable it before committing
			if _, ok := os.LookupEnv("GITHUB_ACTION"); ok {
				panic(fmt.Sprintf("The script `%s` has Focus set to `true`. GitHub Actions requires that "+
					"all tests are run, which Focus circumvents, leading to this error. Please disable Focus on "+
					"all tests.", script.Name))
			}
			focusScripts = append(focusScripts, script)
		}
	}
	// If we have scripts with Focus set, then we replace the normal script slice with the new slice.
	if len(focusScripts) > 0 {
		scripts = focusScripts
	}

	for _, script := range scripts {
		RunScript(t, script, normalizeRows)
	}
}

func ptr[T any](val T) *T {
	return &val
}

// CreateServer creates a server with the given database, returning a connection to the server. The server will close
// when the connection is closed (or loses its connection to the server). The accompanying WaitGroup may be used to wait
// until the server has closed.
func CreateServer(t *testing.T, database string) (context.Context, *pgx.Conn, *svcs.Controller) {
	require.NotEmpty(t, database)
	port := GetUnusedPort(t)
	controller, err := dserver.RunInMemory(&servercfg.DoltgresConfig{
		ListenerConfig: &servercfg.DoltgresListenerConfig{
			PortNumber: &port,
			HostStr:    ptr("127.0.0.1"),
		},
	})
	require.NoError(t, err)

	fmt.Printf("port is %d\n", port)

	ctx := context.Background()
	err = func() error {
		// The connection attempt may be made before the server has grabbed the port, so we'll retry the first
		// connection a few times.
		var conn *pgx.Conn
		var err error
		for i := 0; i < 3; i++ {
			conn, err = pgx.Connect(ctx, fmt.Sprintf("postgres://postgres:password@127.0.0.1:%d/", port))
			if err == nil {
				break
			} else {
				time.Sleep(time.Second)
			}
		}
		if err != nil {
			return err
		}

		defer conn.Close(ctx)
		_, err = conn.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s;", database))
		return err
	}()
	require.NoError(t, err)

	conn, err := pgx.Connect(ctx, fmt.Sprintf("postgres://postgres:password@127.0.0.1:%d/%s", port, database))
	require.NoError(t, err)
	return ctx, conn, controller
}

// ReadRows reads all of the given rows into a slice, then closes the rows. If `normalizeRows` is true, then the rows
// will be normalized such that all integers are int64, etc.
func ReadRows(rows pgx.Rows, normalizeRows bool) (readRows []sql.Row, err error) {
	defer func() {
		err = errors.Join(err, rows.Err())
	}()
	var slice []sql.Row
	for rows.Next() {
		row, err := rows.Values()
		if err != nil {
			return nil, err
		}
		slice = append(slice, row)
	}
	if normalizeRows {
		return NormalizeRows(slice), nil
	} else {
		// We must always normalize Numeric values, as they have an infinite number of ways to represent the same value
		return NormalizeRowsOnlyNumeric(slice), nil
	}
}

// NormalizeRow normalizes each value's type, as the tests only want to compare values. Returns a new row.
func NormalizeRow(row sql.Row) sql.Row {
	if len(row) == 0 {
		return nil
	}
	newRow := make(sql.Row, len(row))
	for i := range row {
		switch val := row[i].(type) {
		case int:
			newRow[i] = int64(val)
		case int8:
			newRow[i] = int64(val)
		case int16:
			newRow[i] = int64(val)
		case int32:
			newRow[i] = int64(val)
		case uint:
			newRow[i] = int64(val)
		case uint8:
			newRow[i] = int64(val)
		case uint16:
			newRow[i] = int64(val)
		case uint32:
			newRow[i] = int64(val)
		case uint64:
			// PostgreSQL does not support an uint64 type, so we can always convert this to an int64 safely.
			newRow[i] = int64(val)
		case float32:
			newRow[i] = float64(val)
		case pgtype.Numeric:
			if val.NaN {
				newRow[i] = math.NaN()
			} else if val.InfinityModifier != pgtype.Finite {
				newRow[i] = math.Inf(int(val.InfinityModifier))
			} else if !val.Valid {
				newRow[i] = nil
			} else {
				fVal, err := val.Float64Value()
				if err != nil {
					panic(err)
				}
				if !fVal.Valid {
					panic("no idea why the numeric float value is invalid")
				}
				newRow[i] = fVal.Float64
			}
		case time.Time:
			newRow[i] = val.Format("2006-01-02 15:04:05")
		case map[string]interface{}:
			str, err := json.Marshal(val)
			if err != nil {
				panic(err)
			}
			newRow[i] = string(str)
		default:
			newRow[i] = val
		}
	}
	return newRow
}

// NormalizeRows normalizes each value's type within each row, as the tests only want to compare values. Returns a new
// set of rows in the same order.
func NormalizeRows(rows []sql.Row) []sql.Row {
	newRows := make([]sql.Row, len(rows))
	for i := range rows {
		newRows[i] = NormalizeRow(rows[i])
	}
	return newRows
}

// NormalizeRowsOnlyNumeric normalizes Numeric values only. There are an infinite number of ways to represent the same
// value in-memory, so we must at least normalize Numeric values.
func NormalizeRowsOnlyNumeric(rows []sql.Row) []sql.Row {
	newRows := make([]sql.Row, len(rows))
	for rowIdx, row := range rows {
		newRow := make(sql.Row, len(row))
		copy(newRow, row)
		for colIdx := range newRow {
			if numericValue, ok := newRow[colIdx].(pgtype.Numeric); ok {
				val, err := numericValue.Value()
				if err != nil {
					panic(err) // Should never happen
				}
				// Using decimal as an intermediate value will remove all differences between the string formatting
				d := decimal.RequireFromString(val.(string))
				newRow[colIdx] = Numeric(d.String())
			}
		}
		newRows[rowIdx] = newRow
	}
	return newRows
}

// GetUnusedPort returns an unused port.
func GetUnusedPort(t *testing.T) int {
	listener, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	port := listener.Addr().(*net.TCPAddr).Port
	require.NoError(t, listener.Close())
	return port
}

// Numeric creates a numeric value from a string.
func Numeric(str string) pgtype.Numeric {
	numeric := pgtype.Numeric{}
	if err := numeric.Scan(str); err != nil {
		panic(err)
	}
	return numeric
}
