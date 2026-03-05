package machine

import (
	"fmt"
	"zepa-machine/core"
)

const (
	MaxSegmentSizeBits = 12
	MaxSegmentSize     = 1 << MaxSegmentSizeBits
	OffsetMask         = MaxSegmentSize - 1
	SegmentShift       = MaxSegmentSizeBits
	NumSegments        = 4
)

type Privilege int

const (
	KernelPrivilege Privilege = iota
	UserPrivilege
)

type MMUMode int

const (
	ModeFlat MMUMode = iota
	ModeSegmented
)

type AccessType int

const (
	Execute AccessType = 1 << iota
	Write
	Read
)

type Segment struct {
	Base          uint32
	Limit         uint32
	GrowsPositive bool
	Protection    AccessType
	Priv          Privilege
}

type MMU struct {
	Mode       MMUMode
	Segments   [NumSegments]Segment
	tlb        *TLB
	MemorySize uint32
}

func (m *MMU) Translate(virtualAddress uint32, access AccessType, currentPriv Privilege) (uint32, error) {
	var physicalAddress uint32
	var err error = &core.FaultError{
		Code: core.EXC_SEGMENTATION_FAULT,
		Msg:  "SEGMENTATION_FAULT: Falha na tradução",
	}

	if m.Mode == ModeFlat {
		return virtualAddress, nil
	}

	segmentID := virtualAddress >> SegmentShift
	offset := virtualAddress & OffsetMask

	if segmentID >= uint32(len(m.Segments)) {
		return 0, &core.FaultError{
			Code: core.EXC_SEGMENTATION_FAULT,
			Msg:  fmt.Sprintf("SEGMENTATION_FAULT: ID %d inválido", segmentID),
		}
	}

	seg := m.Segments[segmentID]

	if currentPriv > seg.Priv {
		return 0, &core.FaultError{
			Code: core.EXC_PROTECTION_FAULT,
			Msg:  "PROTECTION_FAULT: Privilege violation",
		}
	}

	if seg.Protection&access == 0 {
		err = &core.FaultError{
			Code: core.EXC_PROTECTION_FAULT,
			Msg:  "PROTECTION_FAULT: Forbidden",
		}
		return 0, err
	}

	if seg.GrowsPositive && offset < seg.Limit {
		physicalAddress = seg.Base + offset
		if m.MemorySize > 0 && physicalAddress >= m.MemorySize {
			return 0, &core.FaultError{Code: core.EXC_MEMORY_VIOLATION, Msg: "MEMORY_VIOLATION: SEGMENT ADDRESS OUT OF RANGE"}
		}
		err = nil
	}

	if !seg.GrowsPositive {
		realOffset := int32(offset) - int32(MaxSegmentSize)
		if uint32(-realOffset) <= seg.Limit {
			physicalAddress = uint32(int32(seg.Base) + realOffset)
			if m.MemorySize > 0 && physicalAddress >= m.MemorySize {
				return 0, &core.FaultError{Code: core.EXC_MEMORY_VIOLATION, Msg: "MEMORY_VIOLATION: SEGMENT ADDRESS OUT OF RANGE"}
			}
			err = nil
		}
	}

	return physicalAddress, err
}

func (m *MMU) ensureTLB() {
	if m.tlb == nil {
		m.tlb = NewTLB(DefaultTLBSize)
	}
}

func (m *MMU) persistDirtyBit(va uint32, pageDirectoryBase uint32, memory []byte) {
	pdIndex := ExtractPDIndex(va)
	ptIndex := ExtractPTIndex(va)
	pdePhysAddr := pageDirectoryBase + (pdIndex * 4)

	if pdePhysAddr+3 >= uint32(len(memory)) {
		return
	}
	pdeValue := uint32(memory[pdePhysAddr]) |
		(uint32(memory[pdePhysAddr+1]) << 8) |
		(uint32(memory[pdePhysAddr+2]) << 16) |
		(uint32(memory[pdePhysAddr+3]) << 24)
	pde := DecodePDE(pdeValue)
	if !pde.Present {
		return
	}

	ptePhysAddr := pde.BaseAddress + (ptIndex * 4)
	if ptePhysAddr+3 >= uint32(len(memory)) {
		return
	}
	pteValue := uint32(memory[ptePhysAddr]) |
		(uint32(memory[ptePhysAddr+1]) << 8) |
		(uint32(memory[ptePhysAddr+2]) << 16) |
		(uint32(memory[ptePhysAddr+3]) << 24)
	pte := DecodePTE(pteValue)
	if !pte.Present || pte.Dirty {
		return
	}

	pte.Dirty = true
	encoded := EncodePTE(pte)
	memory[ptePhysAddr] = byte(encoded)
	memory[ptePhysAddr+1] = byte(encoded >> 8)
	memory[ptePhysAddr+2] = byte(encoded >> 16)
	memory[ptePhysAddr+3] = byte(encoded >> 24)
}

