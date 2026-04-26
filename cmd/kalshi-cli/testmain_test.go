package main_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var (
	testBinary string
	testKeyID  = "test-key-00000000-0000-0000-0000-000000000000"
	testKeyPEM string
)

func TestMain(m *testing.M) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		fmt.Fprintf(os.Stderr, "generate test key: %v\n", err)
		os.Exit(1)
	}
	keyBytes, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal test key: %v\n", err)
		os.Exit(1)
	}
	testKeyPEM = string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyBytes}))

	tmp, err := os.MkdirTemp("", "kalshi-cli-test-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "mkdirtemp: %v\n", err)
		os.Exit(1)
	}
	testBinary = filepath.Join(tmp, "kalshi-cli")
	out, err := exec.Command("go", "build", "-o", testBinary, ".").CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "build failed: %v\n%s\n", err, out)
		os.RemoveAll(tmp)
		os.Exit(1)
	}

	code := m.Run()
	os.RemoveAll(tmp)
	os.Exit(code)
}

// runCLI executes the compiled test binary with the given args, injecting mock
// server URL and test credentials via environment variables.
// Returns stdout, stderr, and the exit code.
func runCLI(t *testing.T, serverURL string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmd := exec.Command(testBinary, args...)
	cmd.Env = append(os.Environ(),
		"KALSHI_KEY_ID="+testKeyID,
		"KALSHI_PRIVATE_KEY="+testKeyPEM,
		"KALSHI_BASE_URL="+serverURL+"/trade-api/v2",
	)
	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	}
	return outBuf.String(), errBuf.String(), exitCode
}
