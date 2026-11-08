// Package jsonc edits JSON-with-comments documents (Windows Terminal's
// settings.json, komorebi.json, ...) without disturbing the parts it does not
// change. Comments, trailing commas, member order and the existing
// indentation are kept; new values are indented to match their neighbours.
package jsonc

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/tailscale/hujson"
)

// ErrNotFound is returned when a pointer does not resolve.
var ErrNotFound = errors.New("json pointer not found")

// Doc is a parsed JSONC document.
type Doc struct {
	root   hujson.Value
	indent string // one indentation level, detected from the document
}

// Parse parses a JSONC document. An empty or whitespace-only input yields an
// empty object.
func Parse(data []byte) (*Doc, error) {
	if len(bytes.TrimSpace(data)) == 0 {
		data = []byte("{}\n")
	}
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf")) // UTF-8 BOM
	root, err := hujson.Parse(data)
	if err != nil {
		return nil, err
	}
	if _, ok := root.Value.(*hujson.Object); !ok {
		return nil, fmt.Errorf("top-level JSON value is not an object")
	}
	return &Doc{root: root, indent: detectIndent(root)}, nil
}

// Bytes returns the document, including any edits.
func (d *Doc) Bytes() []byte {
	return d.root.Pack()
}

// Get returns the standard JSON (comments removed) of the value at ptr.
func (d *Doc) Get(ptr string) (json.RawMessage, bool) {
	v := d.root.Find(ptr)
	if v == nil {
		return nil, false
	}
	return standard(*v), true
}

// GetString returns the string at ptr.
func (d *Doc) GetString(ptr string) (string, bool) {
	raw, ok := d.Get(ptr)
	if !ok {
		return "", false
	}
	var s string
	if json.Unmarshal(raw, &s) != nil {
		return "", false
	}
	return s, true
}

// Kind reports the JSON kind at ptr: '{', '[', '"', '0', 't', 'f', 'n', or 0
// when ptr does not resolve.
func (d *Doc) Kind(ptr string) byte {
	v := d.root.Find(ptr)
	if v == nil {
		return 0
	}
	return byte(v.Value.Kind())
}

// Len returns the number of elements of the array or members of the object
// at ptr, or -1 when ptr does not resolve to a composite value.
func (d *Doc) Len(ptr string) int {
	v := d.root.Find(ptr)
	if v == nil {
		return -1
	}
	switch c := v.Value.(type) {
	case *hujson.Object:
		return len(c.Members)
	case *hujson.Array:
		return len(c.Elements)
	}
	return -1
}

// Keys returns the member names of the object at ptr in document order.
func (d *Doc) Keys(ptr string) []string {
	v := d.root.Find(ptr)
	if v == nil {
		return nil
	}
	obj, ok := v.Value.(*hujson.Object)
	if !ok {
		return nil
	}
	keys := make([]string, 0, len(obj.Members))
	for _, m := range obj.Members {
		keys = append(keys, memberName(m))
	}
	return keys
}

// Set replaces the value at ptr, or adds it when ptr names a missing member
// of an existing object. raw must be valid JSON (it may contain comments).
func (d *Doc) Set(ptr string, raw []byte) error {
	parentPtr, name, err := splitPointer(ptr)
	if err != nil {
		return err
	}
	parent := d.root.Find(parentPtr)
	if parent == nil {
		return fmt.Errorf("%w: %s", ErrNotFound, parentPtr)
	}
	switch c := parent.Value.(type) {
	case *hujson.Object:
		for i := range c.Members {
			if memberName(c.Members[i]) == name {
				return d.replace(&c.Members[i].Value, memberIndent(c, i, d.depthIndent(parentPtr)), raw)
			}
		}
		return d.addMember(c, parentPtr, name, raw)
	case *hujson.Array:
		i, err := strconv.Atoi(name)
		if err != nil || i < 0 || i >= len(c.Elements) {
			return fmt.Errorf("%w: %s", ErrNotFound, ptr)
		}
		return d.replace(&c.Elements[i], elementIndent(c, i, d.depthIndent(parentPtr)), raw)
	default:
		return fmt.Errorf("cannot set %s: parent is not an object or array", ptr)
	}
}

// Append adds raw as the last element of the array at ptr.
func (d *Doc) Append(ptr string, raw []byte) error {
	v := d.root.Find(ptr)
	if v == nil {
		return fmt.Errorf("%w: %s", ErrNotFound, ptr)
	}
	arr, ok := v.Value.(*hujson.Array)
	if !ok {
		return fmt.Errorf("%s is not an array", ptr)
	}
	outer := d.depthIndent(ptr)
	inner, multiline := childLayout(len(arr.Elements), func(i int) hujson.Extra { return arr.Elements[i].BeforeExtra }, outer, d.indent)
	value, err := d.render(raw, inner, multiline)
	if err != nil {
		return err
	}
	if multiline {
		value.BeforeExtra = hujson.Extra(takeTrailingComment(&arr.AfterExtra) + "\n" + inner)
		if len(arr.Elements) == 0 || !bytes.Contains(arr.AfterExtra, []byte("\n")) {
			arr.AfterExtra = hujson.Extra("\n" + outer)
		}
	} else if len(arr.Elements) > 0 {
		value.BeforeExtra = hujson.Extra(" ")
	}
	arr.Elements = append(arr.Elements, value)
	d.root.UpdateOffsets()
	return nil
}