func (m *MMU) TranslateWithPaging(va uint32, access AccessType, currentPriv Privilege, pageDirectoryBase uint32, memory []byte, setPageFaultAddress func(uint32)) (uint32, error) {
	m.ensureTLB()

	if entry, found := m.tlb.Lookup(va); found {
		if currentPriv == UserPrivilege && !entry.UserSupervisor {
			return 0, &core.FaultError{Code: core.EXC_PROTECTION_FAULT, Msg: "PROTECTION_FAULT: USER ACCESS TO KERNEL PAGE"}
		}
		if access&Write != 0 && !entry.ReadWrite {
			return 0, &core.FaultError{Code: core.EXC_PROTECTION_FAULT, Msg: "PROTECTION_FAULT: PAGE IS READ-ONLY"}
		}

		if access&Write != 0 && !entry.Dirty {
			m.tlb.SetDirty(va)
			m.persistDirtyBit(va, pageDirectoryBase, memory)
		}
		offset := ExtractPageOffset(va)
		return MakePhysicalAddress(entry.PhysicalFrame, offset), nil
	}

	pte, err := m.PageTableWalk(va, access, currentPriv, pageDirectoryBase, memory, setPageFaultAddress)
	if err != nil {
		return 0, err
	}

	if access&Write != 0 && !pte.ReadWrite {
		return 0, &core.FaultError{Code: core.EXC_PROTECTION_FAULT, Msg: "PROTECTION_FAULT: PAGE IS READ-ONLY"}
	}

	m.tlb.Insert(va, pte)
	offset := ExtractPageOffset(va)

	return MakePhysicalAddress(pte.BaseAddress, offset), nil
}

func (m *MMU) PageTableWalk(va uint32, access AccessType, currentPriv Privilege, pageDirectoryBase uint32, memory []byte, setPageFaultAddress func(uint32)) (PageTableEntry, error) {
	var emptyPTE PageTableEntry

	pdIndex := ExtractPDIndex(va)
	ptIndex := ExtractPTIndex(va)
	pdePhysAddr := pageDirectoryBase + (pdIndex * 4)

	if pdePhysAddr+3 >= uint32(len(memory)) {
		setPageFaultAddress(va)
		return emptyPTE, &core.FaultError{Code: core.EXC_PAGE_FAULT, Msg: "PAGE_FAULT: PAGE DIRECTORY ACCESS OUT OF BOUNDS"}
	}

	pdeValue := uint32(memory[pdePhysAddr]) |
		(uint32(memory[pdePhysAddr+1]) << 8) |
		(uint32(memory[pdePhysAddr+2]) << 16) |
		(uint32(memory[pdePhysAddr+3]) << 24)

	pde := DecodePDE(pdeValue)
	if !pde.Present {
		setPageFaultAddress(va)
		return emptyPTE, &core.FaultError{Code: core.EXC_PAGE_FAULT, Msg: "PAGE_FAULT: PAGE DIRECTORY ENTRY NOT PRESENT"}
	}

	if currentPriv == UserPrivilege && !pde.UserSupervisor {
		return emptyPTE, &core.FaultError{Code: core.EXC_PROTECTION_FAULT, Msg: "PROTECTION_FAULT: USER ACCESS TO KERNEL PAGE DIRECTORY"}
	}

	ptBase := pde.BaseAddress
	ptePhysAddr := ptBase + (ptIndex * 4)

	if ptePhysAddr+3 >= uint32(len(memory)) {
		setPageFaultAddress(va)
		return emptyPTE, &core.FaultError{Code: core.EXC_PAGE_FAULT, Msg: "PAGE_FAULT: PAGE TABLE ACCESS OUT OF BOUNDS"}
	}

	pteValue := uint32(memory[ptePhysAddr]) |
		(uint32(memory[ptePhysAddr+1]) << 8) |
		(uint32(memory[ptePhysAddr+2]) << 16) |
		(uint32(memory[ptePhysAddr+3]) << 24)

	pte := DecodePTE(pteValue)
	if !pte.Present {
		setPageFaultAddress(va)
		return emptyPTE, &core.FaultError{Code: core.EXC_PAGE_FAULT, Msg: "PAGE_FAULT: PAGE TABLE ENTRY NOT PRESENT"}
	}

	if currentPriv == UserPrivilege && !pte.UserSupervisor {
		return emptyPTE, &core.FaultError{Code: core.EXC_PROTECTION_FAULT, Msg: "PROTECTION_FAULT: USER ACCESS TO KERNEL PAGE"}
	}

	if !pde.Accessed {
		pde.Accessed = true
		encodedPDE := EncodePDE(pde)
		memory[pdePhysAddr] = byte(encodedPDE)
		memory[pdePhysAddr+1] = byte(encodedPDE >> 8)
		memory[pdePhysAddr+2] = byte(encodedPDE >> 16)
		memory[pdePhysAddr+3] = byte(encodedPDE >> 24)
	}

	needsUpdate := !pte.Accessed || (access&Write != 0 && !pte.Dirty)
	if needsUpdate {
		pte.Accessed = true
		if access&Write != 0 {
			pte.Dirty = true
		}
		encodedPTE := EncodePTE(pte)
		memory[ptePhysAddr] = byte(encodedPTE)
		memory[ptePhysAddr+1] = byte(encodedPTE >> 8)
		memory[ptePhysAddr+2] = byte(encodedPTE >> 16)
		memory[ptePhysAddr+3] = byte(encodedPTE >> 24)
	}

	return pte, nil
}
