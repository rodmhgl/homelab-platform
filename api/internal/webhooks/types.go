package webhooks

import "time"

// FalcoWebhookPayload represents the JSON payload sent by Falcosidekick
// Documentation: https://github.com/falcosecurity/falcosidekick#webhook
type FalcoWebhookPayload struct {
	// UUID is a unique identifier for this event
	UUID string `json:"uuid"`

	// Output is the human-readable alert message formatted by the Falco rule
	Output string `json:"output"`

	// Priority is the severity level (Debug, Informational, Notice, Warning, Error, Critical, Alert, Emergency)
	Priority string `json:"priority"`

	// Rule is the name of the Falco rule that triggered this event
	Rule string `json:"rule"`

	// Time is the timestamp when the event occurred
	Time time.Time `json:"time"`

	// OutputFields contains enrichment data extracted by the Falco rule
	// Common keys:
	//   - k8s.ns.name (namespace)
	//   - k8s.pod.name (pod)
	//   - container.name
	//   - container.image.repository
	//   - proc.name (process)
	//   - proc.cmdline (full command line)
	//   - user.name
	//   - user.uid
	//   - fd.name (file descriptor / file path)
	//   - fd.sport (source port for network events)
	//   - fd.rip (remote IP)
	//   - fd.rport (remote port)
	OutputFields map[string]interface{} `json:"output_fields"`

	// Source indicates which Falco plugin generated this event (syscall, k8s_audit, etc.)
	Source string `json:"source"`

	// Tags are labels attached to the rule for categorization
	Tags []string `json:"tags,omitempty"`

	// Hostname is the node where the event occurred
	Hostname string `json:"hostname,omitempty"`
}
