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
		db := NewMongoDb(uri, "test_learning_platform")
		assert.NotNil(t, db)
		assert.Equal(t, "test_learning_platform", db.Name())
	})
}
