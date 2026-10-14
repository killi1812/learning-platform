package mongo

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestNewMongoDb(t *testing.T) {
	origLogger := zap.L()
	defer zap.ReplaceGlobals(origLogger)
	zap.ReplaceGlobals(zap.NewNop())

	uri := os.Getenv("MONGO_CONN")
	if uri == "" {
		uri = "mongodb://127.0.0.1:27017"
	}

	t.Run("Default parameter handling", func(t *testing.T) {
		db, err := NewMongoDb(uri, "test_learning_platform")
		assert.NoError(t, err)
		assert.NotNil(t, db)
		assert.Equal(t, "test_learning_platform", db.Name())
	})

	t.Run("Empty URI returns error", func(t *testing.T) {
		db, err := NewMongoDb("", "test_learning_platform")
		assert.Error(t, err)
		assert.Nil(t, db)
		assert.ErrorIs(t, err, ErrEmptyURI)
	})

	t.Run("Empty dbName returns error", func(t *testing.T) {
		db, err := NewMongoDb(uri, "")
		assert.Error(t, err)
		assert.Nil(t, db)
		assert.ErrorIs(t, err, ErrEmptyDbName)
	})
}
