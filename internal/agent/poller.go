package agent

import (
	"context"
	"log"
	"time"
)

type Daemon struct {
	cfg      *Config
	client   *Client
	executor *Executor
}

func NewDaemon(cfg *Config) *Daemon {
	client := NewClient(cfg.ServerURL, cfg.AuthToken)
	caddyManager := NewCaddyManager(cfg.CaddyConfig)
	executor := NewExecutor(client, caddyManager, cfg.DataDir)

	return &Daemon{
		cfg:      cfg,
		client:   client,
		executor: executor,
	}
}

func (d *Daemon) Run(ctx context.Context) error {
	log.Printf("[agent] Starting DEPO agent %s against %s (server ID: %s)",
		d.cfg.AgentVersion, d.cfg.ServerURL, d.cfg.ServerID)

	// Start heartbeat goroutine
	go d.startHeartbeatLoop(ctx)

	// Start long-polling loop
	return d.startPollingLoop(ctx)
}

func (d *Daemon) startHeartbeatLoop(ctx context.Context) {
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()

	// Initial heartbeat immediately
	d.sendHeartbeat()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			d.sendHeartbeat()
		}
	}
}

func (d *Daemon) sendHeartbeat() {
	metrics := CollectSystemMetrics("/")
	if err := d.client.Heartbeat(metrics.CPUPct, metrics.RAMPct, metrics.DiskPct, metrics.DiskFreeBytes); err != nil {
		log.Printf("[agent] Heartbeat warning: %v", err)
	}
}

func (d *Daemon) startPollingLoop(ctx context.Context) error {
	backoff := 1 * time.Second
	maxBackoff := 30 * time.Second

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		claim, err := d.client.PollJobs(ctx)
		if err != nil {
			log.Printf("[agent] Connection error polling jobs: %v (backing off for %s)", err, backoff)
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(backoff):
			}
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}

		// Reset backoff on successful HTTP communication
		backoff = 1 * time.Second

		if claim != nil {
			log.Printf("[agent] Claimed job %s (type: %s, lease: %s)", claim.JobID, claim.Type, safeShort(claim.LeaseToken, 8))
			if err := d.executor.ExecuteJob(claim); err != nil {
				log.Printf("[agent] Job execution failed: %v", err)
			} else {
				log.Printf("[agent] Job %s completed successfully", claim.JobID)
			}
		}
	}
}
