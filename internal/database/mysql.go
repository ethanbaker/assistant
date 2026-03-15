// mysql.go - contains initialization options to connect to a MySQL database
package database

import (
	"fmt"
	"log"
	"strings"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// MySQLConfig holds database connection configuration
type MySQLConfig struct {
	User            string
	Password        string
	Host            string
	Port            string
	Database        string
	Charset         string
	ParseTime       bool
	Loc             string
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime time.Duration
	LogLevel        string
}

// DSN builds and returns a MySQL DSN string from the configuration fields
func (c *MySQLConfig) DSN() string {
	// Set defaults for optional fields
	charset := c.Charset
	if charset == "" {
		charset = "utf8mb4"
	}

	parseTime := "true"
	if !c.ParseTime {
		parseTime = "false"
	}

	loc := c.Loc
	if loc == "" {
		loc = "Local"
	}

	// Build DSN: username:password@tcp(host:port)/database?params
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=%s&parseTime=%s&loc=%s",
		c.User,
		c.Password,
		c.Host,
		c.Port,
		c.Database,
		charset,
		parseTime,
		loc,
	)
}

// getLogLevel is a helper function to return the string log level as a gorm log level
func (c *MySQLConfig) getLogLevel() logger.LogLevel {
	switch strings.ToLower(strings.TrimSpace(c.LogLevel)) {
	case "info":
		return logger.Info
	case "warn":
		return logger.Warn
	case "error":
		return logger.Error
	case "silent":
		return logger.Silent
	default:
		return logger.Warn
	}
}

// New creates a new GORM database connection to MySQL
func NewMySQLConnection(cfg *MySQLConfig) (*gorm.DB, error) {
	if cfg.User == "" {
		return nil, fmt.Errorf("user is required")
	}
	if cfg.Host == "" {
		return nil, fmt.Errorf("host is required")
	}
	if cfg.Port == "" {
		return nil, fmt.Errorf("port is required")
	}
	if cfg.Database == "" {
		return nil, fmt.Errorf("database is required")
	}

	// Set defaults
	if cfg.MaxIdleConns == 0 {
		cfg.MaxIdleConns = 10
	}
	if cfg.MaxOpenConns == 0 {
		cfg.MaxOpenConns = 100
	}
	if cfg.ConnMaxLifetime == 0 {
		cfg.ConnMaxLifetime = time.Hour
	}

	// Configure GORM logger
	gormOptions := &gorm.Config{
		Logger: logger.Default.LogMode(cfg.getLogLevel()),
	}

	// Build DSN and open database connection
	dsn := cfg.DSN()
	db, err := gorm.Open(mysql.Open(dsn), gormOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Get underlying SQL database
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying database: %w", err)
	}

	// Set connection pool settings
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// Test the connection
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("database connection established successfully")

	return db, nil
}

// Close closes the database connection
func Close(db *gorm.DB) error {
	if db == nil {
		return nil
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying database: %w", err)
	}

	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("failed to close database: %w", err)
	}

	log.Println("database connection closed")

	return nil
}
