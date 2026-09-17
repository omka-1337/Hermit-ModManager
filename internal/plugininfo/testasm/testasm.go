// Package testasm builds minimal .NET assemblies with BepInEx attributes for tests.
package testasm

import (
	"bytes"
	"encoding/binary"
)

// Table numbers used by the builder (ECMA-335 II.22).
const (
	tModule          = 0x00
	tTypeRef         = 0x01
	tTypeDef         = 0x02
	tMemberRef       = 0x0A
	tCustomAttribute = 0x0C
)

// Builder writes a minimal PE32 image with CLI metadata: enough tables for
// custom attributes on a type.
type Builder struct {
	strings, blobs bytes.Buffer
	typeRefs       [][3]uint16 // scope, name, namespace
	// TypeDefs holds name and namespace string indexes; row 1 is <Module>.
	TypeDefs         [][3]uint16
	memberRefs       [][3]uint16 // class, name, signature
	customAttributes [][3]uint16 // parent, type, value
}

func New() *Builder {
	b := &Builder{}
	b.strings.WriteByte(0)
	b.blobs.WriteByte(0)
	b.Str("Test.dll")
	b.TypeDefs = append(b.TypeDefs, [3]uint16{b.Str("<Module>"), 0})
	return b
}

func (b *Builder) Str(s string) uint16 {
	i := uint16(b.strings.Len())
	b.strings.WriteString(s)
	b.strings.WriteByte(0)
	return i
}

func (b *Builder) blob(data []byte) uint16 {
	i := uint16(b.blobs.Len())
	b.blobs.WriteByte(byte(len(data))) // tests keep blobs under 128 bytes
	b.blobs.Write(data)
	return i
}

func SerString(s string) []byte { return append([]byte{byte(len(s))}, s...) }

// attribute adds a custom attribute with a constructor on namespace.name.
func (b *Builder) Attribute(typeDef int, namespace, name string, sig []byte, args ...[]byte) {
	b.typeRefs = append(b.typeRefs, [3]uint16{1<<2 | 2, b.Str(name), b.Str(namespace)})
	class := uint16(len(b.typeRefs))<<3 | 1
	b.memberRefs = append(b.memberRefs, [3]uint16{class, b.Str(".ctor"), b.blob(sig)})
	value := []byte{1, 0}
	for _, a := range args {
		value = append(value, a...)
	}
	value = append(value, 0, 0) // no named arguments
	ctor := uint16(len(b.memberRefs))<<3 | 3
	b.customAttributes = append(b.customAttributes, [3]uint16{uint16(typeDef)<<5 | 3, ctor, b.blob(value)})
}

func (b *Builder) Build() []byte {
	w16 := func(buf *bytes.Buffer, vs ...uint16) {
		for _, v := range vs {
			binary.Write(buf, binary.LittleEndian, v)
		}
	}
	var tables bytes.Buffer
	binary.Write(&tables, binary.LittleEndian, uint32(0))
	tables.Write([]byte{2, 0, 0, 1})
	valid := uint64(1<<tModule | 1<<tTypeRef | 1<<tTypeDef | 1<<tMemberRef | 1<<tCustomAttribute)
	binary.Write(&tables, binary.LittleEndian, valid)
	binary.Write(&tables, binary.LittleEndian, uint64(0))
	for _, n := range []int{1, len(b.typeRefs), len(b.TypeDefs), len(b.memberRefs), len(b.customAttributes)} {
		binary.Write(&tables, binary.LittleEndian, uint32(n))
	}
	w16(&tables, 0, 1, 0, 0, 0) // Module
	for _, r := range b.typeRefs {
		w16(&tables, r[:]...)
	}
	for _, r := range b.TypeDefs {
		binary.Write(&tables, binary.LittleEndian, uint32(0))
		w16(&tables, r[0], r[1], 0, 1, 1)
	}
	for _, r := range b.memberRefs {
		w16(&tables, r[:]...)
	}
	for _, r := range b.customAttributes {
		w16(&tables, r[:]...)
	}

	pad := func(buf *bytes.Buffer) {
		for buf.Len()%4 != 0 {
			buf.WriteByte(0)
		}
	}
	pad(&tables)
	pad(&b.strings)
	pad(&b.blobs)

	streams := []struct {
		name string
		data []byte
	}{{"#~", tables.Bytes()}, {"#Strings", b.strings.Bytes()}, {"#Blob", b.blobs.Bytes()}}
	version := "v4.0.30319\x00\x00"
	headerSize := 16 + len(version) + 4
	for _, s := range streams {
		headerSize += 8 + (len(s.name)+4)&^3
	}
	var md bytes.Buffer
	binary.Write(&md, binary.LittleEndian, []uint32{0x424A5342})
	w16(&md, 1, 1)
	binary.Write(&md, binary.LittleEndian, []uint32{0, uint32(len(version))})
	md.WriteString(version)
	w16(&md, 0, uint16(len(streams)))
	offset := headerSize
	for _, s := range streams {
		binary.Write(&md, binary.LittleEndian, []uint32{uint32(offset), uint32(len(s.data))})
		name := []byte(s.name)
		name = append(name, make([]byte, (len(s.name)+4)&^3-len(s.name))...)
		md.Write(name)
		offset += len(s.data)
	}
	for _, s := range streams {
		md.Write(s.data)
	}

	const fileAlign, rva = 0x200, 0x2000
	var sectionData bytes.Buffer
	binary.Write(&sectionData, binary.LittleEndian, []uint32{72, 0x00050002, rva + 72, uint32(md.Len())})
	sectionData.Write(make([]byte, 72-16))
	sectionData.Write(md.Bytes())

	img := make([]byte, fileAlign)
	copy(img, "MZ")
	binary.LittleEndian.PutUint32(img[0x3c:], 0x40)
	copy(img[0x40:], "PE\x00\x00")
	coff := 0x44
	binary.LittleEndian.PutUint16(img[coff:], 0x14c)
	binary.LittleEndian.PutUint16(img[coff+2:], 1)
	binary.LittleEndian.PutUint16(img[coff+16:], 224)
	opt := coff + 20
	binary.LittleEndian.PutUint16(img[opt:], 0x10b)
	binary.LittleEndian.PutUint32(img[opt+96+14*8:], rva)
	sec := opt + 224
	copy(img[sec:], ".text")
	binary.LittleEndian.PutUint32(img[sec+8:], uint32(sectionData.Len()))
	binary.LittleEndian.PutUint32(img[sec+12:], rva)
	binary.LittleEndian.PutUint32(img[sec+16:], uint32(sectionData.Len()))
	binary.LittleEndian.PutUint32(img[sec+20:], fileAlign)
	return append(img, sectionData.Bytes()...)
}

// Plugin describes a plugin class for Plugin.
type Plugin struct {
	Class, GUID, Name, Version string
}

// PluginDLL returns an assembly with one [BepInPlugin] class per plugin.
func PluginDLL(plugins ...Plugin) []byte {
	b := New()
	for _, p := range plugins {
		b.TypeDefs = append(b.TypeDefs, [3]uint16{b.Str(p.Class), 0})
		b.Attribute(len(b.TypeDefs), "BepInEx", "BepInPlugin", []byte{0x20, 3, 0x01, 0x0E, 0x0E, 0x0E},
			SerString(p.GUID), SerString(p.Name), SerString(p.Version))
	}
	return b.Build()
}
