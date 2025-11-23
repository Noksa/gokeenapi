package gokeenrestapimodels

type Version struct {
	Release      string `json:"release,omitempty"`
	Sandbox      string `json:"sandbox,omitempty"`
	Title        string `json:"title,omitempty"`
	Arch         string `json:"arch,omitempty"`
	Ndm          Ndm    `json:"ndm"`
	Bsp          Bsp    `json:"bsp"`
	Ndw          Ndw    `json:"ndw"`
	Ndw4         Ndw4   `json:"ndw4"`
	Manufacturer string `json:"manufacturer,omitempty"`
	Vendor       string `json:"vendor,omitempty"`
	Series       string `json:"series,omitempty"`
	Model        string `json:"model,omitempty"`
	HwVersion    string `json:"hw_version,omitempty"`
	HwType       string `json:"hw_type,omitempty"`
	HwID         string `json:"hw_id,omitempty"`
	Device       string `json:"device,omitempty"`
	Region       string `json:"region,omitempty"`
	Description  string `json:"description,omitempty"`
	Prompt       string `json:"prompt,omitempty"`
}
type Ndm struct {
	Exact string `json:"exact,omitempty"`
	Cdate string `json:"cdate,omitempty"`
}
type Bsp struct {
	Exact string `json:"exact,omitempty"`
	Cdate string `json:"cdate,omitempty"`
}
type Ndw struct {
	Features   string `json:"features,omitempty"`
	Components string `json:"components,omitempty"`
}
type Ndw4 struct {
	Version string `json:"version,omitempty"`
}
