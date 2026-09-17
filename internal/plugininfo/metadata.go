package plugininfo

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

// Metadata table numbers (ECMA-335 II.22).
const (
	tModule                 = 0x00
	tTypeRef                = 0x01
	tTypeDef                = 0x02
	tFieldPtr               = 0x03
	tField                  = 0x04
	tMethodPtr              = 0x05
	tMethodDef              = 0x06
	tParamPtr               = 0x07
	tParam                  = 0x08
	tInterfaceImpl          = 0x09
	tMemberRef              = 0x0A
	tConstant               = 0x0B
	tCustomAttribute        = 0x0C
	tDeclSecurity           = 0x0E
	tStandAloneSig          = 0x11
	tEvent                  = 0x14
	tProperty               = 0x17
	tModuleRef              = 0x1A
	tTypeSpec               = 0x1B
	tAssembly               = 0x20
	tAssemblyRef            = 0x23
	tFile                   = 0x26
	tExportedType           = 0x27
	tManifestResource       = 0x28
	tGenericParam           = 0x2A
	tMethodSpec             = 0x2B
	tGenericParamConstraint = 0x2C
)

// Coded index definitions: the tables a tag selects, in tag order.
var (
	ciTypeDefOrRef       = []int{tTypeDef, tTypeRef, tTypeSpec}
	ciHasConstant        = []int{tField, tParam, tProperty}
	ciResolutionScope    = []int{tModule, tModuleRef, tAssemblyRef, tTypeRef}
	ciMemberRefParent    = []int{tTypeDef, tTypeRef, tModuleRef, tMethodDef, tTypeSpec}
	ciCustomAttributeTyp = []int{-1, -1, tMethodDef, tMemberRef, -1}
	ciHasCustomAttribute = []int{
		tMethodDef, tField, tTypeRef, tTypeDef, tParam, tInterfaceImpl, tMemberRef, tModule,
		tDeclSecurity, tProperty, tEvent, tStandAloneSig, tModuleRef, tTypeSpec, tAssembly,
		tAssemblyRef, tFile, tExportedType, tManifestResource, tGenericParam,
		tGenericParamConstraint, tMethodSpec,
	}
)

type metadata struct {
	strings, blob []byte
	rows          [64]uint32
	wideString    bool
	wideGUID      bool
	wideBlob      bool
	tables        [64][]byte // raw rows of the tables we read
	rowSize       [64]int
}

func readMetadata(pe *peFile, cliRVA uint32) (*metadata, error) {
	cli, err := pe.at(cliRVA, 72)
	if err != nil {
		return nil, err
	}
	mdRVA := binary.LittleEndian.Uint32(cli[8:])
	mdSize := binary.LittleEndian.Uint32(cli[12:])
	root, err := pe.at(mdRVA, mdSize)
	if err != nil {
		return nil, err
	}
	if len(root) < 20 || binary.LittleEndian.Uint32(root) != 0x424A5342 {
		return nil, ErrNotAssembly
	}
	verLen := int(binary.LittleEndian.Uint32(root[12:]))
	p := 16 + verLen
	if p+4 > len(root) {
		return nil, ErrNotAssembly
	}
	streams := int(binary.LittleEndian.Uint16(root[p+2:]))
	p += 4

	md := &metadata{}
	var tableStream []byte
	for i := 0; i < streams; i++ {
		if p+8 > len(root) {
			return nil, ErrNotAssembly
		}
		off := int(binary.LittleEndian.Uint32(root[p:]))
		size := int(binary.LittleEndian.Uint32(root[p+4:]))
		nameEnd := bytes.IndexByte(root[p+8:], 0)
		if nameEnd < 0 {
			return nil, ErrNotAssembly
		}
		name := string(root[p+8 : p+8+nameEnd])
		p += 8 + (nameEnd+4)&^3 // name is null-terminated and padded to 4 bytes
		if off < 0 || size < 0 || off+size > len(root) {
			return nil, fmt.Errorf("stream %s out of range", name)
		}
		data := root[off : off+size]
		switch name {
		case "#~", "#-":
			tableStream = data
		case "#Strings":
			md.strings = data
		case "#Blob":
			md.blob = data
		}
	}
	if tableStream == nil {
		return nil, ErrNotAssembly
	}
	return md, md.readTables(tableStream)
}