// Delete removes the member or element at ptr. A missing ptr is not an error.
func (d *Doc) Delete(ptr string) error {
	parentPtr, name, err := splitPointer(ptr)
	if err != nil {
		return err
	}
	parent := d.root.Find(parentPtr)
	if parent == nil {
		return nil
	}
	switch c := parent.Value.(type) {
	case *hujson.Object:
		for i := range c.Members {
			if memberName(c.Members[i]) == name {
				c.Members = append(c.Members[:i], c.Members[i+1:]...)
				break
			}
		}
	case *hujson.Array:
		i, err := strconv.Atoi(name)
		if err == nil && i >= 0 && i < len(c.Elements) {
			c.Elements = append(c.Elements[:i], c.Elements[i+1:]...)
		}
	}
	d.root.UpdateOffsets()
	return nil
}

// Ensure creates an empty object at ptr when it does not exist yet. It
// returns an error when ptr exists but is not an object.
func (d *Doc) Ensure(ptr string, empty string) error {
	if v := d.root.Find(ptr); v != nil {
		if byte(v.Value.Kind()) != empty[0] {
			return fmt.Errorf("%s is not a %s", ptr, kindName(empty[0]))
		}
		return nil
	}
	return d.Set(ptr, []byte(empty))
}

// MergePatch applies an RFC 7386 JSON merge patch to the object at ptr:
// members set to null are removed, objects are merged recursively and all
// other values replace the existing ones.
func (d *Doc) MergePatch(ptr string, patch []byte) error {
	p, err := hujson.Parse(patch)
	if err != nil {
		return fmt.Errorf("parse merge patch: %w", err)
	}
	obj, ok := p.Value.(*hujson.Object)
	if !ok {
		return fmt.Errorf("merge patch is not an object")
	}
	if err := d.Ensure(ptr, "{}"); err != nil {
		return err
	}
	for _, m := range obj.Members {
		name := memberName(m)
		child := ptr + "/" + EscapePointer(name)
		switch value := m.Value.Value.(type) {
		case hujson.Literal:
			if value.Kind() == 'n' {
				if err := d.Delete(child); err != nil {
					return err
				}
				continue
			}
		case *hujson.Object:
			if d.Kind(child) == '{' {
				if err := d.MergePatch(child, standard(m.Value)); err != nil {
					return err
				}
				continue
			}
		}
		if err := d.Set(child, standard(m.Value)); err != nil {
			return err
		}
	}
	return nil
}

// FindByName returns the index of the first object in the array at ptr whose
// "name" member equals name, or -1.
func (d *Doc) FindByName(ptr, name string) int {
	return d.FindBy(ptr, "name", name)
}

// FindBy returns the index of the first object in the array at ptr whose
// string member key equals value, or -1.
func (d *Doc) FindBy(ptr, key, value string) int {
	v := d.root.Find(ptr)
	if v == nil {
		return -1
	}
	arr, ok := v.Value.(*hujson.Array)
	if !ok {
		return -1
	}
	for i := range arr.Elements {
		el := arr.Elements[i]
		if s, ok := (&el).Find("/" + EscapePointer(key)).Value.(hujson.Literal); ok && s.Kind() == '"' && s.String() == value {
			return i
		}
	}
	return -1
}

// EscapePointer escapes one JSON pointer reference token (RFC 6901).
func EscapePointer(token string) string {
	return strings.NewReplacer("~", "~0", "/", "~1").Replace(token)
}

func unescapePointer(token string) string {
	return strings.NewReplacer("~1", "/", "~0", "~").Replace(token)
}

func splitPointer(ptr string) (parent, name string, err error) {
	i := strings.LastIndexByte(ptr, '/')
	if i < 0 {
		return "", "", fmt.Errorf("invalid json pointer %q", ptr)
	}
	return ptr[:i], unescapePointer(ptr[i+1:]), nil
}

func kindName(k byte) string {
	switch k {
	case '{':
		return "object"
	case '[':
		return "array"
	default:
		return "value"
	}
}

func memberName(m hujson.ObjectMember) string {
	if lit, ok := m.Name.Value.(hujson.Literal); ok {
		return lit.String()
	}
	return ""
}

func standard(v hujson.Value) []byte {
	c := v.Clone()
	c.BeforeExtra, c.AfterExtra = nil, nil
	c.Standardize()
	return c.Pack()
}

// replace swaps the value of target while keeping its surrounding comments
// and whitespace.
func (d *Doc) replace(target *hujson.Value, indent string, raw []byte) error {
	multiline := target.Value.Kind() != '{' && target.Value.Kind() != '[' || hasNewline(target.Value)
	value, err := d.render(raw, indent, multiline)
	if err != nil {
		return err
	}
	target.Value = value.Value
	d.root.UpdateOffsets()
	return nil
}

