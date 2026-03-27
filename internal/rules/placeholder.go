package rules

// NoOpRule is a placeholder rule used until real implementations are added.
type NoOpRule struct {
	id       string
	severity Severity
}

// NewNoOpRule constructs a placeholder rule with the provided metadata.
func NewNoOpRule(id string, severity Severity) *NoOpRule {
	return &NoOpRule{id: id, severity: severity}
}

// ID returns the placeholder rule ID.
func (r *NoOpRule) ID() string {
	return r.id
}

// Severity returns the placeholder severity.
func (r *NoOpRule) Severity() Severity {
	return r.severity
}

// Check never reports a finding.
func (r *NoOpRule) Check(stmt Statement) *Finding {
	return nil
}
