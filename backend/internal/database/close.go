package database

import (
	"gorm.io/gorm"
)

// Close flushes the SQL connection pool.
func Close(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
