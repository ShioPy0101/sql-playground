package helper

import (
	"context"
	"errors"
	"testing"
)

func TestMeasureSelectStatementPoC(t *testing.T) {
	db, databasePath, cleanup, err := OpenTempSQLiteDBWithPath()
	if err != nil {
		t.Fatalf("OpenTempSQLiteDBWithPath returned error: %v", err)
	}
	defer cleanup()
	defer db.Close()

	constant, err := MeasureSelectStatement(databasePath, `SELECT 1`)
	if err != nil {
		t.Fatalf("measure SELECT 1: %v", err)
	}
	if constant.VMSteps <= 0 {
		t.Fatalf("SELECT 1 VM_STEP = %d, want positive", constant.VMSteps)
	}
	if constant.FullScanSteps != 0 {
		t.Fatalf("SELECT 1 FULLSCAN_STEP = %d, want 0", constant.FullScanSteps)
	}

	if _, err := db.Exec(`
		CREATE TABLE items (id INTEGER PRIMARY KEY, value INTEGER);
		INSERT INTO items (id, value) VALUES (1, 10), (2, 20), (3, 30), (4, 40);
	`); err != nil {
		t.Fatalf("initialize table: %v", err)
	}
	tableScan, err := MeasureSelectStatement(databasePath, `SELECT value FROM items WHERE value > 0`)
	if err != nil {
		t.Fatalf("measure table SELECT: %v", err)
	}
	if tableScan.VMSteps <= constant.VMSteps {
		t.Fatalf("table VM_STEP = %d, want greater than SELECT 1 value %d", tableScan.VMSteps, constant.VMSteps)
	}
	if tableScan.FullScanSteps <= 0 {
		t.Fatalf("table FULLSCAN_STEP = %d, want positive", tableScan.FullScanSteps)
	}

	t.Logf("SELECT 1: VM_STEP=%d FULLSCAN_STEP=%d", constant.VMSteps, constant.FullScanSteps)
	t.Logf("table SELECT: VM_STEP=%d FULLSCAN_STEP=%d", tableScan.VMSteps, tableScan.FullScanSteps)
}

func TestMeasureSelectStatementHonorsCanceledContext(t *testing.T) {
	db, databasePath, cleanup, err := OpenTempSQLiteDBWithPath()
	if err != nil {
		t.Fatalf("OpenTempSQLiteDBWithPath returned error: %v", err)
	}
	defer cleanup()
	defer db.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = MeasureSelectStatementContext(ctx, databasePath, `SELECT 1`)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}
