package app

import (
	utilDb "git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/db"
	utilMongo "git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"gorm.io/gorm"
)

// NewGormDb initializes and returns the primary PostgreSQL connection via GORM.
func NewGormDb() *gorm.DB {
	return utilDb.NewPostgresDb(DatabaseConn, Build == BuildDev)
}

// NewMongoDb initializes and returns the MongoDB connection used for ephemeral session storage.
func NewMongoDb() *mongo.Database {
	return utilMongo.NewMongoDb(MongoConn, "learning_platform")
}

// HandleMigrations orchestrates GORM schema auto-migrations.
func HandleMigrations(gormDb *gorm.DB, runMigration bool, models ...any) error {
	return utilDb.HandleMigrations(gormDb, Build == BuildDev, runMigration, models...)
}
