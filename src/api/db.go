package api

import (
	"log"

	"github.com/OpenFactorioServerManager/factorio-server-manager/bootstrap"
	"github.com/OpenFactorioServerManager/factorio-server-manager/factorio"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

var globalDB *gorm.DB

func GetDB() *gorm.DB {
	return globalDB
}

func SetupDB() {
	config := bootstrap.GetConfig()

	db, err := gorm.Open(sqlite.Open(config.SQLiteDatabaseFile), &gorm.Config{})
	if err != nil {
		log.Printf("Error opening sqlite database: %s", err)
		panic(err)
	}

	err = db.AutoMigrate(
		&factorio.ModAsset{},
		&factorio.ModPreset{},
		&factorio.ModPresetItem{},
		&factorio.ServerModManifest{},
		&factorio.ServerModManifestItem{},
	)
	if err != nil {
		log.Printf("Error AutoMigrating mod tables: %s", err)
		panic(err)
	}

	globalDB = db

	if err := factorio.EnsureDLCAssets(db); err != nil {
		log.Printf("Warning: failed to ensure DLC assets: %s", err)
	}

	// Clean up orphaned manifest items (mod deleted from library but still in manifest)
	db.Exec(`DELETE FROM server_mod_manifest_items WHERE mod_asset_id NOT IN (SELECT id FROM mod_assets WHERE deleted_at IS NULL)`)

	log.Println("Database initialized")
}
