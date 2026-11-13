// Package permissions reports and requests the OS permissions Sayki needs.
package permissions

// Status values: "granted", "denied", "undetermined", "unsupported".
type Status struct {
	Microphone    string `json:"microphone"`
	Accessibility string `json:"accessibility"`
}
