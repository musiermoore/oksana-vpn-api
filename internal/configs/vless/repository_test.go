package vless

import (
	"database/sql"
	"testing"
)

func TestLinkUserHealthCheckFlagHandlesNull(t *testing.T) {
	var flag sql.NullBool
	if err := flag.Scan(nil); err != nil {
		t.Fatalf("scan NULL health-check flag: %v", err)
	}

	user := linkUser{IsHealthCheckUser: flag}
	if user.isHealthCheckAccount() {
		t.Fatal("NULL health-check flag must not grant health-check access")
	}
}

func TestLinkUserRecognizesHealthCheckFlag(t *testing.T) {
	user := linkUser{IsHealthCheckUser: sql.NullBool{Bool: true, Valid: true}}
	if !user.isHealthCheckAccount() {
		t.Fatal("true health-check flag was not recognized")
	}
}
