package main

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"github.com/memohai/memoh/internal/config"
	ctr "github.com/memohai/memoh/internal/containerd"
	"github.com/memohai/memoh/internal/db"
	"github.com/memohai/memoh/internal/handlers"
	"github.com/memohai/memoh/internal/mcp"
	"github.com/memohai/memoh/internal/memory"
	"github.com/memohai/memoh/internal/server"
)

func main() {
	ctx := context.Background()
	cfgPath := os.Getenv("CONFIG_PATH")
	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	if strings.TrimSpace(cfg.Auth.JWTSecret) == "" {
		log.Fatalf("jwt secret is required")
	}
	jwtExpiresIn, err := time.ParseDuration(cfg.Auth.JWTExpiresIn)
	if err != nil {
		log.Fatalf("invalid jwt expires in: %v", err)
	}

	addr := cfg.Server.Addr
	if value := os.Getenv("HTTP_ADDR"); value != "" {
		addr = value
	}

	factory := ctr.DefaultClientFactory{SocketPath: cfg.Containerd.SocketPath}
	client, err := factory.New(ctx)
	if err != nil {
		log.Fatalf("connect containerd: %v", err)
	}
	defer client.Close()

	service := ctr.NewDefaultService(client, cfg.Containerd.Namespace)
	manager := mcp.NewManager(service, cfg.MCP)

	conn, err := db.Open(ctx, cfg.Postgres)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer conn.Close()
	manager.WithDB(conn)

	pingHandler := handlers.NewPingHandler()
	authHandler := handlers.NewAuthHandler(conn, cfg.Auth.JWTSecret, jwtExpiresIn)
	llmClient := memory.NewLLMClient(
		cfg.Memory.BaseURL,
		cfg.Memory.APIKey,
		cfg.Memory.Model,
		time.Duration(cfg.Memory.TimeoutSeconds)*time.Second,
	)
	embedder := memory.NewOpenAIEmbedder(
		cfg.Embeddings.OpenAIAPIKey,
		cfg.Embeddings.OpenAIBaseURL,
		cfg.Embeddings.Model,
		cfg.Embeddings.Dimensions,
		time.Duration(cfg.Embeddings.TimeoutSeconds)*time.Second,
	)
	store, err := memory.NewQdrantStore(
		cfg.Qdrant.BaseURL,
		cfg.Qdrant.APIKey,
		cfg.Qdrant.Collection,
		cfg.Embeddings.Dimensions,
		time.Duration(cfg.Qdrant.TimeoutSeconds)*time.Second,
	)
	if err != nil {
		log.Fatalf("qdrant init: %v", err)
	}
	memoryService := memory.NewService(llmClient, embedder, store)
	memoryHandler := handlers.NewMemoryHandler(memoryService)
	fsHandler := handlers.NewFSHandler(service, manager, cfg.MCP, cfg.Containerd.Namespace)
	swaggerHandler := handlers.NewSwaggerHandler()
	srv := server.NewServer(addr, cfg.Auth.JWTSecret, pingHandler, authHandler, memoryHandler, fsHandler, swaggerHandler)

	if err := srv.Start(); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
