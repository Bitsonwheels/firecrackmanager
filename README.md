# FireCrackManager

A MicroVM management daemon for [Firecracker](https://firecracker-microvm.github.io/). 
FireCrackManager is a part of Artica appliance ecosystem.

It provides a REST API and web-based UI for managing virtual machines, networks, kernel images, and root filesystems.

<img src="http://www.articatech.com/images/2025-12-14_04-03-19.png">

## Features

### Virtual Machine Management
- Create, start, stop, and delete MicroVMs
- Configure vCPU, memory, kernel arguments, and DNS servers
- Real-time VM status monitoring with reachability checks
- Serial console access via WebSocket
- **Autorun**: Automatically start designated VMs when FireCrackManager starts

### Snapshots
- Create full and differential snapshots of running VMs
- List, restore, and delete snapshots
- Preserve VM state for quick recovery

### Disk Management
- Attach additional virtual disks to VMs
- Automatic ext4 filesystem formatting
- Automatic fstab configuration for persistent mounts
- Support for multiple disks per VM

### VM Import/Export
- Export VMs as `.fcrack` archives (virtual appliance format)
- Import `.fcrack` files to create new VMs
- Duplicate existing VMs with all configurations

### Network Management
- Create isolated virtual networks with custom subnets
- Automatic TAP device and bridge creation
- NAT support for internet connectivity
- IP allocation and MAC address generation

### Kernel & RootFS Management
- Download kernel images from URLs
- Download or create root filesystem images
- Upload custom images via web interface
- Set default kernel for new VMs

### User & Group Management
- Multi-user support with role-based access (admin/user)
- **Privilege Groups**: Assign users to groups with specific permissions
- Group-level VM access control (start, stop, console, edit, snapshot, disk)
- Session-based authentication

## Changelog

### 2026-10-02
- Resolve the Firecracker/Jailer binary path at runtime instead of hardcoding
  `/usr/sbin/firecracker`. It now prefers `PATH`, then the common install
  directories (`/usr/local/bin`, …), and only falls back to `/usr/sbin`.
  Installations outside `/usr/sbin` no longer need a manual symlink.
- Fix the systemd unit written by `firecrackmanager -setup`: it no longer
  references the binary by the relative path `./firecrackmanager`, which
  systemd rejected with `bad-setting` ("Neither a valid executable name nor
  an absolute path"). The absolute path of the running binary is used as
  fallback.
- Align the generated unit with the deployed one: hardening
  (`ProtectSystem=strict`, `ProtectHome=read-only`, `PrivateTmp=true`) is
  now enabled and `/home/Builder` is created up front, since a missing
  `ReadWritePaths` entry makes the unit fail to start. `StandardOutput` /
  `StandardError` now go to the journal instead of appending to the log
  file, which already received every line twice via the daemon's own
  `MultiWriter`.
- Add `/etc/firecrackmanager` to `ReadWritePaths` in the unit files so that
  saving the proxy configuration keeps working under `ProtectSystem=strict`.
- Grant the jailer its chroot directory (`/srv/jailer`) in the systemd
  sandbox. Enabling the jailer previously failed because `ProtectSystem=strict`
  made the jail directory read-only; the directory is now created up front.
