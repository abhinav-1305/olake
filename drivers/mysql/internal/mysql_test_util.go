package driver

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/datazip-inc/olake/drivers/abstract"
	"github.com/datazip-inc/olake/types"
	"github.com/datazip-inc/olake/utils/logger"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

const (
	defaultMySQLUser     = "mysql"
	defaultMySQLHost     = "localhost"
	defaultMySQLPort     = 3306
	defaultMySQLPassword = "secret1234"
	defaultMySQLDatabase = "mysql"
	defaultMaxThreads    = 4
	defaultRetryCount    = 3
	initialCDCWaitTime   = 5
)

// ExecuteQuery executes MySQL queries for testing based on the operation type
func ExecuteQuery(ctx context.Context, t *testing.T, conn interface{}, tableName string, operation string) {
	t.Helper()

	db, ok := conn.(*sqlx.DB)
	require.True(t, ok, "Expected *sqlx.DB connection")

	var (
		query string
		err   error
	)

	switch operation {
	case "create":
		query = fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s (
				id INTEGER PRIMARY KEY,
				col1 VARCHAR(255),
				col2 VARCHAR(255)
			)`, tableName)

	case "drop":
		query = fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName)

	case "clean":
		query = fmt.Sprintf("DELETE FROM %s", tableName)

	case "add":
		insertTestData(t, ctx, db, tableName)
		return // Early return since we handle all inserts in the helper function

	case "insert":
		query = fmt.Sprintf(`
			INSERT INTO %s (id, col1, col2) 
			VALUES (10, 'new val', 'new val')`, tableName)

	case "update":
		query = fmt.Sprintf(`
			UPDATE %s 
			SET col1 = 'updated val' 
			WHERE id = (
				SELECT id FROM (
					SELECT id FROM %s ORDER BY RAND() LIMIT 1
				) AS subquery
			)`, tableName, tableName)

	case "delete":
		query = fmt.Sprintf(`
			DELETE FROM %s 
			WHERE id = (
				SELECT id FROM (
					SELECT id FROM %s ORDER BY RAND() LIMIT 1
				) AS subquery
			)`, tableName, tableName)

	default:
		t.Fatalf("Unsupported operation: %s", operation)
	}

	_, err = db.ExecContext(ctx, query)
	require.NoError(t, err, "Failed to execute %s operation", operation)
}

// insertTestData inserts test data into the specified table
func insertTestData(t *testing.T, ctx context.Context, db *sqlx.DB, tableName string) {
	t.Helper()

	for i := 1; i <= 5; i++ {
		query := fmt.Sprintf(`
			INSERT INTO %s (id, col1, col2) 
			VALUES (%d, 'value%d_col1', 'value%d_col2')`,
			tableName, i, i, i)

		_, err := db.ExecContext(ctx, query)
		require.NoError(t, err, "Failed to insert test data row %d", i)
	}
}

// testAndBuildAbstractDriver initializes and returns an AbstractDriver with default configuration
func testAndBuildAbstractDriver(t *testing.T) (*sqlx.DB, *abstract.AbstractDriver) {
	t.Helper()
	logger.Init()

	config := Config{
		Username:   defaultMySQLUser,
		Host:       defaultMySQLHost,
		Port:       defaultMySQLPort,
		Password:   defaultMySQLPassword,
		Database:   defaultMySQLDatabase,
		MaxThreads: defaultMaxThreads,
		RetryCount: defaultRetryCount,
	}

	mysqlDriver := &MySQL{
		config: &config,
		cdcConfig: CDC{
			InitialWaitTime: initialCDCWaitTime,
		},
	}
	mysqlDriver.CDCSupport = true
	absDriver := abstract.NewAbstractDriver(context.Background(), mysqlDriver)

	state := &types.State{
		Type:    types.StreamType,
		RWMutex: &sync.RWMutex{},
	}
	absDriver.SetupState(state)
	require.NoError(t, absDriver.Setup(context.Background()), "Failed to setup MySQL driver")

	return mysqlDriver.client, absDriver
}
