package helper

import (
	"context"
	"errors"
	"fmt"
	"time"
	"unsafe"

	"modernc.org/libc"
	"modernc.org/libc/sys/types"
	sqlite3 "modernc.org/sqlite/lib"
)

var errStatementStatusNotReadOnly = errors.New("statement status is limited to read-only statements")

// NativeStatementStatus contains counters reported by sqlite3_stmt_status.
type NativeStatementStatus struct {
	DurationMS     float64
	VMSteps        int64
	FullScanSteps  int64
	SortOperations int64
	AutoIndexRows  int64
}

// MeasureSelectStatement opens an independent read-only connection to the same
// database file and executes one prepared statement to collect native SQLite
// counters. It does not access database/sql or modernc.org/sqlite internals.
func MeasureSelectStatement(databasePath string, statement string) (NativeStatementStatus, error) {
	return MeasureSelectStatementContext(context.Background(), databasePath, statement)
}

// MeasureStatement executes one statement against a disposable writable
// database and collects native SQLite counters. Callers are responsible for
// restricting the accepted SQL and discarding the database afterward.
func MeasureStatement(databasePath string, statement string) (NativeStatementStatus, error) {
	return MeasureStatementContext(context.Background(), databasePath, statement)
}

// MeasureSelectStatementContext checks cancellation between sqlite3_step calls.
// Benchmark queries are deliberately restricted and bounded, so one step cannot
// hide unbounded user SQL work.
func MeasureSelectStatementContext(ctx context.Context, databasePath string, statement string) (NativeStatementStatus, error) {
	return measureStatementContext(ctx, databasePath, statement, true)
}

// MeasureStatementContext supports both reads and writes. Benchmark databases
// are stage-local temporary files, so mutations never reach task or user data.
func MeasureStatementContext(ctx context.Context, databasePath string, statement string) (NativeStatementStatus, error) {
	return measureStatementContext(ctx, databasePath, statement, false)
}

func measureStatementContext(ctx context.Context, databasePath string, statement string, readOnly bool) (NativeStatementStatus, error) {
	tls := libc.NewTLS()
	defer tls.Close()

	db, err := openLowLevelSQLite(tls, databasePath, readOnly)
	if err != nil {
		return NativeStatementStatus{}, err
	}
	defer sqlite3.Xsqlite3_close_v2(tls, db)

	prepared, err := prepareLowLevelSQLite(tls, db, statement)
	if err != nil {
		return NativeStatementStatus{}, err
	}
	defer sqlite3.Xsqlite3_finalize(tls, prepared)

	if readOnly && sqlite3.Xsqlite3_stmt_readonly(tls, prepared) == 0 {
		return NativeStatementStatus{}, errStatementStatusNotReadOnly
	}
	if count := sqlite3.Xsqlite3_bind_parameter_count(tls, prepared); count != 0 {
		return NativeStatementStatus{}, fmt.Errorf("statement status does not support %d unbound parameters", count)
	}

	startedAt := time.Now()
	for {
		if err := ctx.Err(); err != nil {
			return NativeStatementStatus{}, err
		}
		rc := sqlite3.Xsqlite3_step(tls, prepared)
		if err := ctx.Err(); err != nil {
			return NativeStatementStatus{}, err
		}
		switch rc {
		case sqlite3.SQLITE_ROW:
			continue
		case sqlite3.SQLITE_DONE:
			return NativeStatementStatus{
				DurationMS:     float64(time.Since(startedAt).Nanoseconds()) / float64(time.Millisecond),
				VMSteps:        int64(sqlite3.Xsqlite3_stmt_status(tls, prepared, sqlite3.SQLITE_STMTSTATUS_VM_STEP, 0)),
				FullScanSteps:  int64(sqlite3.Xsqlite3_stmt_status(tls, prepared, sqlite3.SQLITE_STMTSTATUS_FULLSCAN_STEP, 0)),
				SortOperations: int64(sqlite3.Xsqlite3_stmt_status(tls, prepared, sqlite3.SQLITE_STMTSTATUS_SORT, 0)),
				AutoIndexRows:  int64(sqlite3.Xsqlite3_stmt_status(tls, prepared, sqlite3.SQLITE_STMTSTATUS_AUTOINDEX, 0)),
			}, nil
		default:
			return NativeStatementStatus{}, lowLevelSQLiteError(tls, db, rc)
		}
	}
}

func openLowLevelSQLite(tls *libc.TLS, databasePath string, readOnly bool) (uintptr, error) {
	filename, err := libc.CString(databasePath)
	if err != nil {
		return 0, err
	}
	defer libc.Xfree(tls, filename)

	output := allocatePointer(tls)
	if output == 0 {
		return 0, errors.New("allocate sqlite database pointer")
	}
	defer libc.Xfree(tls, output)

	flags := int32(sqlite3.SQLITE_OPEN_READWRITE | sqlite3.SQLITE_OPEN_URI)
	if readOnly {
		flags = sqlite3.SQLITE_OPEN_READONLY | sqlite3.SQLITE_OPEN_URI
	}
	rc := sqlite3.Xsqlite3_open_v2(
		tls,
		filename,
		output,
		flags,
		0,
	)
	db := *(*uintptr)(unsafe.Pointer(output))
	if rc != sqlite3.SQLITE_OK {
		if db != 0 {
			defer sqlite3.Xsqlite3_close_v2(tls, db)
		}
		return 0, lowLevelSQLiteError(tls, db, rc)
	}
	return db, nil
}

func prepareLowLevelSQLite(tls *libc.TLS, db uintptr, statement string) (uintptr, error) {
	sqlText, err := libc.CString(statement)
	if err != nil {
		return 0, err
	}
	defer libc.Xfree(tls, sqlText)

	output := allocatePointer(tls)
	if output == 0 {
		return 0, errors.New("allocate sqlite statement pointer")
	}
	defer libc.Xfree(tls, output)

	if rc := sqlite3.Xsqlite3_prepare_v2(tls, db, sqlText, -1, output, 0); rc != sqlite3.SQLITE_OK {
		return 0, lowLevelSQLiteError(tls, db, rc)
	}
	prepared := *(*uintptr)(unsafe.Pointer(output))
	if prepared == 0 {
		return 0, errors.New("sqlite statement is empty")
	}
	return prepared, nil
}

func allocatePointer(tls *libc.TLS) uintptr {
	return libc.Xmalloc(tls, types.Size_t(unsafe.Sizeof(uintptr(0))))
}

func lowLevelSQLiteError(tls *libc.TLS, db uintptr, resultCode int32) error {
	if db == 0 {
		return fmt.Errorf("sqlite result code %d", resultCode)
	}
	return fmt.Errorf("sqlite result code %d: %s", resultCode, libc.GoString(sqlite3.Xsqlite3_errmsg(tls, db)))
}
