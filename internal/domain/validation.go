package domain

import "sort"

// Severity ranks a validation finding.
type Severity int

const (
	// SeverityInfo is advisory (e.g. an item declares no mod ID).
	SeverityInfo Severity = iota
	// SeverityWarning is a likely problem that does not block (e.g. an unknown
	// mod ID, or a possible load-order issue).
	SeverityWarning
	// SeverityError is a blocking problem (e.g. a missing required dependency).
	SeverityError
)

func (s Severity) String() string {
	switch s {
	case SeverityError:
		return "ERROR"
	case SeverityWarning:
		return "WARNING"
	default:
		return "INFO"
	}
}

// Finding codes.
const (
	CodeMissingDependency = "missing-dependency"
	CodeDelisted          = "delisted"
	CodeBanned            = "banned"
	CodeUnknownModID      = "unknown-mod-id"
	CodeUnusedModID       = "unused-mod-id"
	CodeUnusedMap         = "unused-map"
	CodeNoModID           = "no-mod-id"
	CodeLoadOrder         = "load-order"
	CodeBuildCompat       = "build-compat"
	CodeModIDClash        = "mod-id-clash"
)

// Finding is one validation result.
type Finding struct {
	Severity Severity
	Code     string
	Message  string
	Subject  string // the mod ID or workshop ID the finding concerns
}

// Report is an ordered set of findings.
type Report struct {
	Findings []Finding
}

// Add appends a finding.
func (r *Report) Add(f Finding) { r.Findings = append(r.Findings, f) }

// Addf appends a finding with a formatted message-free shorthand.
func (r *Report) AddFinding(sev Severity, code, subject, msg string) {
	r.Findings = append(r.Findings, Finding{Severity: sev, Code: code, Subject: subject, Message: msg})
}

// HasErrors reports whether any finding is an error.
func (r Report) HasErrors() bool {
	for _, f := range r.Findings {
		if f.Severity == SeverityError {
			return true
		}
	}
	return false
}

// Count returns the number of findings at the given severity.
func (r Report) Count(sev Severity) int {
	n := 0
	for _, f := range r.Findings {
		if f.Severity == sev {
			n++
		}
	}
	return n
}

// Sorted returns findings ordered by descending severity, then code, then
// subject - stable and presentation-friendly.
func (r Report) Sorted() []Finding {
	out := append([]Finding(nil), r.Findings...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Severity != out[j].Severity {
			return out[i].Severity > out[j].Severity
		}
		if out[i].Code != out[j].Code {
			return out[i].Code < out[j].Code
		}
		return out[i].Subject < out[j].Subject
	})
	return out
}
