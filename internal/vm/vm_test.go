package vm

import (
	"os"
	"path/filepath"
	"strings"

	"firecrackmanager/internal/database"
	"testing"
)

func TestWriteRootFSDNSConfigReplacesResolvedStubSymlink(t *testing.T) {
	mountPoint := t.TempDir()

	etcDir := filepath.Join(mountPoint, "etc")
	runResolvedDir := filepath.Join(mountPoint, "run", "systemd", "resolve")
	systemdDir := filepath.Join(mountPoint, "usr", "lib", "systemd")
	for _, dir := range []string{etcDir, runResolvedDir, systemdDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("MkdirAll(%s) error = %v", dir, err)
		}
	}
	if err := os.WriteFile(filepath.Join(systemdDir, "systemd"), []byte("systemd"), 0755); err != nil {
		t.Fatalf("WriteFile(systemd) error = %v", err)
	}
	if err := os.Symlink("../run/systemd/resolve/stub-resolv.conf", filepath.Join(etcDir, "resolv.conf")); err != nil {
		t.Fatalf("Symlink(resolv.conf) error = %v", err)
	}

	if err := writeRootFSDNSConfig(mountPoint, "8.8.8.8, 1.1.1.1", nil); err != nil {
		t.Fatalf("writeRootFSDNSConfig() error = %v", err)
	}

	info, err := os.Lstat(filepath.Join(etcDir, "resolv.conf"))
	if err != nil {
		t.Fatalf("Lstat(resolv.conf) error = %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("resolv.conf should be a regular file, got symlink")
	}

	resolvContent, err := os.ReadFile(filepath.Join(etcDir, "resolv.conf"))
	if err != nil {
		t.Fatalf("ReadFile(resolv.conf) error = %v", err)
	}
	if !strings.Contains(string(resolvContent), "nameserver 8.8.8.8") || !strings.Contains(string(resolvContent), "nameserver 1.1.1.1") {
		t.Fatalf("resolv.conf does not contain expected nameservers: %s", resolvContent)
	}

	resolvedContent, err := os.ReadFile(filepath.Join(etcDir, "systemd", "resolved.conf.d", "99-firecrackmanager-dns.conf"))
	if err != nil {
		t.Fatalf("ReadFile(resolved drop-in) error = %v", err)
	}
	if got, want := string(resolvedContent), "DNS=8.8.8.8 1.1.1.1"; !strings.Contains(got, want) {
		t.Fatalf("resolved drop-in = %q, want it to contain %q", got, want)
	}
}

func TestWriteRootFSDNSConfigRejectsInvalidServers(t *testing.T) {
	if err := writeRootFSDNSConfig(t.TempDir(), "not-an-ip", nil); err == nil {
		t.Fatalf("writeRootFSDNSConfig() should reject invalid DNS servers")
	}
}

// Regressionstest: der jailer darf firecracker nicht aus der Supervision loesen.
// --daemonize und --new-pid-ns lassen den jailer nach dem Fork sofort exiten,
// wodurch startVMInternal per cmd.Wait() den Tod des VM meldet, obwohl er noch
// bootet (Monitor raeumt dann Socket und Jail-Verzeichnis eines laufenden VM auf).
func TestBuildJailerArgsKeepsFirecrackerSupervised(t *testing.T) {
	m := &Manager{jailerConfig: &JailerConfig{
		Enabled:    true,
		ChrootBase: DefaultJailerChrootBase,
		UID:        1000,
		GID:        1000,
		CgroupVer:  2,
	}}

	args := m.buildJailerArgs(&database.VM{ID: "abc123"}, &jailInfo{socketPath: "/run/firecracker.socket"})
	joined := strings.Join(args, " ")

	for _, forbidden := range []string{"--daemonize", "--new-pid-ns"} {
		if strings.Contains(joined, forbidden) {
			t.Errorf("jailer args enthalten %s, firecracker waere nicht mehr als Kindprozess ueberwachbar: %s", forbidden, joined)
		}
	}
	// chroot-Isolation muss erhalten bleiben
	for _, required := range []string{"--chroot-base-dir " + DefaultJailerChrootBase, "--uid 1000", "--gid 1000"} {
		if !strings.Contains(joined, required) {
			t.Errorf("jailer args fehlen %q: %s", required, joined)
		}
	}
}
