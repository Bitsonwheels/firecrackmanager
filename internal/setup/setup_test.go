package setup

import (
	"strings"
	"testing"
)

// systemd 拒绝相对 ExecStart 路径（bad-setting），日志写 journal，
// 沙箱白名单必须覆盖 data/log/config/builder 四个可写目录。
func TestServiceFileContent(t *testing.T) {
	content := serviceFileContent("/usr/local/bin/firecrackmanager")

	if !strings.Contains(content, "ExecStart=/usr/local/bin/firecrackmanager -config "+DefaultConfigPath) {
		t.Errorf("ExecStart must be absolute and carry -config, got:\n%s", content)
	}

	for _, want := range []string{
		"StandardOutput=journal",
		"StandardError=journal",
		"ProtectSystem=strict",
		"ProtectHome=read-only",
		"PrivateTmp=true",
		"ReadWritePaths=" + DefaultDataDir + " " + DefaultLogDir + " /etc/firecrackmanager /run " + DefaultBuilderDir + " " + DefaultJailerDir,
	} {
		if !strings.Contains(content, want) {
			t.Errorf("unit should contain %q, got:\n%s", want, content)
		}
	}
}
