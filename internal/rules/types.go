package rules

// Severity describes the impact level of a finding.
type Severity string

const (
	SeverityDangerous Severity = "DANGEROUS"
	SeverityWarning   Severity = "WARNING"
	SeverityInfo      Severity = "INFO"
)

// Finding captures a rule violation for a single SQL statement.
type Finding struct {
	RuleID   string
	Severity Severity
	Line     int
	FilePath string
	Summary  string
	Why      string
	LockType string
	Fix      string
}

// Rule checks a statement and returns a finding when it matches.
type Rule interface {
	ID() string
	Severity() Severity
	Check(stmt Statement) *Finding
}

// ContextRule checks a statement with access to the full parse result.
type ContextRule interface {
	Rule
	CheckWithContext(stmt Statement, result ParseResult) *Finding
}

// PostProcessRule runs once after per-statement checks have completed.
type PostProcessRule interface {
	Rule
	CheckResult(result ParseResult, findings []Finding) *Finding
}

// Statement represents a parsed SQL statement and its metadata.
type Statement struct {
	Raw          string
	Line         int
	Node         interface{}
	ParseError   string
	IgnoredRules []string
}

// ParseResult contains the parser output for a file.
type ParseResult struct {
	Statements []Statement
	RawSQL     string
	FilePath   string
}
