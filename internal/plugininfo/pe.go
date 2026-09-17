// Package plugininfo reads BepInEx plugin attributes ([BepInPlugin],
// [BepInDependency], ...) from .NET assemblies without loading them.
//
// It implements just enough of ECMA-335 (PE file, CLI header, metadata
// streams and tables) to find custom attributes whose constructors are
// BepInEx types and decode their arguments.
package plugininfo

import (
	"encoding/binary"
	"errors"
	"fmt"
)

var ErrNotAssembly = errors.New("not a .NET assembly")

type section struct {
	virtualAddress, virtualSize, rawOffset, rawSize uint32
}

// peFile maps RVAs of a PE image to its bytes.
type peFile struct {
	data     []byte
	sections []section
}

func parsePE(data []byte) (*peFile, uint32, error) {
	if len(data) < 0x40 || data[0] != 'M' || data[1] != 'Z' {
		return nil, 0, ErrNotAssembly
	}
	peOff := int(binary.LittleEndian.Uint32(data[0x3c:]))
	if peOff < 0 || peOff+24 > len(data) || string(data[peOff:peOff+4]) != "PE\x00\x00" {
		return nil, 0, ErrNotAssembly
	}
	coff := peOff + 4
	numSections := int(binary.LittleEndian.Uint16(data[coff+2:]))
	optSize := int(binary.LittleEndian.Uint16(data[coff+16:]))
	opt := coff + 20
	if opt+optSize > len(data) || optSize < 2 {
		return nil, 0, ErrNotAssembly
	}
	var dirs int
	switch binary.LittleEndian.Uint16(data[opt:]) {
	case 0x10b: // PE32
		dirs = opt + 96
	case 0x20b: // PE32+
		dirs = opt + 112
	default:
		return nil, 0, ErrNotAssembly
	}
	// Data directory 14 is the CLI header.
	cliDir := dirs + 14*8
	if cliDir+8 > opt+optSize {
		return nil, 0, ErrNotAssembly
	}
	cliRVA := binary.LittleEndian.Uint32(data[cliDir:])
	if cliRVA == 0 {
		return nil, 0, ErrNotAssembly
	}

	pe := &peFile{data: data}
	sec := opt + optSize
	for i := 0; i < numSections; i++ {
		s := sec + i*40
		if s+40 > len(data) {
			return nil, 0, ErrNotAssembly
		}
		pe.sections = append(pe.sections, section{
			virtualSize:    binary.LittleEndian.Uint32(data[s+8:]),
			virtualAddress: binary.LittleEndian.Uint32(data[s+12:]),
			rawSize:        binary.LittleEndian.Uint32(data[s+16:]),
			rawOffset:      binary.LittleEndian.Uint32(data[s+20:]),
		})
	}
	return pe, cliRVA, nil
}

// at returns the bytes at an RVA, up to size.
func (pe *peFile) at(rva, size uint32) ([]byte, error) {
	for _, s := range pe.sections {
		span := max(s.virtualSize, s.rawSize)
		if rva >= s.virtualAddress && rva < s.virtualAddress+span {
			off := uint64(s.rawOffset) + uint64(rva-s.virtualAddress)
			end := off + uint64(size)
			if end > uint64(len(pe.data)) {
				break
			}
			return pe.data[off:end], nil
		}
	}
	return nil, fmt.Errorf("RVA %#x outside the image", rva)
}
