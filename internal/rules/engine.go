package rules

import "sort"

const (
	ruleIDAddColumnDefault         = "ADD_COLUMN_DEFAULT"
	ruleIDIndexWithoutConcurrently = "INDEX_WITHOUT_CONCURRENTLY"
	ruleIDForeignKeyNoNotValid     = "FOREIGN_KEY_NO_NOT_VALID"
	ruleIDDropObject               = "DROP_OBJECT"
	ruleIDAlterColumnType          = "ALTER_COLUMN_TYPE"
	ruleIDNotNullNoDefault         = "NOT_NULL_NO_DEFAULT"
	ruleIDTruncateTable            = "TRUNCATE_TABLE"
	ruleIDMissingFKIndex           = "MISSING_FK_INDEX"
	ruleIDRenameObject             = "RENAME_OBJECT"
	ruleIDMissingLockTimeout       = "MISSING_LOCK_TIMEOUT"
)

// Engine runs registered rules against parsed SQL statements.
type Engine struct {
	rules []Rule
}

// NewEngine builds a rule engine with the provided rules.
func NewEngine(rules []Rule) *Engine {
	cloned := append([]Rule(nil), rules...)
	return &Engine{rules: cloned}
}

// RuleOverrideSource exposes rule override settings without coupling the rules package to config loading.
type RuleOverrideSource interface {
	RuleOverrides() map[string]string
}

// DefaultEngine registers all MVP rule IDs with placeholder implementations.
func DefaultEngine() *Engine {
	return NewEngine([]Rule{
		&AddColumnDefaultRule{},
		&IndexWithoutConcurrentlyRule{},
		&ForeignKeyNoNotValidRule{},
		&DropObjectRule{},
		&AlterColumnTypeRule{},
		&NotNullNoDefaultRule{},
		&TruncateTableRule{},
		&MissingFKIndexRule{},
		&RenameObjectRule{},
		&MissingLockTimeoutRule{},
	})
}

// DefaultEngineWithOverrides applies ignore/severity overrides to the default rule set.
func DefaultEngineWithOverrides(source RuleOverrideSource) *Engine {
	engine := DefaultEngine()
	if engine == nil || source == nil {
		return engine
	}

	overrides := source.RuleOverrides()
	if len(overrides) == 0 {
		return engine
	}

	configuredRules := make([]Rule, 0, len(engine.rules))
	for _, rule := range engine.rules {
		if rule == nil {
			continue
		}

		override := overrides[rule.ID()]
		if override == "ignore" {
			continue
		}

		configuredRules = append(configuredRules, withRuleOverride(rule, override))
	}

	return NewEngine(configuredRules)
}

// Run executes all registered rules against each statement and returns sorted findings.
func (e *Engine) Run(result ParseResult) []Finding {
	if e == nil || len(e.rules) == 0 || len(result.Statements) == 0 {
		return nil
	}

	findings := make([]Finding, 0)
	for _, stmt := range result.Statements {
		for _, rule := range e.rules {
			if rule == nil || isIgnored(stmt.IgnoredRules, rule.ID()) {
				continue
			}

			var finding *Finding
			if contextRule, ok := rule.(ContextRule); ok {
				finding = contextRule.CheckWithContext(stmt, result)
			} else {
				finding = rule.Check(stmt)
			}

			if finding == nil {
				continue
			}

			if finding.FilePath == "" {
				finding.FilePath = result.FilePath
			}

			findings = append(findings, *finding)
		}
	}

	for _, rule := range e.rules {
		postProcessRule, ok := rule.(PostProcessRule)
		if !ok {
			continue
		}

		finding := postProcessRule.CheckResult(result, findings)
		if finding == nil {
			continue
		}

		if finding.FilePath == "" {
			finding.FilePath = result.FilePath
		}

		findings = append(findings, *finding)
	}

	sort.SliceStable(findings, func(i, j int) bool {
		return findings[i].Line < findings[j].Line
	})

	return findings
}

func isIgnored(ignoredRules []string, ruleID string) bool {
	for _, ignoredRule := range ignoredRules {
		if ignoredRule == ruleID {
			return true
		}
	}

	return false
}

type configuredRule struct {
	Rule
	severityOverride *Severity
}

func withRuleOverride(rule Rule, override string) Rule {
	severity := severityForOverride(override)
	if severity == nil {
		return rule
	}

	return configuredRule{
		Rule:             rule,
		severityOverride: severity,
	}
}

func severityForOverride(override string) *Severity {
	switch override {
	case "error":
		severity := SeverityDangerous
		return &severity
	case "warning":
		severity := SeverityWarning
		return &severity
	case "info":
		severity := SeverityInfo
		return &severity
	default:
		return nil
	}
}

func (r configuredRule) Severity() Severity {
	if r.severityOverride != nil {
		return *r.severityOverride
	}
	return r.Rule.Severity()
}

func (r configuredRule) Check(stmt Statement) *Finding {
	return r.applySeverity(r.Rule.Check(stmt))
}

func (r configuredRule) CheckWithContext(stmt Statement, result ParseResult) *Finding {
	contextRule, ok := r.Rule.(ContextRule)
	if !ok {
		return r.Check(stmt)
	}
	return r.applySeverity(contextRule.CheckWithContext(stmt, result))
}

func (r configuredRule) CheckResult(result ParseResult, findings []Finding) *Finding {
	postProcessRule, ok := r.Rule.(PostProcessRule)
	if !ok {
		return nil
	}
	return r.applySeverity(postProcessRule.CheckResult(result, findings))
}

func (r configuredRule) applySeverity(finding *Finding) *Finding {
	if finding == nil || r.severityOverride == nil {
		return finding
	}

	cloned := *finding
	cloned.Severity = *r.severityOverride
	return &cloned
}
