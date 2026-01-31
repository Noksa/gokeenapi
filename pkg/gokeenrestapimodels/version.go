package gokeenrestapimodels

// Version represents router firmware and hardware information from /rci/show/version endpoint.
type Version struct {
	// Release is the firmware release channel (e.g., "stable", "beta")
	Release string `json:"release,omitempty"`
	// Sandbox is the firmware sandbox identifier
	Sandbox string `json:"sandbox,omitempty"`
	// Title is the user-friendly firmware version string (e.g., "4.3.6.3")
	Title string `json:"title,omitempty"`
	// Arch is the CPU architecture (e.g., "mips", "aarch64")
	Arch string `json:"arch,omitempty"`
	// Ndm contains NDM (Network Device Manager) version info
	Ndm Ndm `json:"ndm"`
	// Bsp contains BSP (Board Support Package) version info
	Bsp Bsp `json:"bsp"`
	// Ndw contains NDW (Network Device Web) version info
	Ndw Ndw `json:"ndw"`
	// Ndw4 contains NDW4 web interface version info
	Ndw4 Ndw4 `json:"ndw4"`
	// Manufacturer is the device manufacturer name
	Manufacturer string `json:"manufacturer,omitempty"`
	// Vendor is the device vendor name
	Vendor string `json:"vendor,omitempty"`
	// Series is the product series (e.g., "KN")
	Series string `json:"series,omitempty"`
	// Model is the router model name (e.g., "Keenetic Giga")
	Model string `json:"model,omitempty"`
	// HwVersion is the hardware revision
	HwVersion string `json:"hw_version,omitempty"`
	// HwType is the hardware type identifier
	HwType string `json:"hw_type,omitempty"`
	// HwID is the unique hardware identifier
	HwID string `json:"hw_id,omitempty"`
	// Device is the device identifier
	Device string `json:"device,omitempty"`
	// Region is the device region code
	Region string `json:"region,omitempty"`
	// Description is the device description
	Description string `json:"description,omitempty"`
	// Prompt is the CLI prompt string
	Prompt string `json:"prompt,omitempty"`
}

// Ndm contains Network Device Manager version information.
type Ndm struct {
	// Exact is the exact NDM version string
	Exact string `json:"exact,omitempty"`
	// Cdate is the compilation date
	Cdate string `json:"cdate,omitempty"`
}

// Bsp contains Board Support Package version information.
type Bsp struct {
	// Exact is the exact BSP version string
	Exact string `json:"exact,omitempty"`
	// Cdate is the compilation date
	Cdate string `json:"cdate,omitempty"`
}

// Ndw contains Network Device Web interface version information.
type Ndw struct {
	// Features lists enabled web interface features
	Features string `json:"features,omitempty"`
	// Components lists installed web components
	Components string `json:"components,omitempty"`
}

// Ndw4 contains NDW4 web interface version information.
type Ndw4 struct {
	// Version is the NDW4 version string
	Version string `json:"version,omitempty"`
}
