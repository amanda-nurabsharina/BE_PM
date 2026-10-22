package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort        string
	DBDriver       string
	DBHost         string
	DBPort         string
	DBUser         string
	DBPassword     string
	DBName         string
	DBSSLMode      string
	SQLitePath     string
	JWTSecret      string
	AllowedOrigins string
	RateLimitMax   int
	UploadDir      string
	MaxUploadSize  int64
}

func LoadConfig() *Config {
	_ = godotenv.Load()

	rateMax, err := strconv.Atoi(getEnv("RATE_LIMIT_MAX", "100"))
	if err != nil {
		rateMax = 100
	}

	maxSize, err := strconv.ParseInt(getEnv("MAX_UPLOAD_SIZE", "2097152"), 10, 64) // 2MB default
	if err != nil {
		maxSize = 2 * 1024 * 1024
	}

	return &Config{
		AppPort:        getEnv("APP_PORT", "8080"),
		DBDriver:       getEnv("DB_DRIVER", "sqlite"),
		DBHost:         getEnv("DB_HOST", "localhost"),
		DBPort:         getEnv("DB_PORT", "5432"),
		DBUser:         getEnv("DB_USER", "postgres"),
		DBPassword:     getEnv("DB_PASSWORD", "postgres"),
		DBName:         getEnv("DB_NAME", "creatortask_db"),
		DBSSLMode:      getEnv("DB_SSLMODE", "disable"),
		SQLitePath:     getEnv("SQLITE_PATH", "creatortask.db"),
		JWTSecret:      getEnv("JWT_SECRET", "super-secret-creatortask-jwt-key-2026"),
		AllowedOrigins: getEnv("ALLOWED_ORIGINS", "http://localhost:3000,http://127.0.0.1:3000"),
		RateLimitMax:   rateMax,
		UploadDir:      getEnv("UPLOAD_DIR", "./uploads"),
		MaxUploadSize:  maxSize,
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
