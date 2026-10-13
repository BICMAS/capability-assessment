package main

import "testing"

func TestDatabaseName(t *testing.T) {
	name, err := databaseName("postgresql://bicmas:bicmas@localhost:5432/org_directory?sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	if name != "org_directory" {
		t.Fatalf("database = %s", name)
	}
}

func TestDatabaseNameRejectsMissingName(t *testing.T) {
	if _, err := databaseName("postgresql://bicmas:bicmas@localhost:5432"); err == nil {
		t.Fatal("expected an error")
	}
}
