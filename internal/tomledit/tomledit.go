// Package tomledit changes individual keys of one table in a TOML document
// and leaves every other byte alone: comments, ordering, quoting, blank lines
// and line endings are kept. Each edit is checked with a real TOML decoder:
// the result must decode to the original document with only the requested
// keys changed, otherwise the edit is refused.
package tomledit

import (
	"bytes"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

// Change sets Key to Value (a TOML literal such as `"4285F4"` or `2`), or
// removes Key when Delete is set.
type Change struct {
	Key    string
	Value  string
	Delete bool
}

// String returns a TOML basic string literal for s.
func String(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			if r < 0x20 || r == 0x7f {
				fmt.Fprintf(&b, `\u%04X`, r)
			} else {
				b.WriteRune(r)
			}
		}
	}
	b.WriteByte('"')
	return b.String()
}

// Get returns the literal text of key in [table].
func Get(src []byte, table, key string) (string, bool, error) {
	stmts, err := scan(src)
	if err != nil {
		return "", false, err
	}
	start, end, found := section(stmts, table)
	if !found {
		return "", false, nil
	}
	for _, s := range stmts[start:end] {
		if s.kind == stmtKeyValue && s.key == key {
			return string(src[s.valStart:s.valEnd]), true, nil
		}
	}
	return "", false, nil
}

// Edit applies changes to the keys of [table], creating the table at the end
// of the document when it does not exist.
func Edit(src []byte, table string, changes []Change) ([]byte, error) {
	var before map[string]any
	if _, err := toml.Decode(string(src), &before); err != nil {
		return nil, fmt.Errorf("existing file is not valid TOML: %w", err)
	}
	stmts, err := scan(src)
	if err != nil {
		return nil, err
	}
	nl := "\n"
	if bytes.Contains(src, []byte("\r\n")) {
		nl = "\r\n"
	}

	type splice struct {
		start, end int
		text       string
	}
	var splices []splice
	var inserts []string

	start, end, found := section(stmts, table)
	if !found {
		for _, s := range stmts {
			if s.kind == stmtHeader {
				break
			}
			if s.kind == stmtKeyValue && (s.key == table || strings.HasPrefix(s.key, table+".")) {
				return nil, fmt.Errorf("[%s] is defined with dotted keys or an inline table; edit it by hand", table)
			}
		}
	}
	for _, c := range changes {
		idx := -1
		if found {
			for i := start + 1; i < end; i++ {
				if stmts[i].kind == stmtKeyValue && stmts[i].key == c.Key {
					idx = i
					break
				}
			}
		}
		switch {
		case idx >= 0 && c.Delete:
			s := stmts[idx]
			splices = append(splices, splice{s.lineStart, s.lineEnd, ""})
		case idx >= 0:
			s := stmts[idx]
			splices = append(splices, splice{s.valStart, s.valEnd, c.Value})
		case !c.Delete:
			inserts = append(inserts, bareOrQuotedKey(c.Key)+" = "+c.Value+nl)
		}
	}
	if len(inserts) > 0 {
		text := strings.Join(inserts, "")
		if found {
			at := stmts[start].lineEnd
			for i := start + 1; i < end; i++ {
				if stmts[i].kind == stmtKeyValue {
					at = stmts[i].lineEnd
				}
			}
			if at == len(src) && (len(src) == 0 || src[len(src)-1] != '\n') {
				text = nl + text
			}
			splices = append(splices, splice{at, at, text})
		} else {
			prefix := ""
			if len(src) > 0 && src[len(src)-1] != '\n' {
				prefix = nl
			}
			if len(bytes.TrimSpace(src)) > 0 {
				prefix += nl
			}
			splices = append(splices, splice{len(src), len(src), prefix + "[" + table + "]" + nl + text})
		}
	}

	sort.SliceStable(splices, func(i, j int) bool { return splices[i].start < splices[j].start })
	var out bytes.Buffer
	last := 0
	for _, s := range splices {
		if s.start < last {
			return nil, fmt.Errorf("overlapping edits in [%s]", table)
		}
		out.Write(src[last:s.start])
		out.WriteString(s.text)
		last = s.end
	}
	out.Write(src[last:])

	if err := verify(before, out.Bytes(), table, changes); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// verify decodes the edited document and compares it with the original plus
// the requested changes.
func verify(before map[string]any, out []byte, table string, changes []Change) error {
	var after map[string]any
	if _, err := toml.Decode(string(out), &after); err != nil {
		return fmt.Errorf("edit produced invalid TOML: %w", err)
	}
	want := before
	tbl, _ := want[table].(map[string]any)
	if tbl == nil {
		tbl = map[string]any{}
		want[table] = tbl
	}
	for _, c := range changes {
		if c.Delete {
			delete(tbl, c.Key)
			continue
		}
		var v map[string]any
		if _, err := toml.Decode("v = "+c.Value, &v); err != nil {
			return fmt.Errorf("value for %s is not a TOML literal: %w", c.Key, err)
		}
		tbl[c.Key] = v["v"]
	}
	if len(tbl) == 0 {
		if _, ok := after[table]; !ok {
			delete(want, table)
		}
	}
	if !reflect.DeepEqual(want, after) {
		return fmt.Errorf("refusing to write: editing [%s] would change other settings", table)
	}
	return nil
}

func bareOrQuotedKey(key string) string {
	for _, r := range key {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-') {
			return String(key)
		}
	}
	if key == "" {
		return `""`
	}
	return key
}

