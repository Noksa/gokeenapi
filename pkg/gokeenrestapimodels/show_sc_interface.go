package gokeenrestapimodels

type Address struct {
	Address string `json:"address"`
}
type IP struct {
	Address Address `json:"address"`
}
type Asc struct {
	Jc   string `json:"jc"`
	Jmin string `json:"jmin"`
	Jmax string `json:"jmax"`
	S1   string `json:"s1"`
	S2   string `json:"s2"`
	H1   string `json:"h1"`
	H2   string `json:"h2"`
	H3   string `json:"h3"`
	H4   string `json:"h4"`
}
type Endpoint struct {
	Address string `json:"address"`
}
type KeepaliveInterval struct {
	Interval int `json:"interval"`
}
type AllowIps struct {
	Address string `json:"address"`
	Mask    string `json:"mask"`
}
type Peer struct {
	Key               string            `json:"key"`
	Comment           string            `json:"comment,omitempty"`
	Endpoint          Endpoint          `json:"endpoint"`
	KeepaliveInterval KeepaliveInterval `json:"keepalive-interval"`
	PresharedKey      string            `json:"preshared-key"`
	AllowIps          []AllowIps        `json:"allow-ips"`
}
type Wireguard struct {
	Asc  Asc    `json:"asc"`
	Peer []Peer `json:"peer"`
}
type RciShowScInterface struct {
	Description string    `json:"description,omitempty"`
	IP          IP        `json:"ip"`
	Wireguard   Wireguard `json:"wireguard"`
}
