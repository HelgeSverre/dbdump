package database

import (
	"database/sql"
	"fmt"
)

// TableInfo represents information about a table
type TableInfo struct {
	Name        string
	RowCount    int64
	DataSize    int64
	IndexSize   int64
	TotalSize   int64
	SizeDisplay string
}

// Inspector handles database inspection operations
type Inspector struct {
	db *sql.DB
}

// NewInspector creates a new Inspector
func NewInspector(db *sql.DB) *Inspector {
	return &Inspector{db: db}
}

// ListTables returns a list of all tables in the database
func (i *Inspector) ListTables() ([]string, error) {
	query := "SHOW TABLES"
	rows, err := i.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list tables: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var tables []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return nil, fmt.Errorf("failed to scan table name: %w", err)
		}
		tables = append(tables, table)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tables: %w", err)
	}

	return tables, nil
}

// GetTableInfo retrieves detailed information about a table
func (i *Inspector) GetTableInfo(tableName string) (*TableInfo, error) {
	query := `
		SELECT
			table_name,
			IFNULL(table_rows, 0) as row_count,
			IFNULL(data_length, 0) as data_size,
			IFNULL(index_length, 0) as index_size,
			IFNULL(data_length + index_length, 0) as total_size
		FROM information_schema.tables
		WHERE table_schema = DATABASE()
		AND table_name = ?
	`

	var info TableInfo
	err := i.db.QueryRow(query, tableName).Scan(
		&info.Name,
		&info.RowCount,
		&info.DataSize,
		&info.IndexSize,
		&info.TotalSize,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get table info: %w", err)
	}

	info.SizeDisplay = formatBytes(info.TotalSize)

	return &info, nil
}

// GetAllTablesInfo retrieves information for all tables
func (i *Inspector) GetAllTablesInfo() ([]TableInfo, error) {
	query := `
		SELECT
			table_name,
			IFNULL(table_rows, 0) as row_count,
			IFNULL(data_length, 0) as data_size,
			IFNULL(index_length, 0) as index_size,
			IFNULL(data_length + index_length, 0) as total_size
		FROM information_schema.tables
		WHERE table_schema = DATABASE()
		ORDER BY total_size DESC
	`

	rows, err := i.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get tables info: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var tables []TableInfo
	for rows.Next() {
		var info TableInfo
		if err := rows.Scan(
			&info.Name,
			&info.RowCount,
			&info.DataSize,
			&info.IndexSize,
			&info.TotalSize,
		); err != nil {
			return nil, fmt.Errorf("failed to scan table info: %w", err)
		}

		info.SizeDisplay = formatBytes(info.TotalSize)
		tables = append(tables, info)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating table info: %w", err)
	}

	return tables, nil
}

// formatBytes formats byte size into human-readable format
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	sizes := []string{"KB", "MB", "GB", "TB"}
	if exp >= len(sizes) {
		exp = len(sizes) - 1
	}

	return fmt.Sprintf("%.1f %s", float64(bytes)/float64(div), sizes[exp])
}
