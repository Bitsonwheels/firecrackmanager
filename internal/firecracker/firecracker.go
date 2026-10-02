// Package firecracker 解析 Firecracker 与 Jailer 可执行文件的路径。
package firecracker

import (
	"os"
	"os/exec"
	"path/filepath"
)

// 旧版安装脚本把二进制放在这里，作为兜底返回值。
const (
	DefaultBinary = "/usr/sbin/firecracker"
	DefaultJailer = "/usr/sbin/jailer"
)

// 发行版/官方 tar 包常见的安装目录。sudo 的 secure_path 往往不含 /usr/local/bin，
// 只查 PATH 会漏掉它，所以这里再显式探一遍。
var searchDirs = []string{
	"/usr/local/sbin",
	"/usr/local/bin",
	"/usr/sbin",
	"/usr/bin",
	"/sbin",
	"/bin",
}

// BinaryPath 返回 firecracker 可执行文件路径：优先 PATH，其次常见安装目录，最后兜底默认路径。
// 返回值不保证存在，调用方需自行 Stat 并显性报错。
func BinaryPath() string { return resolve("firecracker", DefaultBinary) }

// JailerPath 返回 jailer 可执行文件路径，规则同 BinaryPath。
func JailerPath() string { return resolve("jailer", DefaultJailer) }

func resolve(name, fallback string) string {
	if p, err := exec.LookPath(name); err == nil {
		return p
	}
	for _, dir := range searchDirs {
		p := filepath.Join(dir, name)
		if info, err := os.Stat(p); err == nil && !info.IsDir() && info.Mode()&0o111 != 0 {
			return p
		}
	}
	return fallback
}
