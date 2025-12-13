package installer

import (
	"fmt"
	"os"
	"strings"
)

// Distro represents a Linux distribution
type Distro int

const (
	Unknown Distro = iota
	Arch
	Fedora
	Debian
	Ubuntu
)

func (d Distro) String() string {
	switch d {
	case Arch:
		return "Arch Linux"
	case Fedora:
		return "Fedora"
	case Debian:
		return "Debian"
	case Ubuntu:
		return "Ubuntu"
	default:
		return "Unknown"
	}
}

// DetectDistro detects the current Linux distribution
func DetectDistro() (Distro, error) {
	// Read /etc/os-release
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return Unknown, fmt.Errorf("failed to read /etc/os-release: %w", err)
	}

	content := string(data)
	lines := strings.Split(content, "\n")

	var id, idLike string
	for _, line := range lines {
		if strings.HasPrefix(line, "ID=") {
			id = strings.Trim(strings.TrimPrefix(line, "ID="), "\"")
		}
		if strings.HasPrefix(line, "ID_LIKE=") {
			idLike = strings.Trim(strings.TrimPrefix(line, "ID_LIKE="), "\"")
		}
	}

	// Check ID first
	switch id {
	case "arch", "cachyos", "endeavouros", "manjaro":
		return Arch, nil
	case "fedora":
		return Fedora, nil
	case "debian":
		return Debian, nil
	case "ubuntu":
		return Ubuntu, nil
	}

	// Check ID_LIKE
	if strings.Contains(idLike, "arch") {
		return Arch, nil
	}
	if strings.Contains(idLike, "fedora") {
		return Fedora, nil
	}
	if strings.Contains(idLike, "debian") {
		return Debian, nil
	}

	return Unknown, fmt.Errorf("unsupported distribution: %s", id)
}

// IsSupported checks if the distro is supported
func IsSupported(distro Distro) bool {
	switch distro {
	case Arch:
		return true
	case Fedora, Debian, Ubuntu:
		return false // Not implemented yet
	default:
		return false
	}
}
