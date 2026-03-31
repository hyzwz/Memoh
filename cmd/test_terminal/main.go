package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	pb "github.com/memohai/memoh/internal/mcp/mcpcontainer"
)

func main() {
	target := os.Args[1] // e.g. "10.88.0.2:9090"

	conn, err := grpc.NewClient(target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                10 * time.Second,
			Timeout:             5 * time.Second,
			PermitWithoutStream: true,
		}),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dial: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	client := pb.NewContainerServiceClient(conn)
	ctx := context.Background()

	stream, err := client.Exec(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "exec: %v\n", err)
		os.Exit(1)
	}

	err = stream.Send(&pb.ExecInput{
		Command: "/bin/sh",
		WorkDir: "/data",
		Pty:     true,
		Resize:  &pb.TerminalResize{Cols: 80, Rows: 24},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "send: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Stream opened, waiting for output...")
	start := time.Now()

	// Send periodic stdin to keep the stream active
	go func() {
		for {
			time.Sleep(5 * time.Second)
			elapsed := time.Since(start).Round(time.Second)
			if err := stream.Send(&pb.ExecInput{StdinData: []byte("\n")}); err != nil {
				fmt.Fprintf(os.Stderr, "[%v] stdin send failed: %v\n", elapsed, err)
				return
			}
			fmt.Printf("[%v] sent stdin newline\n", elapsed)
		}
	}()

	for {
		output, err := stream.Recv()
		if err != nil {
			fmt.Fprintf(os.Stderr, "[%v] recv error: %v\n", time.Since(start).Round(time.Millisecond), err)
			os.Exit(1)
		}
		switch output.GetStream() {
		case pb.ExecOutput_STDOUT, pb.ExecOutput_STDERR:
			if data := output.GetData(); len(data) > 0 {
				fmt.Printf("[%v] data(%d bytes): %q\n", time.Since(start).Round(time.Millisecond), len(data), string(data))
			}
		case pb.ExecOutput_EXIT:
			fmt.Printf("[%v] EXIT code=%d\n", time.Since(start).Round(time.Millisecond), output.GetExitCode())
			os.Exit(0)
		}

		if time.Since(start) > 40*time.Second {
			fmt.Println("SUCCESS: survived >40 seconds")
			os.Exit(0)
		}
	}
}
