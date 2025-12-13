/**
 * Desktop monitor type definitions
 */

package desktopmonitor

// SessionInfo represents session information
type SessionInfo struct {
	SessionID string
	User      string
	Seat      string
	Type      string // 'wayland' | 'x11'
	State     string
	Active    bool
	Idle      bool
	Locked    bool
	VT        int
	Display   string
}

// CompositorInfo represents compositor information
type CompositorInfo struct {
	Name      string
	Version   string
	Available bool
	Branch    string
	Commit    string
	BuildDate string
}

// WorkspaceInfo represents workspace information
type WorkspaceInfo struct {
	ID              int
	Name            string
	Monitor         string
	MonitorID       int
	Windows         int
	HasFullscreen   bool
	LastWindow      string
	LastWindowTitle string
}

// WindowInfo represents window information
type WindowInfo struct {
	Address   string
	Title     string
	Class     string
	Workspace struct {
		ID   int
		Name string
	}
	Monitor    int
	PID        int
	Floating   bool
	Fullscreen interface{} // bool or number
	Mapped     bool
	Hidden     bool
	Pinned     bool
}

// MonitorInfo represents monitor information
type MonitorInfo struct {
	ID              int
	Name            string
	Description     string
	Make            string
	Model           string
	Serial          string
	Width           int
	Height          int
	RefreshRate     float64
	X               int
	Y               int
	ActiveWorkspace struct {
		ID   int
		Name string
	}
	Scale      float64
	Transform  int
	VRR        bool
	DPMSStatus bool
}

// DesktopStatus represents complete desktop status
type DesktopStatus struct {
	Session    SessionInfo
	Compositor CompositorInfo
	Workspaces []WorkspaceInfo
	Windows    []WindowInfo
	Monitors   []MonitorInfo
}

// CompositorType represents compositor types
type CompositorType string

const (
	CompositorTypeHyprland CompositorType = "hyprland"
	CompositorTypeSway     CompositorType = "sway"
	CompositorTypeNiri     CompositorType = "niri"
	CompositorTypeI3       CompositorType = "i3"
	CompositorTypeUnknown  CompositorType = "unknown"
)
