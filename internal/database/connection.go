package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"
)

// Connection represents a database connection configuration
type Connection struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSH      SSHConfig
	TLS      TLSConfig
}

// DSN returns the data source name for MySQL connection
// Uses mysql.Config for proper escaping and timeout configuration
func (c *Connection) DSN() (string, error) {
	cfg := mysql.NewConfig()
	cfg.User = c.User
	cfg.Passwd = c.Password
	cfg.Net = "tcp"
	cfg.Addr = fmt.Sprintf("%s:%d", c.Host, c.Port)
	cfg.DBName = c.Database
	cfg.Timeout = 5 * time.Second
	cfg.ReadTimeout = 30 * time.Second
	cfg.WriteTimeout = 30 * time.Second
	cfg.ParseTime = true

	tlsParam, err := tlsDSNParam(c.TLS)
	if err != nil {
		return "", err
	}
	if tlsParam != "" {
		cfg.TLSConfig = tlsParam
	}

	return cfg.FormatDSN(), nil
}

// Connect establishes a connection to the database
func (c *Connection) Connect() (*sql.DB, error) {
	dsn, err := c.DSN()
	if err != nil {
		return nil, err
	}

	// Register any custom TLS config (custom CA/cert/verify-ca) so the DSN's
	// tls=<name> parameter resolves before the driver dials.
	if err := registerCustomTLS(c.TLS); err != nil {
		return nil, err
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Verify the connection
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}
