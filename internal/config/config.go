package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv       string
	IsProduction bool

	DBSimpleProtocol bool
	DBHost           string
	DBPort           string
	DBUser           string
	DBPassword       string
	DBName           string
	DBSSLMode        string

	ApiPort string

	SecretKey string

	EvolutionApi string
	EvolutionKey string

	RedisHost string
	RedisPort string

	Pepper string

	EvolutionApiUrl string

	RabbitURL string

	StripeSecretKey     string
	StripeWebhookSecret string

	WebhookURLDiscord string

	SMTPHost     string
	SMTPPort     string
	SMTPUser     string
	SMTPPassword string
	SMTPFrom     string

	FrontendURL string

	SuperAdiminPepper string
}

func LoadConfig() (*Config, error) {
	// Carrega o arquivo .env
	_ = godotenv.Load(".env")

	config := &Config{
		AppEnv:              getEnv("APP_ENV"),
		DBHost:              getEnv("DB_HOST"),
		DBPort:              getEnv("DB_PORT"),
		DBUser:              getEnv("DB_USER"),
		DBPassword:          getEnv("DB_PASSWORD"),
		DBName:              getEnv("DB_NAME"),
		DBSSLMode:           getEnv("DB_SSLMODE"),
		ApiPort:             getEnv("API_PORT"),
		SecretKey:           getEnv("JWT_SECRET"),
		EvolutionApi:        getEnv("EVOLUTION_API"),
		EvolutionKey:        getEnv("EVOLUTION_KEY"),
		RedisHost:           getEnv("REDIS_HOST"),
		RedisPort:           getEnv("REDIS_PORT"),
		Pepper:              getEnv("PEPPER"),
		EvolutionApiUrl:     getEnv("EVOLUTION_API_URL"),
		RabbitURL:           getEnv("RABBITMQ_URL"),
		StripeSecretKey:     getEnv("STRIPE_SECRET_KEY"),
		StripeWebhookSecret: getEnv("STRIPE_WEBHOOK_SECRET"),
		WebhookURLDiscord:   getEnv("WEBHOOK_URL_DISCORD"),
		DBSimpleProtocol:    getEnvBool("DB_SIMPLE_PROTOCOL"),
		SMTPHost:            getEnv("SMTP_HOST"),
		SMTPPort:            getEnv("SMTP_PORT"),
		SMTPUser:            getEnv("SMTP_USER"),
		SMTPPassword:        getEnv("SMTP_PASSWORD"),
		SMTPFrom:            getEnv("SMTP_FROM"),
		FrontendURL:         getEnv("FRONTEND_URL"),
		SuperAdiminPepper:   getEnv("SUPER_ADMIN_PEPPER"),
		IsProduction:        getEnvBool("IS_PRODUCTION"),
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

func getEnvBool(key string) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return false
	}

	// ParseBool aceita: 1, t, T, TRUE, true, True, 0, f, F, FALSE, false, False
	b, err := strconv.ParseBool(value)
	if err != nil {
		return false
	}

	return b
}