const (
	stmtKeyValue = iota + 1
	stmtHeader
)

type stmt struct {
	kind  int
	table string // header: normalized table name
	array bool   // header: [[array of tables]]
	key   string // key/value: normalized key
	// lineStart and lineEnd span the statement's lines, including the final
	// newline. valStart and valEnd span a value literal.
	lineStart, lineEnd int
	valStart, valEnd   int
}

// section returns the statement range of [table]: the header index and the
// index of the next header.
func section(stmts []stmt, table string) (int, int, bool) {
	for i, s := range stmts {
		if s.kind == stmtHeader && !s.array && s.table == table {
			end := len(stmts)
			for j := i + 1; j < len(stmts); j++ {
				if stmts[j].kind == stmtHeader {
					end = j
					break
				}
			}
			return i, end, true
		}
	}
	return 0, 0, false
}

// scan splits a TOML document into headers and key/value statements.
func scan(src []byte) ([]stmt, error) {
	var stmts []stmt
	p := &scanner{src: src}
	for p.pos < len(src) {
		lineStart := p.pos
		p.skipSpace()
		if p.eol() {
			p.nextLine()
			continue
		}
		switch src[p.pos] {
		case '#':
			p.nextLine()
			continue
		case '[':
			s := stmt{kind: stmtHeader, lineStart: lineStart}
			p.pos++
			if p.peek('[') {
				s.array = true
				p.pos++
			}
			nameStart := p.pos
			if err := p.scanKeyUntil(']'); err != nil {
				return nil, err
			}
			s.table = normalizeKey(string(src[nameStart:p.pos]))
			p.pos++
			if s.array {
				if !p.peek(']') {
					return nil, p.errorf("unterminated array table header")
				}
				p.pos++
			}
			if err := p.finishLine(); err != nil {
				return nil, err
			}
			s.lineEnd = p.pos
			stmts = append(stmts, s)
		default:
			s := stmt{kind: stmtKeyValue, lineStart: lineStart}
			keyStart := p.pos
			if err := p.scanKeyUntil('='); err != nil {
				return nil, err
			}
			s.key = normalizeKey(string(src[keyStart:p.pos]))
			p.pos++
			p.skipSpace()
			s.valStart = p.pos
			if err := p.scanValue(); err != nil {
				return nil, err
			}
			s.valEnd = p.pos
			if err := p.finishLine(); err != nil {
				return nil, err
			}
			s.lineEnd = p.pos
			stmts = append(stmts, s)
		}
	}
	return stmts, nil
}

type scanner struct {
	src []byte
	pos int
}

func (p *scanner) errorf(format string, args ...any) error {
	line := 1 + bytes.Count(p.src[:min(p.pos, len(p.src))], []byte("\n"))
	return fmt.Errorf("toml line %d: %s", line, fmt.Sprintf(format, args...))
}

func (p *scanner) peek(c byte) bool { return p.pos < len(p.src) && p.src[p.pos] == c }

func (p *scanner) hasPrefix(s string) bool { return bytes.HasPrefix(p.src[p.pos:], []byte(s)) }

func (p *scanner) skipSpace() {
	for p.pos < len(p.src) && (p.src[p.pos] == ' ' || p.src[p.pos] == '\t') {
		p.pos++
	}
}

func (p *scanner) eol() bool {
	return p.pos >= len(p.src) || p.src[p.pos] == '\n' || p.hasPrefix("\r\n")
}

func (p *scanner) nextLine() {
	if i := bytes.IndexByte(p.src[p.pos:], '\n'); i >= 0 {
		p.pos += i + 1
	} else {
		p.pos = len(p.src)
	}
}

// finishLine accepts trailing whitespace and a comment, then moves past the
// newline.
func (p *scanner) finishLine() error {
	p.skipSpace()
	if p.peek('#') || p.eol() {
		p.nextLine()
		return nil
	}
	return p.errorf("unexpected text after value")
}

