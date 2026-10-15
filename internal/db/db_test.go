package db

import (
	"fmt"
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

func TestProjectDeployDirDefaultAndMigration(t *testing.T) {
	database := setupTestDB(t)

	// Create a test server first (foreign key requirement)
	_, enrHash, _ := crypto.GenerateEnrollmentToken()
	server, err := database.CreateServer("vps-test-deploydir", "192.168.1.101", "linux", enrHash)
	if err != nil {
		t.Fatalf("CreateServer failed: %v", err)
	}

	// 1. CreateProject normalization tests
	testCases := []struct {
		name      string
		deployDir string
		expected  string
	}{
		{"empty deploy dir", "", "/var/www/depo"},
		{"legacy /var/www", "/var/www", "/var/www/depo"},
		{"legacy /var/www/", "/var/www/", "/var/www/depo"},
		{"whitespace string", "   ", "/var/www/depo"},
		{"windows backslash", `\var\www`, "/var/www/depo"},
		{"windows backslash trailing", `\var\www\`, "/var/www/depo"},
		{"custom path preserved", "/opt/custom/app", "/opt/custom/app"},
	}

	for i, tc := range testCases {
		proj := &Project{
			Name:      fmt.Sprintf("test-proj-%d", i),
			ServerID:  server.ID,
			DeployDir: tc.deployDir,
		}
		if err := database.CreateProject(proj); err != nil {
			t.Fatalf("CreateProject failed for %s: %v", tc.name, err)
		}
		fetched, err := database.GetProjectByID(proj.ID)
		if err != nil {
			t.Fatalf("GetProjectByID failed for %s: %v", tc.name, err)
		}
		if fetched.DeployDir != tc.expected {
			t.Errorf("[%s] DeployDir = %q, expected %q", tc.name, fetched.DeployDir, tc.expected)
		}
	}

	// 2. UpdateProject normalization tests
	upProj := &Project{
		Name:      "test-update-deploydir",
		ServerID:  server.ID,
		DeployDir: "/opt/initial/path",
	}
	if err := database.CreateProject(upProj); err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}
	// Try updating to legacy /var/www -> should normalize to /var/www/depo
	upProj.DeployDir = "/var/www"
	if err := database.UpdateProject(upProj); err != nil {
		t.Fatalf("UpdateProject failed: %v", err)
	}
	fetchedUp, _ := database.GetProjectByID(upProj.ID)
	if fetchedUp.DeployDir != "/var/www/depo" {
		t.Errorf("UpdateProject with /var/www yielded %q, expected /var/www/depo", fetchedUp.DeployDir)
	}

	// 3. Insert raw legacy projects with /var/www, /var/www/, empty, and backslashes to test Migration 2 behavior
	legacyInserts := []struct {
		id  string
		dir string
	}{
		{"proj-mig-1", "/var/www"},
		{"proj-mig-2", "/var/www/"},
		{"proj-mig-3", ""},
		{"proj-mig-4", `\var\www`},
		{"proj-mig-5", `\var\www\`},
	}
	for _, l := range legacyInserts {
		_, err = database.Exec(`INSERT INTO projects (id, name, server_id, deploy_dir) VALUES (?, ?, ?, ?)`,
			l.id, l.id, server.ID, l.dir)
		if err != nil {
			t.Fatalf("raw insert %s failed: %v", l.id, err)
		}
	}

	// Run migration 2 SQL directly from definition
	_, err = database.Exec(migrations[1].sql)
	if err != nil {
		t.Fatalf("migration 2 execution failed: %v", err)
	}

	for _, l := range legacyInserts {
		p, err := database.GetProjectByID(l.id)
		if err != nil {
			t.Fatalf("GetProjectByID(%s) failed: %v", l.id, err)
		}
		if p.DeployDir != "/var/www/depo" {
			t.Errorf("project %s with original dir %q has DeployDir = %q, expected /var/www/depo", l.id, l.dir, p.DeployDir)
		}
	}
}

