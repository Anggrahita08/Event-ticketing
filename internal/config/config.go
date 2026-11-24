package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort       string
	DBHost        string
	DBPort        string
	DBUser        string
	DBPassword    string
	DBName        string
	DBSSLMode     string
	JWTSecret     string
	AdminEmail    string
	AdminPassword string
	WorkerCount   string
	QueueCapacity string
}

func LoadConfig() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Println("Peringatan: File .env tidak ditemukan, menggunakan environment variable sistem")
	}

	return &Config{
		AppPort:       getEnv("APP_PORT", "8080"),
		DBHost:        getEnv("DB_HOST", "localhost"),
		DBPort:        getEnv("DB_PORT", "5442"),
		DBUser:        getEnv("DB_USER", "kiko"),
		DBPassword:    getEnv("DB_PASSWORD", ""),
		DBName:        getEnv("DB_NAME", "ticketing_golang"),
		DBSSLMode:     getEnv("DB_SSLMODE", "disable"),
		JWTSecret:     getEnv("JWT_SECRET", "secret"),
		AdminEmail:    getEnv("ADMIN_EMAIL", "admin@example.com"),
		AdminPassword: getEnv("ADMIN_PASSWORD", "Admin12345"),
		WorkerCount:   getEnv("IMPORT_WORKER_COUNT", "4"),
		QueueCapacity: getEnv("IMPORT_QUEUE_CAPACITY", "10"),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
