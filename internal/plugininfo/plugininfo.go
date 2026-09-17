package plugininfo

import (
	"encoding/binary"
	"errors"
	"os"
	"sort"
	"unicode/utf8"
)

// Plugin is a class marked with [BepInPlugin].
type Plugin struct {
	GUID    string `json:"guid"`
	Name    string `json:"name"`
	Version string `json:"version"`
	// Class is the full name of the plugin class.
	Class             string       `json:"class"`
	Dependencies      []Dependency `json:"dependencies"`
	Incompatibilities []string     `json:"incompatibilities"`
	// Processes limits the plugin to these game executables, if any.
	Processes []string `json:"processes"`
}

// Dependency is a [BepInDependency] on another plugin GUID.
type Dependency struct {
	GUID string `json:"guid"`
	Soft bool   `json:"soft"`
	// MinVersion is set by the (guid, version range) constructor.
	MinVersion string `json:"minVersion"`
}

// Read returns the BepInEx plugins declared in an assembly file. Assemblies
// without plugins (libraries) return an empty list.
func Read(path string) ([]Plugin, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(data)
}

// Parse is Read for assembly bytes.
func Parse(data []byte) (plugins []Plugin, err error) {
	// The parser indexes into untrusted data; turn any overlooked corruption
	// into an error instead of crashing the manager.
	defer func() {
		if r := recover(); r != nil {
			plugins, err = nil, errors.New("malformed assembly")
		}
	}()
	pe, cliRVA, err := parsePE(data)
	if err != nil {
		return nil, err
	}
	md, err := readMetadata(pe, cliRVA)
	if err != nil {
		return nil, err
	}
	return md.plugins()
}

const (
	dependencyFlagSoft = 2
)

func (md *metadata) plugins() ([]Plugin, error) {
	byClass := map[uint32]*Plugin{}
	extras := map[uint32][]attribute{}

	for i := uint32(1); i <= md.rows[tCustomAttribute]; i++ {
		attr, ok := md.attribute(i)
		if !ok || attr.parentTable != tTypeDef {
			continue
		}
		switch attr.name {
		case "BepInPlugin":
			if len(attr.args) == 3 {
				byClass[attr.parent] = &Plugin{
					GUID: attr.args[0].s, Name: attr.args[1].s, Version: attr.args[2].s,
					Class:             md.typeDefName(attr.parent),
					Dependencies:      []Dependency{},
					Incompatibilities: []string{},
					Processes:         []string{},
				}
			}
		case "BepInDependency", "BepInIncompatibility", "BepInProcess":
			extras[attr.parent] = append(extras[attr.parent], attr)
		}
	}

	var plugins []Plugin
	for class, p := range byClass {
		for _, a := range extras[class] {
			if len(a.args) == 0 {
				continue
			}
			switch a.name {
			case "BepInDependency":
				d := Dependency{GUID: a.args[0].s}
				if len(a.args) > 1 {
					if a.args[1].isString {
						d.MinVersion = a.args[1].s
					} else {
						d.Soft = a.args[1].i&dependencyFlagSoft != 0
					}
				}
				p.Dependencies = append(p.Dependencies, d)
			case "BepInIncompatibility":
				p.Incompatibilities = append(p.Incompatibilities, a.args[0].s)
			case "BepInProcess":
				p.Processes = append(p.Processes, a.args[0].s)
			}
		}
		plugins = append(plugins, *p)
	}
	sort.Slice(plugins, func(i, j int) bool { return plugins[i].GUID < plugins[j].GUID })
	if plugins == nil {
		plugins = []Plugin{}
	}
	return plugins, nil
}

type argument struct {
	s        string
	i        int32
	isString bool
}

type attribute struct {
	name        string
	parentTable int
	parent      uint32
	args        []argument
}

