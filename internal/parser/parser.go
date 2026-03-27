package parser

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	pg_query "github.com/pganalyze/pg_query_go/v5"

	"github.com/emartai/locksmith/internal/rules"
)

type ParseResult = rules.ParseResult
type Statement = rules.Statement

// Parser reads SQL migration files and turns them into statement metadata.
type Parser struct{}

// ParseFile reads and parses a SQL file from disk.
func ParseFile(path string) (*ParseResult, error) {
	return (&Parser{}).ParseFile(path)
}

// ParseFile reads and parses a SQL file from disk.
func (p *Parser) ParseFile(path string) (*ParseResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("read %s: file not found", path)
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	return p.parseSQL(string(data), path)
}

func (p *Parser) parseSQL(rawSQL, path string) (*ParseResult, error) {
	result := &ParseResult{
		RawSQL:   rawSQL,
		FilePath: path,
	}

	if strings.TrimSpace(rawSQL) == "" {
		return result, nil
	}

	statements := splitStatements(rawSQL)
	result.Statements = make([]Statement, 0, len(statements))

	for _, part := range statements {
		raw := strings.TrimSpace(part.sql)
		if raw == "" {
			continue
		}

		stmt := Statement{
			Raw:          raw,
			Line:         part.line,
			IgnoredRules: ignoredRulesForLine(rawSQL, part.line),
		}

		tree, err := pg_query.Parse(raw)
		if err == nil && tree != nil && len(tree.Stmts) > 0 {
			stmt.Node = tree.Stmts[0].Stmt
		} else if err != nil {
			stmt.ParseError = formatParseError(part.line, raw, err)
		}

		result.Statements = append(result.Statements, stmt)
	}

	result.FilePath = filepath.Clean(result.FilePath)
	return result, nil
}

type statementPart struct {
	sql  string
	line int
}

func splitStatements(raw string) []statementPart {
	var (
		parts            []statementPart
		builder          strings.Builder
		line             = 1
		statementLine    = 1
		statementStarted bool
	)

	startStatement := func() {
		if !statementStarted {
			statementLine = line
			statementStarted = true
		}
	}

	flush := func() {
		if builder.Len() == 0 {
			statementStarted = false
			return
		}

		parts = append(parts, statementPart{
			sql:  builder.String(),
			line: statementLine,
		})
		builder.Reset()
		statementStarted = false
	}

	for i := 0; i < len(raw); {
		ch := raw[i]

		if !statementStarted {
			if ch == ' ' || ch == '\t' || ch == '\r' || ch == '\n' {
				if ch == '\n' {
					line++
				}
				i++
				continue
			}

			if ch == '-' && i+1 < len(raw) && raw[i+1] == '-' {
				next, lines := consumeLineComment(raw, i)
				line += lines
				i = next
				continue
			}

			if ch == '/' && i+1 < len(raw) && raw[i+1] == '*' {
				next, lines := consumeBlockComment(raw, i)
				line += lines
				i = next
				continue
			}
		}

		startStatement()

		switch ch {
		case '\'':
			next, lines := consumeQuoted(raw, i, '\'')
			builder.WriteString(raw[i:next])
			line += lines
			i = next
		case '"':
			next, lines := consumeQuoted(raw, i, '"')
			builder.WriteString(raw[i:next])
			line += lines
			i = next
		case '-':
			if i+1 < len(raw) && raw[i+1] == '-' {
				next, lines := consumeLineComment(raw, i)
				builder.WriteString(raw[i:next])
				line += lines
				i = next
				continue
			}
			builder.WriteByte(ch)
			i++
		case '/':
			if i+1 < len(raw) && raw[i+1] == '*' {
				next, lines := consumeBlockComment(raw, i)
				builder.WriteString(raw[i:next])
				line += lines
				i = next
				continue
			}
			builder.WriteByte(ch)
			i++
		case '$':
			tag, next, ok := consumeDollarQuote(raw, i)
			if ok {
				builder.WriteString(tag)
				i = next
				lines := strings.Count(tag, "\n")
				line += lines
				bodyEnd, bodyLines := consumeUntil(raw, i, tag)
				builder.WriteString(raw[i:bodyEnd])
				line += bodyLines
				i = bodyEnd
				continue
			}
			builder.WriteByte(ch)
			i++
		case ';':
			builder.WriteByte(ch)
			i++
			flush()
		default:
			builder.WriteByte(ch)
			if ch == '\n' {
				line++
			}
			i++
		}
	}

	flush()
	return parts
}

func consumeQuoted(raw string, start int, quote byte) (int, int) {
	lines := 0
	i := start + 1
	for i < len(raw) {
		if raw[i] == '\n' {
			lines++
		}
		if raw[i] == quote {
			if i+1 < len(raw) && raw[i+1] == quote {
				i += 2
				continue
			}
			return i + 1, lines
		}
		i++
	}
	return len(raw), lines
}

func consumeLineComment(raw string, start int) (int, int) {
	i := start
	lines := 0
	for i < len(raw) {
		if raw[i] == '\n' {
			lines++
			return i + 1, lines
		}
		i++
	}
	return len(raw), lines
}

func consumeBlockComment(raw string, start int) (int, int) {
	i := start + 2
	lines := 0
	for i < len(raw) {
		if raw[i] == '\n' {
			lines++
		}
		if i+1 < len(raw) && raw[i] == '*' && raw[i+1] == '/' {
			return i + 2, lines
		}
		i++
	}
	return len(raw), lines
}

func consumeDollarQuote(raw string, start int) (string, int, bool) {
	i := start + 1
	for i < len(raw) {
		if raw[i] == '$' {
			tag := raw[start : i+1]
			return tag, i + 1, true
		}
		if raw[i] == '\n' || raw[i] == '\r' || raw[i] == ' ' || raw[i] == '\t' {
			return "", start, false
		}
		i++
	}
	return "", start, false
}

func consumeUntil(raw string, start int, marker string) (int, int) {
	lines := 0
	i := start
	for i < len(raw) {
		if strings.HasPrefix(raw[i:], marker) {
			lines += strings.Count(marker, "\n")
			return i + len(marker), lines
		}
		if raw[i] == '\n' {
			lines++
		}
		i++
	}
	return len(raw), lines
}

func ignoredRulesForLine(rawSQL string, statementLine int) []string {
	if statementLine <= 1 {
		return nil
	}

	lines := strings.Split(rawSQL, "\n")
	if statementLine-2 >= len(lines) {
		return nil
	}

	comment := strings.TrimSpace(lines[statementLine-2])
	const prefix = "-- locksmith:ignore "
	if !strings.HasPrefix(comment, prefix) {
		return nil
	}

	rawRules := strings.Split(strings.TrimSpace(strings.TrimPrefix(comment, prefix)), ",")
	rulesList := make([]string, 0, len(rawRules))
	for _, ruleID := range rawRules {
		ruleID = strings.TrimSpace(ruleID)
		if ruleID != "" {
			rulesList = append(rulesList, ruleID)
		}
	}

	return rulesList
}

func formatParseError(line int, raw string, err error) string {
	snippet := strings.TrimSpace(raw)
	if len(snippet) > 80 {
		snippet = snippet[:80]
	}
	return fmt.Sprintf("line %d: %s: %v", line, snippet, err)
}