func (md *metadata) readTables(s []byte) error {
	if len(s) < 24 {
		return ErrNotAssembly
	}
	heaps := s[6]
	md.wideString, md.wideGUID, md.wideBlob = heaps&1 != 0, heaps&2 != 0, heaps&4 != 0
	valid := binary.LittleEndian.Uint64(s[8:])
	p := 24
	for t := 0; t < 64; t++ {
		if valid&(1<<t) != 0 {
			if p+4 > len(s) {
				return ErrNotAssembly
			}
			md.rows[t] = binary.LittleEndian.Uint32(s[p:])
			p += 4
		}
	}
	if heaps&0x40 != 0 {
		p += 4 // extra data after row counts in some #- streams
	}

	// Tables are stored back to back in table order; only the ones up to
	// CustomAttribute are needed, so their row sizes are enough to walk there.
	for t := 0; t <= tCustomAttribute; t++ {
		md.rowSize[t] = md.sizeOf(t)
		if md.rowSize[t] < 0 {
			return fmt.Errorf("unsupported table %#x", t)
		}
		n := int(md.rows[t]) * md.rowSize[t]
		if p+n > len(s) {
			return ErrNotAssembly
		}
		md.tables[t] = s[p : p+n]
		p += n
	}
	return nil
}

func (md *metadata) idx(t int) int {
	if md.rows[t] < 1<<16 {
		return 2
	}
	return 4
}

func (md *metadata) coded(tables []int) int {
	bits := 0
	for n := len(tables) - 1; n > 0; n >>= 1 {
		bits++
	}
	var maxRows uint32
	for _, t := range tables {
		if t >= 0 {
			maxRows = max(maxRows, md.rows[t])
		}
	}
	if maxRows < 1<<(16-bits) {
		return 2
	}
	return 4
}

func (md *metadata) heap(wide bool) int {
	if wide {
		return 4
	}
	return 2
}

func (md *metadata) sizeOf(t int) int {
	str, guid, blob := md.heap(md.wideString), md.heap(md.wideGUID), md.heap(md.wideBlob)
	switch t {
	case tModule:
		return 2 + str + 3*guid
	case tTypeRef:
		return md.coded(ciResolutionScope) + 2*str
	case tTypeDef:
		return 4 + 2*str + md.coded(ciTypeDefOrRef) + md.idx(tField) + md.idx(tMethodDef)
	case tFieldPtr:
		return md.idx(tField)
	case tField:
		return 2 + str + blob
	case tMethodPtr:
		return md.idx(tMethodDef)
	case tMethodDef:
		return 4 + 2 + 2 + str + blob + md.idx(tParam)
	case tParamPtr:
		return md.idx(tParam)
	case tParam:
		return 2 + 2 + str
	case tInterfaceImpl:
		return md.idx(tTypeDef) + md.coded(ciTypeDefOrRef)
	case tMemberRef:
		return md.coded(ciMemberRefParent) + str + blob
	case tConstant:
		return 2 + md.coded(ciHasConstant) + blob
	case tCustomAttribute:
		return md.coded(ciHasCustomAttribute) + md.coded(ciCustomAttributeTyp) + blob
	}
	return -1
}

// row returns row i (1-based) of table t split into its columns' raw values.
func (md *metadata) row(t int, i uint32) ([]byte, error) {
	if i == 0 || i > md.rows[t] {
		return nil, fmt.Errorf("row %d of table %#x out of range", i, t)
	}
	size := md.rowSize[t]
	start := int(i-1) * size
	return md.tables[t][start : start+size], nil
}

// cursor reads the columns of one row.
type cursor struct {
	b []byte
	p int
}

func (c *cursor) read(size int) uint32 {
	var v uint32
	if size == 2 {
		v = uint32(binary.LittleEndian.Uint16(c.b[c.p:]))
	} else {
		v = binary.LittleEndian.Uint32(c.b[c.p:])
	}
	c.p += size
	return v
}

func (md *metadata) str(i uint32) string {
	if int(i) >= len(md.strings) {
		return ""
	}
	s := md.strings[i:]
	if end := bytes.IndexByte(s, 0); end >= 0 {
		return string(s[:end])
	}
	return string(s)
}

func (md *metadata) blobAt(i uint32) ([]byte, error) {
	if int(i) >= len(md.blob) {
		return nil, errors.New("blob index out of range")
	}
	n, size, err := compressedUint(md.blob[i:])
	if err != nil {
		return nil, err
	}
	start := int(i) + size
	if start+int(n) > len(md.blob) {
		return nil, errors.New("blob out of range")
	}
	return md.blob[start : start+int(n)], nil
}

// compressedUint decodes an ECMA-335 compressed unsigned integer.
func compressedUint(b []byte) (value uint32, size int, err error) {
	switch {
	case len(b) >= 1 && b[0]&0x80 == 0:
		return uint32(b[0]), 1, nil
	case len(b) >= 2 && b[0]&0xC0 == 0x80:
		return uint32(b[0]&0x3F)<<8 | uint32(b[1]), 2, nil
	case len(b) >= 4 && b[0]&0xE0 == 0xC0:
		return uint32(b[0]&0x1F)<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3]), 4, nil
	}
	return 0, 0, errors.New("invalid compressed integer")
}
