package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	MONGO_CONNECTION_URL string
}

var AppConfig Config

func LoadConfig() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalln("error loading from .env")
	}
	AppConfig = Config{
		MONGO_CONNECTION_URL: os.Getenv("MONGO_CONNECTION_URL"),
	}
	log.Println("Configuration loaded.")
}
