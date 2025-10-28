#!/usr/bin/env bash

set -euo pipefail

# Sample data generator for dbdump testing and benchmarking
# Usage: ./generate-sample-data.sh [size] [host] [port] [database]
#
# Size options:
#   small   - ~10MB   (1K users, 10K orders)
#   medium  - ~100MB  (10K users, 100K orders, audit logs)
#   large   - ~1GB    (100K users, 1M orders, extensive audit logs)
#   xlarge  - ~10GB   (1M users, 10M orders, massive audit logs)

# Configuration
SIZE="${1:-medium}"
HOST="${2:-127.0.0.1}"
PORT="${3:-3308}"
DATABASE="${4:-testdb}"
PASSWORD="${MYSQL_ROOT_PASSWORD:-testpass123}"

# Size presets (using case for bash 3.2 compatibility)
case "$SIZE" in
    small)
        USERS=1000
        ORDERS=10000
        AUDITS=5000
        ;;
    medium)
        USERS=10000
        ORDERS=100000
        AUDITS=50000
        ;;
    large)
        USERS=100000
        ORDERS=1000000
        AUDITS=500000
        ;;
    xlarge)
        USERS=1000000
        ORDERS=10000000
        AUDITS=5000000
        ;;
    *)
        echo "Error: Invalid size '$SIZE'. Use: small, medium, large, or xlarge"
        exit 1
        ;;
esac

echo "==================================="
echo "dbdump Sample Data Generator"
echo "==================================="
echo "Target: $HOST:$PORT/$DATABASE"
echo "Size:   $SIZE"
echo "Users:  $USERS"
echo "Orders: $ORDERS"
echo "Audits: $AUDITS"
echo "==================================="
echo ""

# MySQL connection command
MYSQL_CMD="mysql -h $HOST -P $PORT -u root -p$PASSWORD $DATABASE"

echo "[1/6] Creating schema..."

$MYSQL_CMD <<'EOF'
-- Drop existing tables
DROP TABLE IF EXISTS audits;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS cache;
DROP TABLE IF EXISTS telescope_entries;
DROP TABLE IF EXISTS order_items;
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS products;
DROP TABLE IF EXISTS users;