// attribute decodes custom attribute row i if its constructor is a BepInEx type.
func (md *metadata) attribute(i uint32) (attribute, bool) {
	raw, err := md.row(tCustomAttribute, i)
	if err != nil {
		return attribute{}, false
	}
	c := cursor{b: raw}
	parent := c.read(md.coded(ciHasCustomAttribute))
	ctor := c.read(md.coded(ciCustomAttributeTyp))
	value, err := md.blobAt(c.read(md.heap(md.wideBlob)))
	if err != nil {
		return attribute{}, false
	}

	// Only constructors referenced from another assembly (MemberRef) can be
	// BepInEx's attribute types.
	if ctor&7 != 3 {
		return attribute{}, false
	}
	memberRef, err := md.row(tMemberRef, ctor>>3)
	if err != nil {
		return attribute{}, false
	}
	mc := cursor{b: memberRef}
	class := mc.read(md.coded(ciMemberRefParent))
	mc.read(md.heap(md.wideString)) // .ctor
	signature, err := md.blobAt(mc.read(md.heap(md.wideBlob)))
	if err != nil || class&7 != 1 { // declaring type must be a TypeRef
		return attribute{}, false
	}
	typeRef, err := md.row(tTypeRef, class>>3)
	if err != nil {
		return attribute{}, false
	}
	tc := cursor{b: typeRef}
	tc.read(md.coded(ciResolutionScope))
	name := md.str(tc.read(md.heap(md.wideString)))
	namespace := md.str(tc.read(md.heap(md.wideString)))
	if namespace != "BepInEx" {
		return attribute{}, false
	}

	parentBits := 5
	attr := attribute{
		name:        name,
		parentTable: ciHasCustomAttribute[parent&(1<<parentBits-1)],
		parent:      parent >> parentBits,
	}
	params, ok := paramTypes(signature)
	if !ok {
		return attribute{}, false
	}
	attr.args, ok = decodeArgs(value, params)
	return attr, ok
}

// Element types used by BepInEx attribute constructors.
const (
	elemI4        = 0x08
	elemString    = 0x0E
	elemValueType = 0x11 // enums such as DependencyFlags, stored as int32
)

// paramTypes reads the parameter types of a constructor signature.
func paramTypes(sig []byte) ([]byte, bool) {
	if len(sig) < 3 || sig[0]&0x10 != 0 { // generic constructors are not expected
		return nil, false
	}
	count, n, err := compressedUint(sig[1:])
	if err != nil {
		return nil, false
	}
	p := 1 + n
	if p >= len(sig) || sig[p] != 0x01 { // void return type
		return nil, false
	}
	p++
	var types []byte
	for i := uint32(0); i < count; i++ {
		if p >= len(sig) {
			return nil, false
		}
		switch t := sig[p]; t {
		case elemString, elemI4:
			types = append(types, t)
			p++
		case elemValueType:
			_, n, err := compressedUint(sig[p+1:])
			if err != nil {
				return nil, false
			}
			types = append(types, t)
			p += 1 + n
		default:
			return nil, false
		}
	}
	return types, true
}

// decodeArgs reads fixed constructor arguments from a custom attribute blob.
func decodeArgs(value, types []byte) ([]argument, bool) {
	if len(value) < 2 || binary.LittleEndian.Uint16(value) != 1 {
		return nil, false
	}
	p := 2
	args := make([]argument, 0, len(types))
	for _, t := range types {
		switch t {
		case elemString:
			if p >= len(value) {
				return nil, false
			}
			if value[p] == 0xFF { // null string
				args = append(args, argument{isString: true})
				p++
				continue
			}
			n, size, err := compressedUint(value[p:])
			if err != nil || p+size+int(n) > len(value) {
				return nil, false
			}
			s := value[p+size : p+size+int(n)]
			if !utf8.Valid(s) {
				return nil, false
			}
			args = append(args, argument{s: string(s), isString: true})
			p += size + int(n)
		default:
			if p+4 > len(value) {
				return nil, false
			}
			args = append(args, argument{i: int32(binary.LittleEndian.Uint32(value[p:]))})
			p += 4
		}
	}
	return args, true
}

func (md *metadata) typeDefName(i uint32) string {
	raw, err := md.row(tTypeDef, i)
	if err != nil {
		return ""
	}
	c := cursor{b: raw}
	c.read(4)
	name := md.str(c.read(md.heap(md.wideString)))
	namespace := md.str(c.read(md.heap(md.wideString)))
	if namespace == "" {
		return name
	}
	return namespace + "." + name
}
