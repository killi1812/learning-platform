package app

import (
	"errors"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

// LoadConfig loads in program configuration should be a first thing called in the program
func LoadConfig() {
	zap.S().Debugf("Loading env variables")

	if err := godotenv.Load("../.env"); err != nil {
		zap.S().DPanicf("Env load err = %+v\n", err)
		zap.S().Infof("Can't load config using real env")
	}

	// App config
	Port = loadInt("PORT")

	// Secrets
	AccessKey = loadString("ACCESS_KEY")
	RefreshKey = loadString("REFRESH_KEY")

	if err := ValidateSecrets(AccessKey, RefreshKey); err != nil {
		if Build == BuildProd {
			zap.S().Panicf("Fatal: Insecure JWT configuration in production: %v", err)
		} else {
			zap.S().Warnf("⚠️ Insecure JWT configuration in dev: %v", err)
		}
	}

	MongoConn = loadString("MONGO_CONN")
	DatabaseConn = loadString("DATABASE_CONN")
	SeaweedFSEndpoint = loadString("SEAWEEDFS_ENDPOINT")
	RabbitMqConn = loadString("RABBITMQ_CONN")
	LogQuery = loadBool("LOG_QUERY")

	zap.S().Debugf("Finished loading env variables")
}

// ValidateSecrets checks that ACCESS_KEY and REFRESH_KEY meet security standards.
func ValidateSecrets(accessKey, refreshKey string) error {
	if strings.TrimSpace(accessKey) == "" {
		return errors.New("ACCESS_KEY must not be empty")
	}
	if strings.TrimSpace(refreshKey) == "" {
		return errors.New("REFRESH_KEY must not be empty")
	}
	if accessKey == "your-access-key-here" || refreshKey == "your-refresh-key-here" {
		return errors.New("JWT keys must not use default placeholder values")
	}
	if len(accessKey) < 32 {
		return errors.New("ACCESS_KEY must be at least 32 characters long")
	}
	if len(refreshKey) < 32 {
		return errors.New("REFRESH_KEY must be at least 32 characters long")
	}
	if accessKey == refreshKey {
		return errors.New("ACCESS_KEY and REFRESH_KEY must be different")
	}
	return nil
}

func loadInt(name string) int {
	rez := os.Getenv(name)
	if rez == "" {
		zap.S().Errorf("Env variable %s is empty\n", name)
	}

	num, err := strconv.Atoi(rez)
	if err != nil {
		zap.S().Errorf("Failed to parse int %s, will use default (0)\n", rez)
		return 0
	}

	zap.S().Debugf("Loaded %s = %d", name, num)
	return num
}

func loadString(name string) string {
	rez := strings.TrimSpace(os.Getenv(name))
	if rez == "" {
		zap.S().Errorf("Env variable %s is empty", name)
		return rez
	}
	upper := strings.ToUpper(name)
	if strings.Contains(upper, "KEY") || strings.Contains(upper, "SECRET") || strings.Contains(upper, "CONN") || strings.Contains(upper, "PASSWORD") || strings.Contains(upper, "TOKEN") {
		zap.S().Debugf("Loaded %s = [REDACTED]", name)
	} else {
		zap.S().Debugf("Loaded %s = %s", name, rez)
	}
	return rez
}

func loadBool(name string) bool {
	rez := os.Getenv(name)
	if rez == "" {
		zap.S().Errorf("Env variable %s is empty", name)
	}
	return rez == "true" || rez == "True"
}
