package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"depo/internal/agent"
)

var Version = "1.0.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	switch command {
	case "enroll":
		enrollCmd := flag.NewFlagSet("enroll", flag.ExitOnError)
		serverURL := enrollCmd.String("server", "", "Central DEPO control plane URL (e.g. https://depo.example.com)")
		token := enrollCmd.String("token", "", "One-time enrollment token generated from dashboard")
		configPath := enrollCmd.String("config", getDefaultConfigPath(), "Path to store agent config file")
		_ = enrollCmd.Parse(os.Args[2:])

		if *serverURL == "" || *token == "" {
			fmt.Println("Error: both --server and --token flags are required for enrollment")
			enrollCmd.PrintDefaults()
			os.Exit(1)
		}

		hostname, _ := os.Hostname()
		client := agent.NewClient(*serverURL, "")
		serverID, authToken, err := client.Enroll(*token, hostname, runtime.GOOS, Version)
		if err != nil {
			log.Fatalf("Enrollment failed: %v", err)
		}

		cfg := agent.DefaultConfig()
		cfg.ServerURL = *serverURL
		cfg.ServerID = serverID
		cfg.AuthToken = authToken
		cfg.AgentVersion = Version

		if err := cfg.Save(*configPath); err != nil {
			log.Fatalf("Failed to save config to %s: %v", *configPath, err)
		}

		fmt.Printf("Successfully enrolled server '%s' (ID: %s)!\nConfig saved to: %s\n", hostname, serverID, *configPath)

	case "run":
		runCmd := flag.NewFlagSet("run", flag.ExitOnError)
		configPath := runCmd.String("config", getDefaultConfigPath(), "Path to agent config file")
		_ = runCmd.Parse(os.Args[2:])

		cfg, err := agent.LoadConfig(*configPath)
		if err != nil {
			log.Fatalf("Failed to load agent configuration: %v\nRun 'depo-agent enroll' first.", err)
		}

		daemon := agent.NewDaemon(cfg)

		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()

		if err := daemon.Run(ctx); err != nil {
			log.Fatalf("Agent error: %v", err)
		}

	case "version":
		fmt.Printf("depo-agent version %s (%s/%s)\n", Version, runtime.GOOS, runtime.GOARCH)

	default:
		printUsage()
		os.Exit(1)
	}
}

func getDefaultConfigPath() string {
	if runtime.GOOS == "windows" {
		return "depo-agent.json"
	}
	return "/etc/depo-agent/config.json"
}

func printUsage() {
	fmt.Println("DEPO Agent - Outbound VPS Deployment Worker")
	fmt.Println("\nUsage:")
	fmt.Println("  depo-agent enroll --server <URL> --token <ENROLL_TOKEN> [--config <PATH>]")
	fmt.Println("  depo-agent run [--config <PATH>]")
	fmt.Println("  depo-agent version")
}
