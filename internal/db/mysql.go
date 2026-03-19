package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/DrummDaddy/task_service/internal/config"
	_ "github.com/go-sql-driver/mysql"
)

func OpenMySQL(cfg config.Config) (*sql.DB, error) {
	if cfg.MySQL.DSN == "" {
		return nil, fmt.Errorf("MySQL DSN is empty")
	}
	db, err := sql.Open("mysql", cfg.MySQL.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to open MySQL connection: %v", err)
	}
	maxLife := cfg.MySQL.ConnMaxLifetime
	if maxLife <= 0 {
		maxLife = 5 * time.Minute
	}
	if cfg.MySQL.MaxIdleConns > 0 {
		db.SetMaxIdleConns(cfg.MySQL.MaxIdleConns)
	}
	if cfg.MySQL.MaxOpenConns > 0 {
		db.SetMaxOpenConns(cfg.MySQL.MaxOpenConns)
	}
	db.SetConnMaxLifetime(maxLife)
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping MySQL: %v", err)
	}
	return db, nil
}
