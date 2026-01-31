// Package gokeenrestapimodels provides data structures for Keenetic router REST API responses.
package gokeenrestapimodels

// ParseRequest represents a CLI command to be executed on the router via RCI interface.
type ParseRequest struct {
	// Parse contains the CLI command string (e.g., "ip route 10.0.0.0 255.0.0.0 Wireguard0 auto")
	Parse string `json:"parse"`
}

// ParseResponse wraps the result of a CLI command execution.
type ParseResponse struct {
	Parse Parse `json:"parse"`
}

// Status represents the execution status of a single CLI command.
type Status struct {
	// Status indicates success or error (e.g., "error", "ok")
	Status string `json:"status"`
	// Code is the error code if Status is "error"
	Code string `json:"code"`
	// Ident identifies the command or parameter that caused the error
	Ident string `json:"ident"`
	// Message provides human-readable description of the result
	Message string `json:"message"`
}

// Parse contains the parsed result of a CLI command execution.
type Parse struct {
	// Prompt is the CLI prompt returned after command execution
	Prompt string `json:"prompt"`
	// DynamicData holds the raw response body for additional parsing (not serialized to JSON)
	DynamicData string `json:"-"`
	// Status contains execution status for each command in the batch
	Status []Status `json:"status"`
}