func (d *Doc) addMember(obj *hujson.Object, parentPtr, name string, raw []byte) error {
	outer := d.depthIndent(parentPtr)
	inner, multiline := childLayout(len(obj.Members), func(i int) hujson.Extra { return obj.Members[i].Name.BeforeExtra }, outer, d.indent)
	value, err := d.render(raw, inner, multiline)
	if err != nil {
		return err
	}
	value.BeforeExtra = hujson.Extra(" ")
	key, _ := json.Marshal(name)
	member := hujson.ObjectMember{Name: hujson.Value{Value: hujson.Literal(key)}, Value: value}
	if multiline {
		member.Name.BeforeExtra = hujson.Extra(takeTrailingComment(&obj.AfterExtra) + "\n" + inner)
		if len(obj.Members) == 0 || !bytes.Contains(obj.AfterExtra, []byte("\n")) {
			obj.AfterExtra = hujson.Extra("\n" + outer)
		}
	} else if len(obj.Members) > 0 {
		member.Name.BeforeExtra = hujson.Extra(" ")
	}
	obj.Members = append(obj.Members, member)
	d.root.UpdateOffsets()
	return nil
}

// takeTrailingComment removes and returns a comment that trails the current
// last child on its own line ("x": 1 // note). When a child is appended the
// comment has to follow the new comma, so it moves to the new child's prefix.
func takeTrailingComment(after *hujson.Extra) string {
	text := string(*after)
	nl := strings.IndexByte(text, '\n')
	if nl <= 0 || strings.TrimSpace(text[:nl]) == "" {
		return ""
	}
	*after = hujson.Extra(text[nl:])
	return text[:nl]
}

// render parses raw and lays it out for the given indentation. Composite
// values are indented with the document's unit when multiline is set and
// compacted otherwise.
func (d *Doc) render(raw []byte, indent string, multiline bool) (hujson.Value, error) {
	std, err := hujson.Standardize(append([]byte(nil), raw...))
	if err != nil {
		return hujson.Value{}, fmt.Errorf("invalid JSON value: %w", err)
	}
	var out bytes.Buffer
	if multiline {
		err = json.Indent(&out, std, indent, d.indent)
	} else {
		err = compactSpaced(&out, std)
	}
	if err != nil {
		return hujson.Value{}, err
	}
	v, err := hujson.Parse(out.Bytes())
	if err != nil {
		return hujson.Value{}, err
	}
	v.BeforeExtra, v.AfterExtra = nil, nil
	return v, nil
}

// compactSpaced writes {"a": 1, "b": 2} style output for one-line values.
func compactSpaced(out *bytes.Buffer, std []byte) error {
	var compact bytes.Buffer
	if err := json.Compact(&compact, std); err != nil {
		return err
	}
	inString, escaped := false, false
	for _, c := range compact.Bytes() {
		out.WriteByte(c)
		if inString {
			switch {
			case escaped:
				escaped = false
			case c == '\\':
				escaped = true
			case c == '"':
				inString = false
			}
			continue
		}
		switch c {
		case '"':
			inString = true
		case ':', ',':
			out.WriteByte(' ')
		}
	}
	return nil
}

func hasNewline(v hujson.ValueTrimmed) bool {
	return bytes.Contains(hujson.Value{Value: v}.Pack(), []byte("\n"))
}

// childLayout returns the indentation and line style for a new child of a
// composite whose own line is indented with outer.
func childLayout(n int, before func(int) hujson.Extra, outer, unit string) (string, bool) {
	for i := n - 1; i >= 0; i-- {
		b := string(before(i))
		if nl := strings.LastIndexByte(b, '\n'); nl >= 0 {
			return b[nl+1:], true
		}
		return "", false // siblings share a line: keep the new one inline
	}
	return outer + unit, true
}

func memberIndent(obj *hujson.Object, i int, fallback string) string {
	b := string(obj.Members[i].Name.BeforeExtra)
	if nl := strings.LastIndexByte(b, '\n'); nl >= 0 {
		return b[nl+1:]
	}
	return fallback
}

func elementIndent(arr *hujson.Array, i int, fallback string) string {
	b := string(arr.Elements[i].BeforeExtra)
	if nl := strings.LastIndexByte(b, '\n'); nl >= 0 {
		return b[nl+1:]
	}
	return fallback
}

// depthIndent returns the indentation of the line that holds the value at ptr,
// derived from the pointer depth.
func (d *Doc) depthIndent(ptr string) string {
	if ptr == "" {
		return ""
	}
	return strings.Repeat(d.indent, strings.Count(ptr, "/"))
}

func detectIndent(root hujson.Value) string {
	obj := root.Value.(*hujson.Object)
	for _, m := range obj.Members {
		b := string(m.Name.BeforeExtra)
		if nl := strings.LastIndexByte(b, '\n'); nl >= 0 && nl+1 < len(b) {
			return b[nl+1:]
		}
	}
	return "    "
}