// scanKeyUntil moves to the first stop byte outside quotes on this line.
func (p *scanner) scanKeyUntil(stop byte) error {
	for p.pos < len(p.src) {
		switch c := p.src[p.pos]; {
		case c == stop:
			return nil
		case c == '"':
			if err := p.scanBasicString(); err != nil {
				return err
			}
		case c == '\'':
			if err := p.scanLiteralString(); err != nil {
				return err
			}
		case c == '\n':
			return p.errorf("expected %q", stop)
		default:
			p.pos++
		}
	}
	return p.errorf("expected %q", stop)
}

func (p *scanner) scanBasicString() error {
	p.pos++ // opening quote
	for p.pos < len(p.src) {
		switch p.src[p.pos] {
		case '\\':
			p.pos += 2
		case '"':
			p.pos++
			return nil
		case '\n':
			return p.errorf("unterminated string")
		default:
			p.pos++
		}
	}
	return p.errorf("unterminated string")
}

func (p *scanner) scanLiteralString() error {
	p.pos++
	i := bytes.IndexAny(p.src[p.pos:], "'\n")
	if i < 0 || p.src[p.pos+i] != '\'' {
		return p.errorf("unterminated string")
	}
	p.pos += i + 1
	return nil
}

func (p *scanner) scanMultiline(delim string) error {
	p.pos += len(delim)
	for p.pos < len(p.src) {
		if delim == `"""` && p.src[p.pos] == '\\' {
			p.pos += 2
			continue
		}
		if p.hasPrefix(delim) {
			p.pos += len(delim)
			// Up to two quotes may directly precede the closing delimiter.
			for i := 0; i < 2 && p.peek(delim[0]); i++ {
				p.pos++
			}
			return nil
		}
		p.pos++
	}
	return p.errorf("unterminated multi-line string")
}

// scanValue moves past one value, which may span lines (arrays, inline
// tables and multi-line strings).
func (p *scanner) scanValue() error {
	switch {
	case p.hasPrefix(`"""`):
		return p.scanMultiline(`"""`)
	case p.hasPrefix(`'''`):
		return p.scanMultiline(`'''`)
	case p.peek('"'):
		return p.scanBasicString()
	case p.peek('\''):
		return p.scanLiteralString()
	case p.peek('[') || p.peek('{'):
		return p.scanComposite()
	}
	// Numbers, booleans and dates end at a comment or the end of the line;
	// local date-times may contain one space.
	start := p.pos
	for !p.eol() && !p.peek('#') {
		p.pos++
	}
	for p.pos > start && (p.src[p.pos-1] == ' ' || p.src[p.pos-1] == '\t') {
		p.pos--
	}
	if p.pos == start {
		return p.errorf("missing value")
	}
	return nil
}

func (p *scanner) scanComposite() error {
	depth := 0
	for p.pos < len(p.src) {
		switch {
		case p.hasPrefix(`"""`):
			if err := p.scanMultiline(`"""`); err != nil {
				return err
			}
			continue
		case p.hasPrefix(`'''`):
			if err := p.scanMultiline(`'''`); err != nil {
				return err
			}
			continue
		}
		switch p.src[p.pos] {
		case '"':
			if err := p.scanBasicString(); err != nil {
				return err
			}
			continue
		case '\'':
			if err := p.scanLiteralString(); err != nil {
				return err
			}
			continue
		case '#':
			p.nextLine()
			continue
		case '[', '{':
			depth++
		case ']', '}':
			depth--
			if depth == 0 {
				p.pos++
				return nil
			}
		}
		p.pos++
	}
	return p.errorf("unterminated array or inline table")
}

// normalizeKey turns `a . "b c"` into `a.b c` for comparisons.
func normalizeKey(raw string) string {
	var parts []string
	var cur strings.Builder
	raw = strings.TrimSpace(raw)
	for i := 0; i < len(raw); i++ {
		switch c := raw[i]; c {
		case '"':
			j := i + 1
			for j < len(raw) && raw[j] != '"' {
				if raw[j] == '\\' {
					j++
				}
				j++
			}
			var v map[string]any
			if _, err := toml.Decode("v = "+raw[i:min(j+1, len(raw))], &v); err == nil {
				cur.WriteString(fmt.Sprint(v["v"]))
			}
			i = j
		case '\'':
			j := strings.IndexByte(raw[i+1:], '\'')
			if j < 0 {
				j = len(raw) - i - 1
			}
			cur.WriteString(raw[i+1 : i+1+j])
			i += j + 1
		case '.':
			parts = append(parts, strings.TrimSpace(cur.String()))
			cur.Reset()
		case ' ', '\t':
			// whitespace around dots is insignificant
		default:
			cur.WriteByte(c)
		}
	}
	parts = append(parts, strings.TrimSpace(cur.String()))
	return strings.Join(parts, ".")
}
