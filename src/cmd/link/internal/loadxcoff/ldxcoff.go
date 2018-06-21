// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package loadxcoff implements a XCOFF file reader.
package loadxcoff

import (
	"cmd/internal/bio"
	"cmd/internal/objabi"
	"cmd/internal/sys"
	"cmd/link/internal/sym"
	"debug/xcoff"
	"errors"
	"fmt"
)

// Xcoff section with its symbol
type ldSection struct {
	xcoff.Section
	sym *sym.Symbol
}

// TODO(brainman): maybe just add ReadAt method to bio.Reader instead of creating xcoffBiobuf

// xcoffBiobuf makes bio.Reader look like io.ReaderAt.
type xcoffBiobuf bio.Reader

func (f *xcoffBiobuf) ReadAt(p []byte, off int64) (int, error) {
	ret := ((*bio.Reader)(f)).Seek(off, 0)
	if ret < 0 {
		return 0, errors.New("fail to seek")
	}
	n, err := f.Read(p)
	if err != nil {
		return 0, err
	}
	return n, nil
}

// Load loads the Xcoff file pn from f.
// Symbols are written into syms, and a slice of the text symbols is returned.
func Load(arch *sys.Arch, syms *sym.Symbols, input *bio.Reader, pkg string, length int64, pn string) (textp []*sym.Symbol, err error) {
	errorf := func(str string, args ...interface{}) ([]*sym.Symbol, error) {
		return nil, fmt.Errorf("loadxcoff: %v: %v", pn, fmt.Sprintf(str, args...))
	}
	localSymVersion := syms.IncVersion()

	var ldSections []*ldSection

	f, err := xcoff.NewFile((*xcoffBiobuf)(input))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	for _, sect := range f.Sections {
		//only text, data and bss section
		if sect.Type < xcoff.STYP_TEXT || sect.Type > xcoff.STYP_BSS {
			continue
		}
		lds := new(ldSection)
		lds.Section = *sect
		name := fmt.Sprintf("%s(%s)", pkg, lds.Name)
		s := syms.Lookup(name, localSymVersion)

		switch lds.Type {
		default:
			errorf("Unrecognize section type 0x%x", lds.Type)
		case xcoff.STYP_TEXT:
			s.Type = sym.STEXT
		case xcoff.STYP_DATA:
			s.Type = sym.SNOPTRDATA
		case xcoff.STYP_BSS:
			s.Type = sym.SNOPTRBSS
		}

		s.Size = int64(lds.Size)
		if s.Type != sym.SNOPTRBSS {
			data, err := lds.Section.Data()
			if err != nil {
				return nil, err
			}
			s.P = data
		}

		lds.sym = s
		ldSections = append(ldSections, lds)
	}

	// sx = symbol from file
	// s = symbol for syms
	for _, sx := range f.Symbols {
		// get symbol type
		stype, err := getSymbolType(f, sx)
		if err != "" {
			errorf("Error reading symbol %s: %s", sx.Name, err)
		}
		if stype == sym.Sxxx {
			continue
		}

		s := syms.Lookup(sx.Name, 0)

		// Text symbol
		if s.Type == sym.STEXT {
			if s.Attr.OnList() {
				errorf("symbol %s listed multiple times", s.Name)
			}
			s.Attr |= sym.AttrOnList
			textp = append(textp, s)
			// for s = s.Sub; s != nil; s = s.Sub {
			// 	if s.Attr.OnList() {
			// 		return nil, nil, fmt.Errorf("symbol %s listed multiple times", s.Name)
			// 	}
			// 	s.Attr |= sym.AttrOnList
			// 	textp = append(textp, s)
			// }
		}
	}

	// Read relocation
	for _, sect := range ldSections {
		// TODO(aix): Dwarf section relocation if needed
		if sect.Type != xcoff.STYP_TEXT && sect.Type != xcoff.STYP_DATA {
			continue
		}
		rs := make([]sym.Reloc, sect.Nreloc)
		for i, rx := range sect.Relocs {
			r := &rs[i]

			r.Sym = syms.Lookup(rx.Symbol.Name, 0)
			if uint64(int32(rx.VirtualAddress)) != rx.VirtualAddress {
				errorf("Virtual Address of a relocation is too big: 0x%x", rx.VirtualAddress)
			}
			r.Off = int32(rx.VirtualAddress)
			switch rx.Type {
			default:
				errorf("%s: section %s: unknown relocation of type 0x%x", pn, sect.Name, rx.Type)
			case xcoff.R_POS:
				// Reloc the address of r.Sym
				// Length should be 64
				if rx.Length != 64 {
					errorf("%s: section %s: relocation R_POS has length different from 64: %d", pn, sect.Name, rx.Length)
				}
				r.Siz = 8
				r.Type = objabi.R_CONST
				r.Add = int64(rx.Symbol.Value)

			case xcoff.R_RBR:
				r.Siz = 4
				r.Type = objabi.R_CALLPOWER
				r.Add = 0 //

			}
		}
		s := sect.sym
		s.R = rs
		s.R = s.R[:sect.Nreloc]
	}
	return textp, nil

}

