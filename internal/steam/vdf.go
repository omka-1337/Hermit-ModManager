package steam

import (
	"errors"
	"fmt"
	"strings"
)

// KeyValues is a node of Valve's text KeyValues format (.vdf, .acf).
// A node holds either a string value or child nodes.
type KeyValues struct {
	Value    string
	Children []Pair
}

type Pair struct {
	Key   string
	Value *KeyValues
}

// Get returns the first child with the given key, compared case-insensitively
// since Steam is inconsistent about key casing.
func (kv *KeyValues) Get(key string) *KeyValues {
	if kv == nil {
		return nil
	}
	for _, p := range kv.Children {
		if strings.EqualFold(p.Key, key) {
			return p.Value
		}
	}
	return nil
}

// String returns the string value at the given key path, or "".
func (kv *KeyValues) String(path ...string) string {
	for _, key := range path {
		kv = kv.Get(key)
	}
	if kv == nil {
		return ""
	}
	return kv.Value
}

func ParseVDF(data string) (*KeyValues, error) {
	p := &vdfParser{data: data}
	root, err := p.parseChildren(false)
	if err != nil {
		return nil, fmt.Errorf("parse vdf at offset %d: %w", p.pos, err)
	}
	return root, nil
}

type vdfParser struct {
	data string
	pos  int
}

func (p *vdfParser) parseChildren(nested bool) (*KeyValues, error) {
	node := &KeyValues{}
	for {
		tok, quoted, err := p.next()
		if err != nil {
			return nil, err
		}
		switch {
		case tok == "" && !quoted:
			if nested {
				return nil, errors.New("unexpected end of input")
			}
			return node, nil
		case tok == "}" && !quoted:
			if !nested {
				return nil, errors.New("unexpected '}'")
			}
			return node, nil
		case tok == "{" && !quoted:
			return nil, errors.New("unexpected '{'")
		}

		key := tok
		val, valQuoted, err := p.next()
		if err != nil {
			return nil, err
		}
		switch {
		case val == "{" && !valQuoted:
			child, err := p.parseChildren(true)
			if err != nil {
				return nil, err
			}
			node.Children = append(node.Children, Pair{key, child})
		case val == "" && !valQuoted, val == "}" && !valQuoted:
			return nil, fmt.Errorf("missing value for key %q", key)
		default:
			node.Children = append(node.Children, Pair{key, &KeyValues{Value: val}})
		}
	}
}

// next returns the next token; quoted reports whether it was a quoted string.
// An empty unquoted token means end of input.
func (p *vdfParser) next() (tok string, quoted bool, err error) {
	for p.pos < len(p.data) {
		c := p.data[p.pos]
		switch {
		case c == ' ' || c == '\t' || c == '\r' || c == '\n':
			p.pos++
		case strings.HasPrefix(p.data[p.pos:], "//"):
			if i := strings.IndexByte(p.data[p.pos:], '\n'); i >= 0 {
				p.pos += i + 1
			} else {
				p.pos = len(p.data)
			}
		case c == '{' || c == '}':
			p.pos++
			return string(c), false, nil
		case c == '"':
			return p.quoted()
		default:
			start := p.pos
			for p.pos < len(p.data) && !strings.ContainsRune(" \t\r\n{}\"", rune(p.data[p.pos])) {
				p.pos++
			}
			return p.data[start:p.pos], true, nil
		}
	}
	return "", false, nil
}

func (p *vdfParser) quoted() (string, bool, error) {
	p.pos++ // opening quote
	var b strings.Builder
	for p.pos < len(p.data) {
		c := p.data[p.pos]
		switch {
		case c == '"':
			p.pos++
			return b.String(), true, nil
		case c == '\\' && p.pos+1 < len(p.data):
			switch n := p.data[p.pos+1]; n {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			default:
				b.WriteByte(n)
			}
			p.pos += 2
		default:
			b.WriteByte(c)
			p.pos++
		}
	}
	return "", false, errors.New("unterminated string")
}
