package rmn

import "strings"

// splitNote separates an event line's trailing "//" note from its body. A
// note starts at the first "//" that is not inside a string, bracket, paren,
// or brace.
func splitNote(line string) (body, note string) {
	inStr := false
	depth := 0
	for i := 0; i+1 < len(line); i++ {
		c := line[i]
		switch {
		case inStr:
			if c == '\\' {
				i++
			} else if c == '"' {
				inStr = false
			}
		case c == '"':
			inStr = true
		case c == '(' || c == '[' || c == '{':
			depth++
		case c == ')' || c == ']' || c == '}':
			depth--
		case c == '/' && line[i+1] == '/' && depth == 0:
			return strings.TrimSpace(line[:i]), strings.TrimSpace(line[i+2:])
		}
	}
	return strings.TrimSpace(line), ""
}

// tokenize splits a string on whitespace at bracket/paren/brace depth zero.
// Bracket groups, parenthesised unit groups, and outcome objects are kept
// intact; string literals are respected.
func tokenize(s string) []string {
	var toks []string
	var b strings.Builder
	depth := 0
	inStr := false
	flush := func() {
		if b.Len() > 0 {
			toks = append(toks, b.String())
			b.Reset()
		}
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case inStr:
			b.WriteByte(c)
			if c == '\\' && i+1 < len(s) {
				i++
				b.WriteByte(s[i])
			} else if c == '"' {
				inStr = false
			}
		case c == '"':
			inStr = true
			b.WriteByte(c)
		case c == '(' || c == '[' || c == '{':
			depth++
			b.WriteByte(c)
		case c == ')' || c == ']' || c == '}':
			if depth > 0 {
				depth--
			}
			b.WriteByte(c)
		case (c == ' ' || c == '\t') && depth == 0:
			flush()
		default:
			b.WriteByte(c)
		}
	}
	flush()
	return toks
}

// splitTop splits s on sep at bracket/paren/brace depth zero, respecting
// string literals.
func splitTop(s string, sep byte) []string {
	var out []string
	depth := 0
	inStr := false
	start := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case inStr:
			if c == '\\' {
				i++
			} else if c == '"' {
				inStr = false
			}
		case c == '"':
			inStr = true
		case c == '(' || c == '[' || c == '{':
			depth++
		case c == ')' || c == ']' || c == '}':
			depth--
		case c == sep && depth == 0:
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	out = append(out, s[start:])
	return out
}

// stripOuter removes one matching pair of outer delimiters, if present.
func stripOuter(s string, open, close byte) (string, bool) {
	if len(s) >= 2 && s[0] == open && s[len(s)-1] == close {
		return s[1 : len(s)-1], true
	}
	return s, false
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }
func isAlpha(c byte) bool { return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') }
func isIdentByte(c byte) bool {
	return isAlpha(c) || isDigit(c) || c == '_' || c == '-'
}

func isIdent(s string) bool {
	if s == "" || !isAlpha(s[0]) {
		return false
	}
	for i := 1; i < len(s); i++ {
		if !isIdentByte(s[i]) {
			return false
		}
	}
	return true
}

// isUpperIdent reports whether s is an identifier whose first letter is upper
// case (used for faction instance IDs and zone names).
func isUpperIdent(s string) bool {
	return isIdent(s) && s[0] >= 'A' && s[0] <= 'Z'
}
