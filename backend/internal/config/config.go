package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort      string
	DBHost          string
	DBPort          string
	DBUser          string
	DBPassword      string
	DBName          string
	JWTSecret       string
	JWTExpiryHours      int
	EncryptionKey       string
	SyncEnabled         bool
	SyncIntervalCommits time.Duration
	SyncIntervalJira    time.Duration
	ReportAutoGenerate  bool
	ReportAutoTime      string
	R2AccountID         string
	R2AccessKeyID       string
	R2SecretAccessKey   string
	R2BucketName        string
	R2PublicDomain      string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	expiryHours, _ := strconv.Atoi(getEnv("JWT_EXPIRY_HOURS", "72"))

	cfg := &Config{
		ServerPort:     getEnv("SERVER_PORT", "8080"),
		DBHost:         getEnv("DB_HOST", "localhost"),
		DBPort:         getEnv("DB_PORT", "3306"),
		DBUser:         getEnv("DB_USER", "pdt"),
		DBPassword:     getEnv("DB_PASSWORD", ""),
		DBName:         getEnv("DB_NAME", "pdt"),
		JWTSecret:      getEnv("JWT_SECRET", ""),
		JWTExpiryHours: expiryHours,
		EncryptionKey:  getEnv("ENCRYPTION_KEY", ""),
	}

	syncEnabled := getEnv("SYNC_ENABLED", "true")
	cfg.SyncEnabled = syncEnabled == "true" || syncEnabled == "1"

	commitInterval, err := time.ParseDuration(getEnv("SYNC_INTERVAL_COMMITS", "15m"))
	if err != nil {
		commitInterval = 15 * time.Minute
	}
	cfg.SyncIntervalCommits = commitInterval

	jiraInterval, err := time.ParseDuration(getEnv("SYNC_INTERVAL_JIRA", "30m"))
	if err != nil {
		jiraInterval = 30 * time.Minute
	}
	cfg.SyncIntervalJira = jiraInterval

	reportAutoGen := getEnv("REPORT_AUTO_GENERATE", "true")
	cfg.ReportAutoGenerate = reportAutoGen == "true" || reportAutoGen == "1"
	cfg.ReportAutoTime = getEnv("REPORT_AUTO_TIME", "23:00")

	cfg.R2AccountID = getEnv("R2_ACCOUNT_ID", "")
	cfg.R2AccessKeyID = getEnv("R2_ACCESS_KEY_ID", "")
	cfg.R2SecretAccessKey = getEnv("R2_SECRET_ACCESS_KEY", "")
	cfg.R2BucketName = getEnv("R2_BUCKET_NAME", "")
	cfg.R2PublicDomain = getEnv("R2_PUBLIC_DOMAIN", "")

	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}
	if cfg.EncryptionKey == "" {
		return nil, fmt.Errorf("ENCRYPTION_KEY is required")
	}

	return cfg, nil
}

func (c *Config) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName)
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}
