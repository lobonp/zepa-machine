package machine

import "fmt"

const (
	PageSize             uint32 = 4096
	PageShift                   = 12
	PageDirectoryEntries uint32 = 1024
	PageTableEntries     uint32 = 1024

	PDIndexShift uint32 = 22
	PTIndexShift uint32 = 12

	PageOffsetMask  uint32 = 0x00000FFF
	PDIndexMask     uint32 = 0x3FF
	PTIndexMask     uint32 = 0x3FF
	FrameAddressMask uint32 = 0xFFFFF000
)

// PageDirectoryEntry defines a single PDE for 32-bit paging.
type PageDirectoryEntry struct {
	Present        bool
	ReadWrite      bool
	UserSupervisor bool
	WriteThrough   bool
	CacheDisable   bool
	Accessed       bool
	Reserved       bool
	PageSize       bool
	Global         bool
	Available      uint8
	BaseAddress    uint32
}

// PageTableEntry defines a single PTE for 32-bit paging.
type PageTableEntry struct {
	Present        bool
	ReadWrite      bool
	UserSupervisor bool
	WriteThrough   bool
	CacheDisable   bool
	Accessed       bool
	Dirty          bool
	PageAttrTable  bool
	Global         bool
	Available      uint8
	BaseAddress    uint32
}

func EncodePDE(pde PageDirectoryEntry) uint32 {
	var entry uint32

	if pde.Present {
		entry |= 1 << 0
	}
	if pde.ReadWrite {
		entry |= 1 << 1
	}
	if pde.UserSupervisor {
		entry |= 1 << 2
	}
	if pde.WriteThrough {
		entry |= 1 << 3
	}
	if pde.CacheDisable {
		entry |= 1 << 4
	}
	if pde.Accessed {
		entry |= 1 << 5
	}
	if pde.Reserved {
		entry |= 1 << 6
	}
	if pde.PageSize {
		entry |= 1 << 7
	}
	if pde.Global {
		entry |= 1 << 8
	}

	entry |= uint32(pde.Available&0x7) << 9
	entry |= pde.BaseAddress & FrameAddressMask

	return entry
}

func DecodePDE(entry uint32) PageDirectoryEntry {
	return PageDirectoryEntry{
		Present:        (entry & (1 << 0)) != 0,
		ReadWrite:      (entry & (1 << 1)) != 0,
		UserSupervisor: (entry & (1 << 2)) != 0,
		WriteThrough:   (entry & (1 << 3)) != 0,
		CacheDisable:   (entry & (1 << 4)) != 0,
		Accessed:       (entry & (1 << 5)) != 0,
		Reserved:       (entry & (1 << 6)) != 0,
		PageSize:       (entry & (1 << 7)) != 0,
		Global:         (entry & (1 << 8)) != 0,
		Available:      uint8((entry >> 9) & 0x7),
		BaseAddress:    entry & FrameAddressMask,
	}
}

func EncodePTE(pte PageTableEntry) uint32 {
	var entry uint32

	if pte.Present {
		entry |= 1 << 0
	}
	if pte.ReadWrite {
		entry |= 1 << 1
	}
	if pte.UserSupervisor {
		entry |= 1 << 2
	}
	if pte.WriteThrough {
		entry |= 1 << 3
	}
	if pte.CacheDisable {
		entry |= 1 << 4
	}
	if pte.Accessed {
		entry |= 1 << 5
	}
	if pte.Dirty {
		entry |= 1 << 6
	}
	if pte.PageAttrTable {
		entry |= 1 << 7
	}
	if pte.Global {
		entry |= 1 << 8
	}

	entry |= uint32(pte.Available&0x7) << 9
	entry |= pte.BaseAddress & FrameAddressMask

	return entry
}

func DecodePTE(entry uint32) PageTableEntry {
	return PageTableEntry{
		Present:        (entry & (1 << 0)) != 0,
		ReadWrite:      (entry & (1 << 1)) != 0,
		UserSupervisor: (entry & (1 << 2)) != 0,
		WriteThrough:   (entry & (1 << 3)) != 0,
		CacheDisable:   (entry & (1 << 4)) != 0,
		Accessed:       (entry & (1 << 5)) != 0,
		Dirty:          (entry & (1 << 6)) != 0,
		PageAttrTable:  (entry & (1 << 7)) != 0,
		Global:         (entry & (1 << 8)) != 0,
		Available:      uint8((entry >> 9) & 0x7),
		BaseAddress:    entry & FrameAddressMask,
	}
}

func ExtractPDIndex(virtualAddr uint32) uint32 {
	return (virtualAddr >> PDIndexShift) & PDIndexMask
}

func ExtractPTIndex(virtualAddr uint32) uint32 {
	return (virtualAddr >> PTIndexShift) & PTIndexMask
}

func ExtractPageOffset(virtualAddr uint32) uint32 {
	return virtualAddr & PageOffsetMask
}

func MakePhysicalAddress(frameAddr uint32, offset uint32) uint32 {
	return (frameAddr & FrameAddressMask) | (offset & PageOffsetMask)
}

func IsPageAligned(addr uint32) bool {
	return (addr & PageOffsetMask) == 0
}

// PageFault represents a page fault exception.
type PageFault struct {
	VirtualAddress uint32
	Reason         string
	IsWrite        bool
}

func (pf *PageFault) Error() string {
	accessType := "read"
	if pf.IsWrite {
		accessType = "write"
	}
	return pf.Reason + " (" + accessType + " access at 0x" + fmt.Sprintf("%X", pf.VirtualAddress) + ")"
}