-- Users table
CREATE TABLE users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_email (email),
    INDEX idx_created (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Products table
CREATE TABLE products (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    price DECIMAL(10,2) NOT NULL,
    stock INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_price (price)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Orders table
CREATE TABLE orders (
    id INT AUTO_INCREMENT PRIMARY KEY,
    user_id INT NOT NULL,
    total DECIMAL(10,2) NOT NULL,
    status ENUM('pending', 'processing', 'completed', 'cancelled') DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_user (user_id),
    INDEX idx_status (status),
    INDEX idx_created (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Order items table
CREATE TABLE order_items (
    id INT AUTO_INCREMENT PRIMARY KEY,
    order_id INT NOT NULL,
    product_id INT NOT NULL,
    quantity INT NOT NULL,
    price DECIMAL(10,2) NOT NULL,
    FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE,
    FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE,
    INDEX idx_order (order_id),
    INDEX idx_product (product_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Audits table (noisy - should be excluded)
CREATE TABLE audits (
    id INT AUTO_INCREMENT PRIMARY KEY,
    user_type VARCHAR(255),
    user_id INT,
    event VARCHAR(255) NOT NULL,
    auditable_type VARCHAR(255),
    auditable_id INT,
    old_values TEXT,
    new_values TEXT,
    ip_address VARCHAR(45),
    user_agent TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_user (user_type, user_id),
    INDEX idx_auditable (auditable_type, auditable_id),
    INDEX idx_created (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Sessions table (noisy - should be excluded)
CREATE TABLE sessions (
    id VARCHAR(255) PRIMARY KEY,
    user_id INT,
    ip_address VARCHAR(45),
    user_agent TEXT,
    payload TEXT,
    last_activity INT,
    INDEX idx_user (user_id),
    INDEX idx_last_activity (last_activity)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Cache table (noisy - should be excluded)
CREATE TABLE cache (
    `key` VARCHAR(255) PRIMARY KEY,
    value MEDIUMTEXT,
    expiration INT,
    INDEX idx_expiration (expiration)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Telescope entries (noisy - should be excluded)
CREATE TABLE telescope_entries (
    sequence BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    uuid CHAR(36) NOT NULL UNIQUE,
    batch_id CHAR(36) NOT NULL,
    family_hash VARCHAR(255),
    should_display_on_index BOOLEAN DEFAULT 1,
    type VARCHAR(20) NOT NULL,
    content LONGTEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_batch (batch_id),
    INDEX idx_family (family_hash),
    INDEX idx_created (created_at),
    INDEX idx_type (type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Stored procedure example
DROP PROCEDURE IF EXISTS get_user_orders;
DELIMITER $$
CREATE PROCEDURE get_user_orders(IN userId INT)
BEGIN
    SELECT o.*, COUNT(oi.id) as item_count
    FROM orders o
    LEFT JOIN order_items oi ON o.id = oi.order_id
    WHERE o.user_id = userId
    GROUP BY o.id;
END$$
DELIMITER ;

-- Trigger example
DROP TRIGGER IF EXISTS after_order_insert;
DELIMITER $$
CREATE TRIGGER after_order_insert
AFTER INSERT ON orders
FOR EACH ROW
BEGIN
    INSERT INTO audits (event, auditable_type, auditable_id, new_values, created_at)
    VALUES ('created', 'Order', NEW.id, JSON_OBJECT('total', NEW.total, 'status', NEW.status), NOW());
END$$
DELIMITER ;

-- Event example (disabled by default)
-- CREATE EVENT cleanup_old_sessions
-- ON SCHEDULE EVERY 1 DAY
-- DO DELETE FROM sessions WHERE last_activity < UNIX_TIMESTAMP(DATE_SUB(NOW(), INTERVAL 30 DAY));

EOF

echo "✓ Schema created"
echo ""

echo "[2/6] Generating users ($USERS rows)..."
$MYSQL_CMD <<EOF
INSERT INTO users (name, email, password_hash, created_at)
SELECT
    CONCAT('User ', n) as name,
    CONCAT('user', n, '@example.com') as email,
    MD5(CONCAT('password', n)) as password_hash,
    DATE_SUB(NOW(), INTERVAL FLOOR(RAND() * 365) DAY) as created_at
FROM (
    SELECT a.N + b.N * 10 + c.N * 100 + d.N * 1000 + e.N * 10000 + f.N * 100000 + g.N * 1000000 AS n
    FROM
        (SELECT 0 AS N UNION SELECT 1 UNION SELECT 2 UNION SELECT 3 UNION SELECT 4 UNION SELECT 5 UNION SELECT 6 UNION SELECT 7 UNION SELECT 8 UNION SELECT 9) a,
        (SELECT 0 AS N UNION SELECT 1 UNION SELECT 2 UNION SELECT 3 UNION SELECT 4 UNION SELECT 5 UNION SELECT 6 UNION SELECT 7 UNION SELECT 8 UNION SELECT 9) b,
        (SELECT 0 AS N UNION SELECT 1 UNION SELECT 2 UNION SELECT 3 UNION SELECT 4 UNION SELECT 5 UNION SELECT 6 UNION SELECT 7 UNION SELECT 8 UNION SELECT 9) c,
        (SELECT 0 AS N UNION SELECT 1 UNION SELECT 2 UNION SELECT 3 UNION SELECT 4 UNION SELECT 5 UNION SELECT 6 UNION SELECT 7 UNION SELECT 8 UNION SELECT 9) d,
        (SELECT 0 AS N UNION SELECT 1 UNION SELECT 2 UNION SELECT 3 UNION SELECT 4 UNION SELECT 5 UNION SELECT 6 UNION SELECT 7 UNION SELECT 8 UNION SELECT 9) e,
        (SELECT 0 AS N UNION SELECT 1 UNION SELECT 2 UNION SELECT 3 UNION SELECT 4 UNION SELECT 5 UNION SELECT 6 UNION SELECT 7 UNION SELECT 8 UNION SELECT 9) f,
        (SELECT 0 AS N UNION SELECT 1 UNION SELECT 2 UNION SELECT 3 UNION SELECT 4 UNION SELECT 5 UNION SELECT 6 UNION SELECT 7 UNION SELECT 8 UNION SELECT 9) g
) numbers
WHERE n > 0 AND n <= $USERS;
EOF
echo "✓ Users created"
echo ""

echo "[3/6] Generating products..."
$MYSQL_CMD <<EOF
INSERT INTO products (name, description, price, stock)
SELECT
    CONCAT('Product ', n) as name,
    CONCAT('Description for product ', n) as description,
    ROUND(RAND() * 1000, 2) as price,
    FLOOR(RAND() * 1000) as stock
FROM (
    SELECT a.N + b.N * 10 + c.N * 100 AS n
    FROM
        (SELECT 0 AS N UNION SELECT 1 UNION SELECT 2 UNION SELECT 3 UNION SELECT 4 UNION SELECT 5 UNION SELECT 6 UNION SELECT 7 UNION SELECT 8 UNION SELECT 9) a,
        (SELECT 0 AS N UNION SELECT 1 UNION SELECT 2 UNION SELECT 3 UNION SELECT 4 UNION SELECT 5 UNION SELECT 6 UNION SELECT 7 UNION SELECT 8 UNION SELECT 9) b,
        (SELECT 0 AS N UNION SELECT 1 UNION SELECT 2 UNION SELECT 3 UNION SELECT 4 UNION SELECT 5 UNION SELECT 6 UNION SELECT 7 UNION SELECT 8 UNION SELECT 9) c
) numbers
WHERE n > 0 AND n <= 1000;
EOF
echo "✓ Products created"
echo ""

echo "[4/6] Generating orders ($ORDERS rows) - this may take a while..."
$MYSQL_CMD <<EOF
INSERT INTO orders (user_id, total, status, created_at)
SELECT
    FLOOR(1 + RAND() * $USERS) as user_id,
    ROUND(RAND() * 1000, 2) as total,
    ELT(1 + FLOOR(RAND() * 4), 'pending', 'processing', 'completed', 'cancelled') as status,
    DATE_SUB(NOW(), INTERVAL FLOOR(RAND() * 180) DAY) as created_at
FROM (
    SELECT a.N + b.N * 10 + c.N * 100 + d.N * 1000 + e.N * 10000 + f.N * 100000 + g.N * 1000000 AS n
    FROM
        (SELECT 0 AS N UNION SELECT 1 UNION SELECT 2 UNION SELECT 3 UNION SELECT 4 UNION SELECT 5 UNION SELECT 6 UNION SELECT 7 UNION SELECT 8 UNION SELECT 9) a,
        (SELECT 0 AS N UNION SELECT 1 UNION SELECT 2 UNION SELECT 3 UNION SELECT 4 UNION SELECT 5 UNION SELECT 6 UNION SELECT 7 UNION SELECT 8 UNION SELECT 9) b,
        (SELECT 0 AS N UNION SELECT 1 UNION SELECT 2 UNION SELECT 3 UNION SELECT 4 UNION SELECT 5 UNION SELECT 6 UNION SELECT 7 UNION SELECT 8 UNION SELECT 9) c,
        (SELECT 0 AS N UNION SELECT 1 UNION SELECT 2 UNION SELECT 3 UNION SELECT 4 UNION SELECT 5 UNION SELECT 6 UNION SELECT 7 UNION SELECT 8 UNION SELECT 9) d,
        (SELECT 0 AS N UNION SELECT 1 UNION SELECT 2 UNION SELECT 3 UNION SELECT 4 UNION SELECT 5 UNION SELECT 6 UNION SELECT 7 UNION SELECT 8 UNION SELECT 9) e,
        (SELECT 0 AS N UNION SELECT 1 UNION SELECT 2 UNION SELECT 3 UNION SELECT 4 UNION SELECT 5 UNION SELECT 6 UNION SELECT 7 UNION SELECT 8 UNION SELECT 9) f,
        (SELECT 0 AS N UNION SELECT 1 UNION SELECT 2 UNION SELECT 3 UNION SELECT 4 UNION SELECT 5 UNION SELECT 6 UNION SELECT 7 UNION SELECT 8 UNION SELECT 9) g
) numbers
WHERE n > 0 AND n <= $ORDERS;
EOF
echo "✓ Orders created"
echo ""

echo "[5/6] Generating audit logs ($AUDITS rows) - noisy table..."
$MYSQL_CMD <<EOF
INSERT INTO audits (user_type, user_id, event, auditable_type, auditable_id, old_values, new_values, ip_address, user_agent, created_at)
SELECT
    'User' as user_type,
    FLOOR(1 + RAND() * $USERS) as user_id,
    ELT(1 + FLOOR(RAND() * 5), 'created', 'updated', 'deleted', 'viewed', 'exported') as event,
    ELT(1 + FLOOR(RAND() * 3), 'Order', 'Product', 'User') as auditable_type,
    FLOOR(1 + RAND() * 1000) as auditable_id,
    JSON_OBJECT('field', 'value') as old_values,
    JSON_OBJECT('field', 'new_value') as new_values,
    CONCAT(FLOOR(RAND()*255), '.', FLOOR(RAND()*255), '.', FLOOR(RAND()*255), '.', FLOOR(RAND()*255)) as ip_address,
    'Mozilla/5.0 (Test Browser)' as user_agent,
    DATE_SUB(NOW(), INTERVAL FLOOR(RAND() * 90) DAY) as created_at
FROM (
    SELECT a.N + b.N * 10 + c.N * 100 + d.N * 1000 + e.N * 10000 + f.N * 100000 + g.N * 1000000 AS n
    FROM
        (SELECT 0 AS N UNION SELECT 1 UNION SELECT 2 UNION SELECT 3 UNION SELECT 4 UNION SELECT 5 UNION SELECT 6 UNION SELECT 7 UNION SELECT 8 UNION SELECT 9) a,
        (SELECT 0 AS N UNION SELECT 1 UNION SELECT 2 UNION SELECT 3 UNION SELECT 4 UNION SELECT 5 UNION SELECT 6 UNION SELECT 7 UNION SELECT 8 UNION SELECT 9) b,
        (SELECT 0 AS N UNION SELECT 1 UNION SELECT 2 UNION SELECT 3 UNION SELECT 4 UNION SELECT 5 UNION SELECT 6 UNION SELECT 7 UNION SELECT 8 UNION SELECT 9) c,
        (SELECT 0 AS N UNION SELECT 1 UNION SELECT 2 UNION SELECT 3 UNION SELECT 4 UNION SELECT 5 UNION SELECT 6 UNION SELECT 7 UNION SELECT 8 UNION SELECT 9) d,
        (SELECT 0 AS N UNION SELECT 1 UNION SELECT 2 UNION SELECT 3 UNION SELECT 4 UNION SELECT 5 UNION SELECT 6 UNION SELECT 7 UNION SELECT 8 UNION SELECT 9) e,
        (SELECT 0 AS N UNION SELECT 1 UNION SELECT 2 UNION SELECT 3 UNION SELECT 4 UNION SELECT 5 UNION SELECT 6 UNION SELECT 7 UNION SELECT 8 UNION SELECT 9) f,
        (SELECT 0 AS N UNION SELECT 1 UNION SELECT 2 UNION SELECT 3 UNION SELECT 4 UNION SELECT 5 UNION SELECT 6 UNION SELECT 7 UNION SELECT 8 UNION SELECT 9) g
) numbers
WHERE n > 0 AND n <= $AUDITS;
EOF
echo "✓ Audits created"
echo ""

echo "[6/6] Generating other noisy tables..."
$MYSQL_CMD <<'EOF'
-- Sessions
INSERT INTO sessions (id, user_id, ip_address, user_agent, payload, last_activity)
SELECT
    MD5(CONCAT('session', n)) as id,
    FLOOR(1 + RAND() * 1000) as user_id,
    CONCAT(FLOOR(RAND()*255), '.', FLOOR(RAND()*255), '.', FLOOR(RAND()*255), '.', FLOOR(RAND()*255)) as ip_address,
    'Mozilla/5.0 (Test)' as user_agent,
    '{"data":"test"}' as payload,
    UNIX_TIMESTAMP(DATE_SUB(NOW(), INTERVAL FLOOR(RAND() * 7) DAY)) as last_activity
FROM (
    SELECT a.N + b.N * 10 + c.N * 100 + d.N * 1000 AS n
    FROM
        (SELECT 0 AS N UNION SELECT 1 UNION SELECT 2 UNION SELECT 3 UNION SELECT 4 UNION SELECT 5 UNION SELECT 6 UNION SELECT 7 UNION SELECT 8 UNION SELECT 9) a,
        (SELECT 0 AS N UNION SELECT 1 UNION SELECT 2 UNION SELECT 3 UNION SELECT 4 UNION SELECT 5 UNION SELECT 6 UNION SELECT 7 UNION SELECT 8 UNION SELECT 9) b,
        (SELECT 0 AS N UNION SELECT 1 UNION SELECT 2 UNION SELECT 3 UNION SELECT 4 UNION SELECT 5 UNION SELECT 6 UNION SELECT 7 UNION SELECT 8 UNION SELECT 9) c,
        (SELECT 0 AS N UNION SELECT 1 UNION SELECT 2 UNION SELECT 3 UNION SELECT 4 UNION SELECT 5 UNION SELECT 6 UNION SELECT 7 UNION SELECT 8 UNION SELECT 9) d
) numbers
WHERE n > 0 AND n <= 10000;

-- Cache
INSERT INTO cache (`key`, value, expiration)
SELECT
    CONCAT('cache:', MD5(CONCAT('key', n))) as `key`,
    CONCAT('{"cached_data":"', REPEAT('x', 100), '"}') as value,
    UNIX_TIMESTAMP(DATE_ADD(NOW(), INTERVAL FLOOR(RAND() * 24) HOUR)) as expiration
FROM (
    SELECT a.N + b.N * 10 + c.N * 100 AS n
    FROM
        (SELECT 0 AS N UNION SELECT 1 UNION SELECT 2 UNION SELECT 3 UNION SELECT 4 UNION SELECT 5 UNION SELECT 6 UNION SELECT 7 UNION SELECT 8 UNION SELECT 9) a,
        (SELECT 0 AS N UNION SELECT 1 UNION SELECT 2 UNION SELECT 3 UNION SELECT 4 UNION SELECT 5 UNION SELECT 6 UNION SELECT 7 UNION SELECT 8 UNION SELECT 9) b,
        (SELECT 0 AS N UNION SELECT 1 UNION SELECT 2 UNION SELECT 3 UNION SELECT 4 UNION SELECT 5 UNION SELECT 6 UNION SELECT 7 UNION SELECT 8 UNION SELECT 9) c
) numbers
WHERE n > 0 AND n <= 5000;

-- Telescope entries
INSERT INTO telescope_entries (uuid, batch_id, family_hash, type, content, created_at)
SELECT
    UUID() as uuid,
    UUID() as batch_id,
    MD5(CONCAT('family', FLOOR(n/100))) as family_hash,
    ELT(1 + FLOOR(RAND() * 4), 'request', 'query', 'exception', 'log') as type,
    JSON_OBJECT('data', REPEAT('x', 500)) as content,
    DATE_SUB(NOW(), INTERVAL FLOOR(RAND() * 30) DAY) as created_at
FROM (
    SELECT a.N + b.N * 10 + c.N * 100 + d.N * 1000 AS n
    FROM
        (SELECT 0 AS N UNION SELECT 1 UNION SELECT 2 UNION SELECT 3 UNION SELECT 4 UNION SELECT 5 UNION SELECT 6 UNION SELECT 7 UNION SELECT 8 UNION SELECT 9) a,
        (SELECT 0 AS N UNION SELECT 1 UNION SELECT 2 UNION SELECT 3 UNION SELECT 4 UNION SELECT 5 UNION SELECT 6 UNION SELECT 7 UNION SELECT 8 UNION SELECT 9) b,
        (SELECT 0 AS N UNION SELECT 1 UNION SELECT 2 UNION SELECT 3 UNION SELECT 4 UNION SELECT 5 UNION SELECT 6 UNION SELECT 7 UNION SELECT 8 UNION SELECT 9) c,
        (SELECT 0 AS N UNION SELECT 1 UNION SELECT 2 UNION SELECT 3 UNION SELECT 4 UNION SELECT 5 UNION SELECT 6 UNION SELECT 7 UNION SELECT 8 UNION SELECT 9) d
) numbers
WHERE n > 0 AND n <= 20000;
EOF
echo "✓ Noisy tables populated"
echo ""

echo "==================================="
echo "Database Summary"
echo "==================================="
$MYSQL_CMD -e "
SELECT 
    table_name AS 'Table',
    table_rows AS 'Rows (Est.)',
    ROUND(data_length / 1024 / 1024, 2) AS 'Size (MB)'
FROM information_schema.tables 
WHERE table_schema = '$DATABASE'
ORDER BY data_length DESC;
"

echo ""
echo "==================================="
echo "✓ Sample data generation complete!"
echo "==================================="
echo ""
echo "Next steps:"
echo "1. Test dbdump: ./bin/dbdump dump -H $HOST -P $PORT -u root -p $PASSWORD -d $DATABASE --auto"
echo "2. Verify excluded tables: audits, sessions, cache, telescope_entries"
echo "3. Verify triggers/procedures are included in structure dump"
echo ""
