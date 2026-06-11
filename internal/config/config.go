package config

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	ApiPort string

	SecretKey string

	EvolutionApi string
	EvolutionKey string

	RedisHost string
	RedisPort string

	Pepper string

	EvolutionApiUrl string
}

func LoadConfig() (*Config, error) {
	// Carrega o arquivo .env
	_ = godotenv.Load(".env")

	config := &Config{
		DBHost:       getEnv("DB_HOST"),
		DBPort:       getEnv("DB_PORT"),
		DBUser:       getEnv("DB_USER"),
		DBPassword:   getEnv("DB_PASSWORD"),
		DBName:       getEnv("DB_NAME"),
		DBSSLMode:    getEnv("DB_SSLMODE"),
		ApiPort:      getEnv("API_PORT"),
		SecretKey:    getEnv("JWT_SECRET"),
		EvolutionApi: getEnv("EVOLUTION_API"),
		EvolutionKey: getEnv("EVOLUTION_KEY"),
		RedisHost:    getEnv("REDIS_HOST"),
		RedisPort:    getEnv("REDIS_PORT"),
		Pepper:       getEnv("PEPPER"),
		EvolutionApiUrl: getEnv("EVOLUTION_API_URL"),
	}

	return config, nil
}

// getEnv obtém o valor de uma variável de ambiente
// Se a variavel não existir ou estiver vazia, retorna um valor padrão
func getEnv(key string) string {
	if value := os.Getenv(key); value != "" {
		// Remove espaços em branco no início e fim
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}

	return ""
}
