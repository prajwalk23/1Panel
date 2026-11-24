package common

import (
	"context"
	"path"
	"testing"
)

// fork: the PRAGMAs travel in the DSN so every pooled connection gets them.
func TestGetDBWithPathAppliesPragmasToEveryConnection(t *testing.T) {
	db, err := GetDBWithPath(path.Join(t.TempDir(), "pragma.db"))
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	ctx := context.Background()
	c1, err := sqlDB.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer c1.Close()
	c2, err := sqlDB.Conn(ctx) // held at the same time, so this is a second physical connection
	if err != nil {
		t.Fatal(err)
	}
	defer c2.Close()

	want := map[string]int64{"busy_timeout": 5000, "cache_size": -1024, "synchronous": 1, "temp_store": 2}
	for name, conn := range map[string]func(string) (int64, error){
		"conn1": func(p string) (int64, error) { var v int64; err := c1.QueryRowContext(ctx, "PRAGMA "+p).Scan(&v); return v, err },
		"conn2": func(p string) (int64, error) { var v int64; err := c2.QueryRowContext(ctx, "PRAGMA "+p).Scan(&v); return v, err },
	} {
		for pragma, expected := range want {
			got, err := conn(pragma)
			if err != nil {
				t.Fatalf("%s PRAGMA %s: %v", name, pragma, err)
			}
			if got != expected {
				t.Errorf("%s PRAGMA %s = %d, want %d", name, pragma, got, expected)
			}
		}
	}
	var mode string
	if err := c2.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&mode); err != nil || mode != "wal" {
		t.Errorf("journal_mode = %q (err %v), want wal", mode, err)
	}
}
