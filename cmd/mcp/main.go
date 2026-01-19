package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/memohai/memoh/internal/config"
	ctr "github.com/memohai/memoh/internal/containerd"
	"github.com/memohai/memoh/internal/db"
	"github.com/memohai/memoh/internal/mcp"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		return
	}

	ctx := context.Background()
	cfgPath := os.Getenv("CONFIG_PATH")
	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	factory := ctr.DefaultClientFactory{SocketPath: cfg.Containerd.SocketPath}
	client, err := factory.New(ctx)
	if err != nil {
		log.Fatalf("connect containerd: %v", err)
	}
	defer client.Close()

	service := ctr.NewDefaultService(client, cfg.Containerd.Namespace)
	manager := mcp.NewManager(service, cfg.MCP)

	switch os.Args[1] {
	case "init":
		if err := manager.Init(ctx); err != nil {
			log.Fatalf("init: %v", err)
		}
	case "list":
		users, err := manager.ListUsers(ctx)
		if err != nil {
			log.Fatalf("list: %v", err)
		}
		for _, user := range users {
			fmt.Println(user)
		}
	case "create":
		userID := argAt(2)
		if err := manager.EnsureUser(ctx, userID); err != nil {
			log.Fatalf("create: %v", err)
		}
	case "start":
		userID := argAt(2)
		if err := manager.Start(ctx, userID); err != nil {
			log.Fatalf("start: %v", err)
		}
	case "stop":
		stopCmd(ctx, manager, os.Args[2:])
	case "delete":
		userID := argAt(2)
		if err := manager.Delete(ctx, userID); err != nil {
			log.Fatalf("delete: %v", err)
		}
	case "exec":
		withDB(ctx, cfg.Postgres, manager, func() {
			execCmd(ctx, manager, os.Args[2:])
		})
	default:
		usage()
	}
}

func stopCmd(ctx context.Context, manager *mcp.Manager, args []string) {
	fs := flag.NewFlagSet("stop", flag.ExitOnError)
	timeout := fs.Duration("timeout", 10*time.Second, "stop timeout")
	fs.Parse(args)

	userID := fs.Arg(0)
	if userID == "" {
		log.Fatalf("stop: user id required")
	}

	if err := manager.Stop(ctx, userID, *timeout); err != nil {
		log.Fatalf("stop: %v", err)
	}
}

func execCmd(ctx context.Context, manager *mcp.Manager, args []string) {
	fs := flag.NewFlagSet("exec", flag.ExitOnError)
	var envs stringSlice
	cwd := fs.String("cwd", "", "working directory")
	tty := fs.Bool("tty", false, "allocate a tty")
	fs.Var(&envs, "env", "environment variable, can be repeated")
	fs.Parse(args)

	userID := fs.Arg(0)
	cmdArgs := fs.Args()[1:]
	if userID == "" || len(cmdArgs) == 0 {
		log.Fatalf("exec: user id and command required")
	}

	result, err := manager.Exec(ctx, mcp.ExecRequest{
		UserID:   userID,
		Command:  cmdArgs,
		Env:      envs,
		WorkDir:  *cwd,
		Terminal: *tty,
		UseStdio: true,
	})
	if err != nil {
		log.Fatalf("exec: %v", err)
	}
	if result.ExitCode != 0 {
		os.Exit(int(result.ExitCode))
	}
}

func argAt(index int) string {
	if len(os.Args) <= index {
		log.Fatalf("missing argument")
	}
	return os.Args[index]
}

type stringSlice []string

func (s *stringSlice) String() string {
	return fmt.Sprintf("%v", []string(*s))
}

func (s *stringSlice) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func usage() {
	fmt.Println("Usage: mcp <command> [args]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  init")
	fmt.Println("  list")
	fmt.Println("  create <userID>")
	fmt.Println("  start <userID>")
	fmt.Println("  stop <userID> [--timeout=10s]")
	fmt.Println("  delete <userID>")
	fmt.Println("  exec <userID> [--cwd=DIR] [--tty] [--env=K=V] -- <cmd> [args...]")
	fmt.Println("  version-create <userID>")
	fmt.Println("  version-list <userID>")
	fmt.Println("  version-rollback <userID> <version>")
}

func withDB(ctx context.Context, cfg config.PostgresConfig, manager *mcp.Manager, fn func()) {
	conn, err := db.Open(ctx, cfg)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer conn.Close()
	manager.WithDB(conn)
	fn()
}
