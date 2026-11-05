package main

import (
	"context"
	"log"
	"os"

	yily "github.com/yannick2009/yily/sdk/go"
)

type Config struct {
	Host     string `yily:"DB_HOST"`
	Port     int    `yily:"DB_PORT"`
	Debug    bool   `yily:"DEBUG"`
	APIToken string `yily:"API_TOKEN,version=2"`
}

func main() {
	ctx := context.Background()

	client, err := yily.NewClient(yily.Config{
		BaseURL:       "http://localhost:16529",
		EnvironmentID: "00000000-0000-0000-0000-000000000000",
		Token:         os.Getenv("YILY_TOKEN"),
	})
	if err != nil {
		log.Fatalf("client error: %v", err)
	}

	var cfg Config
	if err := client.Load(ctx, &cfg); err != nil {
		log.Fatalf("load error: %v", err)
	}

	log.Println("configuration loaded")
}
