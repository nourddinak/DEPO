package db

import (
	"path/filepath"
	"testing"
	"time"

	"depo/internal/crypto"
)

func setupTestDB(t *testing.T) *DB {
	t.Helper()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_depo.db")

	database, err := Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	t.Cleanup(func() {
		database.Close()
	})

	return database
}

func TestDatabaseMigrationAndUserCRUD(t *testing.T) {
	database := setupTestDB(t)

	count, err := database.CountUsers()
	if err != nil {
		t.Fatalf("CountUsers failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 users, got %d", count)
	}

	hash, _ := crypto.HashPassword("TestPass1234!")
	user, err := database.CreateUser("owner@depo.local", hash, RoleOwner)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	if user.Email != "owner@depo.local" || user.Role != RoleOwner {
		t.Fatalf("unexpected user fields: %+v", user)
	}

	// Verify lookup
	fetched, err := database.GetUserByEmail("owner@depo.local")
	if err != nil {
		t.Fatalf("GetUserByEmail failed: %v", err)
	}
	if fetched.ID != user.ID {
		t.Fatalf("expected user ID %s, got %s", user.ID, fetched.ID)
	}
}

func TestServerAndJobLeaseWorkflow(t *testing.T) {
	database := setupTestDB(t)

	_, enrHash, _ := crypto.GenerateEnrollmentToken()
	server, err := database.CreateServer("vps-lon-1", "192.168.1.100", "linux", enrHash)
	if err != nil {
		t.Fatalf("CreateServer failed: %v", err)
	}

	// Complete enrollment
	_, authHash, _ := crypto.GenerateAgentAuthToken()
	if err := database.CompleteServerEnrollment(server.ID, authHash, "v1.0.0"); err != nil {
		t.Fatalf("CompleteServerEnrollment failed: %v", err)
	}

	updatedServer, err := database.GetServerByID(server.ID)
	if err != nil {
		t.Fatalf("GetServerByID failed: %v", err)
	}
	if updatedServer.Status != ServerStatusOnline {
		t.Fatalf("expected online status, got %s", updatedServer.Status)
	}

	// Create and lease a job
	payload := `{"project_id":"prj-123","release_id":"rel-456"}`
	job, err := database.CreateJob(server.ID, JobDeployRelease, payload)
	if err != nil {
		t.Fatalf("CreateJob failed: %v", err)
	}

	// Lease job
	leasedJob, err := database.LeaseJob(server.ID, 30*time.Second)
	if err != nil {
		t.Fatalf("LeaseJob failed: %v", err)
	}
	if leasedJob == nil || leasedJob.ID != job.ID {
		t.Fatalf("expected leased job %s, got %+v", job.ID, leasedJob)
	}
	if leasedJob.LeaseToken == "" {
		t.Fatal("expected non-empty lease token")
	}

	// Completing job
	if err := database.CompleteJob(leasedJob.ID, leasedJob.LeaseToken); err != nil {
		t.Fatalf("CompleteJob failed: %v", err)
	}

	// Subsequent lease should return nil
	emptyJob, err := database.LeaseJob(server.ID, 30*time.Second)
	if err != nil {
		t.Fatalf("LeaseJob on empty queue failed: %v", err)
	}
	if emptyJob != nil {
		t.Fatalf("expected nil job, got %+v", emptyJob)
	}
}

func TestWebhookDeduplication(t *testing.T) {
	database := setupTestDB(t)

	isNew, err := database.RecordWebhookEvent("delivery-12345", "github", "push", "sha256-payload-hash")
	if err != nil {
		t.Fatalf("RecordWebhookEvent failed: %v", err)
	}
	if !isNew {
		t.Fatal("expected first event to be new")
	}

	// Duplicate
	isNewSecond, err := database.RecordWebhookEvent("delivery-12345", "github", "push", "sha256-payload-hash")
	if err != nil {
		t.Fatalf("RecordWebhookEvent duplicate failed: %v", err)
	}
	if isNewSecond {
		t.Fatal("expected duplicate event to be rejected as not new")
	}
}
