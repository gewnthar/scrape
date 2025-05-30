// backend/database/connection.go
package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/gewnthar/scrape/backend/config" // Adjust to your module path
	_ "github.com/go-sql-driver/mysql"         // MariaDB driver
)

var DB *sql.DB

// InitDB initializes the database connection pool.
func InitDB(cfg config.DatabaseConfig) error {
	var err error
	// DSN: username:password@protocol(address)/dbname?param=value
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.DBName,
	)

	DB, err = sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection pool settings
	DB.SetMaxOpenConns(25)
	DB.SetMaxIdleConns(25)
	DB.SetConnMaxLifetime(5 * time.Minute)

	// Ping the database to verify connection
	err = DB.Ping()
	if err != nil {
		DB.Close() // Close the connection if ping fails
		return fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("Successfully connected to the database!")
	return nil
}

// CloseDB closes the database connection pool.
// Typically called on application shutdown.
func CloseDB() {
	if DB != nil {
		DB.Close()
		log.Println("Database connection closed.")
	}
}