// Convert symbol xcoff type to sym.SymKind
// Returns nil if this shouldn't be added into syms (like .file or .dw symbols )
func getSymbolType(f *xcoff.File, s *xcoff.Symbol) (stype sym.SymKind, err string) {
	// .file symbol
	if s.SectionNumber == -2 {
		if s.StorageClass == xcoff.C_FILE {
			return sym.Sxxx, ""
		}
		return sym.Sxxx, "Unrecognised StorageClass for sectionNumber = -2"
	}

	// extern symbols
	// TODO(aix)
	if s.SectionNumber == 0 {
		return sym.Sxxx, ""
	}

	sectType := f.Sections[s.SectionNumber-1].SectionHeader.Type
	switch sectType {
	default:
		return sym.Sxxx, fmt.Sprintf("getSymbolType for Section type 0x%x not implemented", sectType)
	case xcoff.STYP_DWARF, xcoff.STYP_DEBUG:
		return sym.Sxxx, ""
	case xcoff.STYP_DATA, xcoff.STYP_BSS, xcoff.STYP_TEXT:
	}

	switch s.StorageClass {
	default:
		return sym.Sxxx, fmt.Sprintf("getSymbolType for Storage class 0x%x not implemented", s.StorageClass)
	case xcoff.C_HIDEXT, xcoff.C_EXT, xcoff.C_WEAKEXT:
		switch s.AuxCSect.StorageMappingClass {
		default:
			return sym.Sxxx, fmt.Sprintf("getSymbolType for Storage class 0x%x and Storage Map 0x%x not implemented", s.StorageClass, s.AuxCSect.StorageMappingClass)

		// Program Code
		case xcoff.XMC_PR:
			if sectType == xcoff.STYP_TEXT {
				return sym.STEXT, ""
			}
			return sym.Sxxx, fmt.Sprintf("Unrecognised Section Type 0x%x for Storage Class 0x%x with Storage Map XMC_PR", sectType, s.StorageClass)

		// Read/Write Data
		case xcoff.XMC_RW:
			if sectType == xcoff.STYP_DATA {
				return sym.SDATA, ""
			}
			if sectType == xcoff.STYP_BSS {
				return sym.SDATA, ""
			}
			return sym.Sxxx, fmt.Sprintf("Unrecognised Section Type 0x%x for Storage Class 0x%x with Storage Map XMC_RW", sectType, s.StorageClass)

		//Function descriptor
		case xcoff.XMC_DS:
			if sectType == xcoff.STYP_DATA {
				return sym.SDATA, ""
			}
			return sym.Sxxx, fmt.Sprintf("Unrecognised Section Type 0x%x for Storage Class 0x%x with Storage Map XMC_DS", sectType, s.StorageClass)

		// TOC anchor and TOC entry
		case xcoff.XMC_TC0, xcoff.XMC_TE:
			if sectType == xcoff.STYP_DATA {
				return sym.SXCOFFTOC, ""
			}
			return sym.Sxxx, fmt.Sprintf("Unrecognised Section Type 0x%x for Storage Class 0x%x with Storage Map XMC_DS", sectType, s.StorageClass)

		}
	}

	panic("unreachable")
	return sym.Sxxx, ""

}
