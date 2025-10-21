package database

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

// Connection represents a database connection configuration
type Connection struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
}

// DSN returns the data source name for MySQL connection
func (c *Connection) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
		c.User,
		c.Password,
		c.Host,
		c.Port,
		c.Database,
	)
}

// Connect establishes a connection to the database
func (c *Connection) Connect() (*sql.DB, error) {
	db, err := sql.Open("mysql", c.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Verify the connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// TestConnection tests if the connection is valid
func (c *Connection) TestConnection() error {
	db, err := c.Connect()
	if err != nil {
		return err
	}
	defer db.Close()
	return nil
}
