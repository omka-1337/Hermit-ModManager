package configs

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type Section struct {
	Name    string  `json:"name"`
	Entries []Entry `json:"entries"`
}

type Entry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	// Line is the 0-based line of "key = value", used to save changes.
	Line        int    `json:"line"`
	Description string `json:"description"`
	// Type is BepInEx's setting type, e.g. Boolean, Int32, Single, String, or an enum name.
	Type    string `json:"type"`
	Default string `json:"default"`
	// HasDefault distinguishes an empty default from a missing one.
	HasDefault bool `json:"hasDefault"`
	// AcceptableValues lists enum values; Multiple means flags that combine with ", ".
	AcceptableValues []string `json:"acceptableValues"`
	Multiple         bool     `json:"multiple"`
	// Min and Max bound numeric settings when HasRange is set.
	HasRange bool    `json:"hasRange"`
	Min      float64 `json:"min"`
	Max      float64 `json:"max"`
}

type Document struct {
	// Header holds the description lines before the first section.
	Header   []string  `json:"header"`
	Sections []Section `json:"sections"`
}

var (
	settingType = regexp.MustCompile(`^#\s*Setting type:\s*(.*)$`)
	defaultVal  = regexp.MustCompile(`^#\s*Default value:\s?(.*)$`)
	acceptable  = regexp.MustCompile(`^#\s*Acceptable values:\s*(.*)$`)
	valueRange  = regexp.MustCompile(`^#\s*Acceptable value range:\s*From\s+(\S+)\s+to\s+(\S+)`)
	multiple    = regexp.MustCompile(`^#\s*Multiple values can be set at the same time`)
)

// ParseCfg parses a BepInEx config file.
func ParseCfg(text string) Document {
	doc := Document{Header: []string{}, Sections: []Section{}}
	var pending Entry
	var description []string
	reset := func() {
		pending = Entry{}
		description = nil
	}
	for i, raw := range splitLines(text) {
		line := strings.TrimSpace(raw)
		switch {
		case line == "":
		case strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]"):
			doc.Sections = append(doc.Sections, Section{Name: line[1 : len(line)-1], Entries: []Entry{}})
			reset()
		case strings.HasPrefix(line, "##"):
			text := strings.TrimSpace(strings.TrimPrefix(line, "##"))
			if len(doc.Sections) == 0 {
				doc.Header = append(doc.Header, text)
			} else {
				description = append(description, text)
			}
		case strings.HasPrefix(line, "#"):
			if m := settingType.FindStringSubmatch(line); m != nil {
				pending.Type = strings.TrimSpace(m[1])
			} else if m := defaultVal.FindStringSubmatch(line); m != nil {
				pending.Default, pending.HasDefault = m[1], true
			} else if m := acceptable.FindStringSubmatch(line); m != nil {
				pending.AcceptableValues = splitList(m[1])
			} else if m := valueRange.FindStringSubmatch(line); m != nil {
				lo, err1 := strconv.ParseFloat(m[1], 64)
				hi, err2 := strconv.ParseFloat(m[2], 64)
				if err1 == nil && err2 == nil {
					pending.HasRange, pending.Min, pending.Max = true, lo, hi
				}
			} else if multiple.MatchString(line) {
				pending.Multiple = true
			}
		default:
			key, value, ok := strings.Cut(line, "=")
			if !ok || len(doc.Sections) == 0 || strings.TrimSpace(key) == "" {
				reset()
				continue
			}
			pending.Key = strings.TrimSpace(key)
			pending.Value = strings.TrimSpace(value)
			pending.Line = i
			pending.Description = strings.Join(description, "\n")
			if pending.AcceptableValues == nil {
				pending.AcceptableValues = []string{}
			}
			s := &doc.Sections[len(doc.Sections)-1]
			s.Entries = append(s.Entries, pending)
			reset()
		}
	}
	return doc
}

// Change sets the value of one entry; Line comes from ParseCfg and Section
// and Key guard against the file having changed in between.
type Change struct {
	Section string `json:"section"`
	Key     string `json:"key"`
	Line    int    `json:"line"`
	Value   string `json:"value"`
}

// ApplyChanges rewrites the values of the given entries in text, keeping
// everything else, including line endings, untouched.
func ApplyChanges(text string, changes []Change) (string, error) {
	lines := strings.SplitAfter(text, "\n")
	sectionAt := make([]string, len(lines))
	current := ""
	for i, l := range lines {
		t := strings.TrimSpace(l)
		if strings.HasPrefix(t, "[") && strings.HasSuffix(t, "]") {
			current = t[1 : len(t)-1]
		}
		sectionAt[i] = current
	}
	for _, c := range changes {
		if strings.ContainsAny(c.Value, "\r\n") {
			return "", fmt.Errorf("%s: value must be a single line", c.Key)
		}
		i := c.Line
		if i < 0 || i >= len(lines) || sectionAt[i] != c.Section || lineKey(lines[i]) != c.Key {
			// The file changed since it was read; find the entry by name.
			i = -1
			for j, l := range lines {
				if sectionAt[j] == c.Section && lineKey(l) == c.Key {
					i = j
					break
				}
			}
			if i < 0 {
				return "", fmt.Errorf("setting %q in [%s] not found; reload the file", c.Key, c.Section)
			}
		}
		lines[i] = replaceValue(lines[i], c.Value)
	}
	return strings.Join(lines, ""), nil
}

func lineKey(line string) string {
	t := strings.TrimSpace(line)
	if t == "" || strings.HasPrefix(t, "#") || strings.HasPrefix(t, "[") {
		return ""
	}
	key, _, ok := strings.Cut(t, "=")
	if !ok {
		return ""
	}
	return strings.TrimSpace(key)
}

// replaceValue swaps the value after "=", keeping the key and the line ending.
// It writes "Key = value" like BepInEx does, including "Key = " when empty.
func replaceValue(line, value string) string {
	body, eol := line, ""
	for _, e := range []string{"\r\n", "\n"} {
		if strings.HasSuffix(body, e) {
			body, eol = strings.TrimSuffix(body, e), e
			break
		}
	}
	key := body[:strings.Index(body, "=")]
	return strings.TrimRight(key, " \t") + " = " + value + eol
}

func splitLines(text string) []string {
	return strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
}

func splitList(s string) []string {
	out := []string{}
	for _, v := range strings.Split(s, ",") {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}
