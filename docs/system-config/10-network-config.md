# Network Configuration

## Network Interfaces

### Primary Network Interface
- **Interface**: enp114s0
- **IPv4 Address**: 192.168.50.135/24
- **Broadcast**: 192.168.50.255
- **IPv6 Address**: fe80::f0c1:23bf:6b7b:cbaf/64
- **Type**: Dynamic (DHCP)

### Loopback Interface
- **Interface**: lo
- **IPv4**: 127.0.0.1/8
- **IPv6**: ::1/128

### Docker Network
- **Interface**: docker0
- **IPv4**: 172.17.0.1/16
- **Broadcast**: 172.17.255.255
- **Purpose**: Docker bridge network

## Network Configuration

- **Network Manager**: Likely NetworkManager or systemd-networkd
- **DNS**: Configured via DHCP or systemd-resolved
- **Firewall**: Status unknown (check with `sudo firewall-cmd --state` or `sudo ufw status`)

## Docker

- **Docker Installed**: Yes (docker0 interface present)
- **User Groups**: User is in `docker` group
- **Network**: Default bridge network active

## Network Tools

To check network status:
```bash
# Check interface status
ip addr show

# Check routing
ip route show

# Check DNS
systemd-resolve --status  # or resolvectl status

# Check firewall
sudo firewall-cmd --state  # if using firewalld
sudo ufw status  # if using ufw
```

