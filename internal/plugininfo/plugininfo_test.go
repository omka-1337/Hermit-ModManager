package plugininfo

import (
	"bytes"
	"encoding/binary"
	"errors"
	"reflect"
	"testing"
)

// asmBuilder writes a minimal PE32 image with CLI metadata: enough tables for
// custom attributes on a type.
type asmBuilder struct {
	strings, blobs   bytes.Buffer
	typeRefs         [][3]uint16 // scope, name, namespace
	typeDefs         [][3]uint16 // name, namespace (flags/extends/lists fixed)
	memberRefs       [][3]uint16 // class, name, signature
	customAttributes [][3]uint16 // parent, type, value
}

func newBuilder() *asmBuilder {
	b := &asmBuilder{}
	b.strings.WriteByte(0)
	b.blobs.WriteByte(0)
	b.str("Test.dll")
	b.typeDefs = append(b.typeDefs, [3]uint16{b.str("<Module>"), 0})
	return b
}

func (b *asmBuilder) str(s string) uint16 {
	i := uint16(b.strings.Len())
	b.strings.WriteString(s)
	b.strings.WriteByte(0)
	return i
}

func (b *asmBuilder) blob(data []byte) uint16 {
	i := uint16(b.blobs.Len())
	b.blobs.WriteByte(byte(len(data))) // tests keep blobs under 128 bytes
	b.blobs.Write(data)
	return i
}

func serString(s string) []byte { return append([]byte{byte(len(s))}, s...) }

// attribute adds a custom attribute with a constructor on namespace.name.
func (b *asmBuilder) attribute(typeDef int, namespace, name string, sig []byte, args ...[]byte) {
	b.typeRefs = append(b.typeRefs, [3]uint16{1<<2 | 2, b.str(name), b.str(namespace)})
	class := uint16(len(b.typeRefs))<<3 | 1
	b.memberRefs = append(b.memberRefs, [3]uint16{class, b.str(".ctor"), b.blob(sig)})
	value := []byte{1, 0}
	for _, a := range args {
		value = append(value, a...)
	}
	value = append(value, 0, 0) // no named arguments
	ctor := uint16(len(b.memberRefs))<<3 | 3
	b.customAttributes = append(b.customAttributes, [3]uint16{uint16(typeDef)<<5 | 3, ctor, b.blob(value)})
}

func (b *asmBuilder) build() []byte {
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
	for _, n := range []int{1, len(b.typeRefs), len(b.typeDefs), len(b.memberRefs), len(b.customAttributes)} {
		binary.Write(&tables, binary.LittleEndian, uint32(n))
	}
	w16(&tables, 0, 1, 0, 0, 0) // Module
	for _, r := range b.typeRefs {
		w16(&tables, r[:]...)
	}
	for _, r := range b.typeDefs {
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

func TestParsePluginAttributes(t *testing.T) {
	b := newBuilder()
	b.typeDefs = append(b.typeDefs, [3]uint16{b.str("Plugin"), b.str("MoreCompany")})
	const class = 2
	threeStrings := []byte{0x20, 3, 0x01, 0x0E, 0x0E, 0x0E}
	b.attribute(class, "BepInEx", "BepInPlugin", threeStrings,
		serString("me.swipez.melonloader.morecompany"), serString("MoreCompany"), serString("1.14.0"))
	// BepInDependency(string, DependencyFlags) with a soft flag.
	b.attribute(class, "BepInEx", "BepInDependency", []byte{0x20, 2, 0x01, 0x0E, 0x11, 0x0D},
		serString("com.lib.optional"), []byte{2, 0, 0, 0})
	// BepInDependency(string, string) with a minimum version.
	b.attribute(class, "BepInEx", "BepInDependency", []byte{0x20, 2, 0x01, 0x0E, 0x0E},
		serString("com.lib.core"), serString("1.2.0"))
	b.attribute(class, "BepInEx", "BepInIncompatibility", []byte{0x20, 1, 0x01, 0x0E}, serString("com.other.lobby"))
	b.attribute(class, "BepInEx", "BepInProcess", []byte{0x20, 1, 0x01, 0x0E}, serString("Lethal Company.exe"))
	// Attributes from other namespaces are ignored.
	b.attribute(class, "System.Runtime.CompilerServices", "BepInPlugin", threeStrings,
		serString("x"), serString("y"), serString("z"))

	plugins, err := Parse(b.build())
	if err != nil {
		t.Fatal(err)
	}
	want := []Plugin{{
		GUID: "me.swipez.melonloader.morecompany", Name: "MoreCompany", Version: "1.14.0",
		Class: "MoreCompany.Plugin",
		Dependencies: []Dependency{
			{GUID: "com.lib.optional", Soft: true},
			{GUID: "com.lib.core", MinVersion: "1.2.0"},
		},
		Incompatibilities: []string{"com.other.lobby"},
		Processes:         []string{"Lethal Company.exe"},
	}}
	if !reflect.DeepEqual(plugins, want) {
		t.Errorf("plugins:\n got  %+v\n want %+v", plugins, want)
	}
}

func TestParseLibraryAndGarbage(t *testing.T) {
	plugins, err := Parse(newBuilder().build())
	if err != nil || len(plugins) != 0 {
		t.Errorf("library: %v %v", plugins, err)
	}
	for _, data := range [][]byte{nil, []byte("MZ"), bytes.Repeat([]byte{0xFF}, 512)} {
		if _, err := Parse(data); !errors.Is(err, ErrNotAssembly) {
			t.Errorf("garbage %q: %v", data, err)
		}
	}
	// Truncated images must fail cleanly, never panic.
	img := func() []byte {
		b := newBuilder()
		b.typeDefs = append(b.typeDefs, [3]uint16{b.str("P"), 0})
		b.attribute(2, "BepInEx", "BepInPlugin", []byte{0x20, 3, 0x01, 0x0E, 0x0E, 0x0E}, serString("a"), serString("b"), serString("c"))
		return b.build()
	}()
	for n := 0; n < len(img); n += 7 {
		Parse(img[:n])
	}
}
