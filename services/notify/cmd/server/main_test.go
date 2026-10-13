package main

import "testing"

func TestDatabaseName(t *testing.T) {
	name, err := databaseName("postgresql://bicmas:bicmas@localhost:5432/notify?sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	if name != "notify" {
		t.Fatalf("database = %s", name)
	}
}

func TestDatabaseNameRejectsMissingName(t *testing.T) {
	if _, err := databaseName("postgresql://bicmas:bicmas@localhost:5432"); err == nil {
		t.Fatal("expected an error")
	}
}
