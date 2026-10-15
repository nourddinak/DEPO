package crypto

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// GenerateRandomHex generates a secure random hex string with the given byte length.
func GenerateRandomHex(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("rand.Read: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// GenerateSessionToken creates a prefixed session token.
func GenerateSessionToken() (raw string, hash string, err error) {
	randStr, err := GenerateRandomHex(32)
	if err != nil {
		return "", "", err
	}
	raw = "depo_sess_" + randStr
	hash = SHA256String(raw)
	return raw, hash, nil
}

// GenerateEnrollmentToken creates a prefixed single-use server enrollment token.
func GenerateEnrollmentToken() (raw string, hash string, err error) {
	randStr, err := GenerateRandomHex(24)
	if err != nil {
		return "", "", err
	}
	raw = "depo_enr_" + randStr
	hash = SHA256String(raw)
	return raw, hash, nil
}

// GenerateAgentAuthToken creates a persistent agent authentication token.
func GenerateAgentAuthToken() (raw string, hash string, err error) {
	randStr, err := GenerateRandomHex(32)
	if err != nil {
		return "", "", err
	}
	raw = "depo_agt_" + randStr
	hash = SHA256String(raw)
	return raw, hash, nil
}

// GenerateDeployToken creates a project-scoped deployment token.
func GenerateDeployToken() (raw string, hash string, err error) {
	randStr, err := GenerateRandomHex(32)
	if err != nil {
		return "", "", err
	}
	raw = "depo_dpl_" + randStr
	hash = SHA256String(raw)
	return raw, hash, nil
}

// GenerateInvitationToken creates an invitation token for team onboarding.
func GenerateInvitationToken() (raw string, hash string, err error) {
	randStr, err := GenerateRandomHex(24)
	if err != nil {
		return "", "", err
	}
	raw = "depo_inv_" + randStr
	hash = SHA256String(raw)
	return raw, hash, nil
}

// Short safely truncates a string to at most maxLen without out-of-bounds panic.
func Short(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

