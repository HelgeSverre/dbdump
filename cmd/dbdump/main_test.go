package main

import "testing"

func TestEffectiveConnectionPrefersExplicitPassword(t *testing.T) {
	t.Setenv("DBDUMP_MYSQL_PWD", "from-dbdump-env")
	t.Setenv("MYSQL_PWD", "from-mysql-env")

	conn := connectionFlags{
		Host:     "127.0.0.1",
		Port:     3306,
		User:     "root",
		Password: "from-flag",
		Database: "testdb",
	}

	connection := conn.withResolvedPassword().toConnection()

	if connection.Password != "from-flag" {
		t.Fatalf("expected explicit password to win, got %q", connection.Password)
	}
}

func TestEffectiveConnectionPrefersDBDumpEnvOverMySQLEnv(t *testing.T) {
	t.Setenv("DBDUMP_MYSQL_PWD", "from-dbdump-env")
	t.Setenv("MYSQL_PWD", "from-mysql-env")

	conn := connectionFlags{
		Host:     "127.0.0.1",
		Port:     3306,
		User:     "root",
		Database: "testdb",
	}

	connection := conn.withResolvedPassword().toConnection()

	if connection.Password != "from-dbdump-env" {
		t.Fatalf("expected DBDUMP_MYSQL_PWD to win, got %q", connection.Password)
	}
}
