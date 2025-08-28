package config

import (
	"log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"github.com/nabilulilalbab/bottele/models"
)

var DB *gorm.DB

func ConnectDatabase() {
	db, err := gorm.Open(sqlite.Open("grnstore.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("Gagal terhubung ke database:", err)
	}
	
	// Run migrations
	err = models.AutoMigrate(db)
	if err != nil {
		log.Fatal("Gagal melakukan migrasi database:", err)
	}
	
	DB = db
	log.Println("Database connected and migrated successfully")
}
