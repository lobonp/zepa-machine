package machine

const (
	DefaultTLBSize = 64
)

type TLBEntry struct {
	VirtualPage    uint32
	PhysicalFrame  uint32
	Present        bool
	ReadWrite      bool
	UserSupervisor bool
	Dirty          bool
	Global         bool
	fifoOrder      uint64
}

type TLB struct {
	entries    []TLBEntry
	size       int
	nextInsert uint64
}

func NewTLB(size int) *TLB {
	if size <= 0 {
		size = DefaultTLBSize
	}
	return &TLB{
		entries:    make([]TLBEntry, 0, size),
		size:       size,
		nextInsert: 0,
	}
}

func (tlb *TLB) Lookup(virtualAddr uint32) (TLBEntry, bool) {
	virtualPage := virtualAddr & FrameAddressMask

	for i := range tlb.entries {
		if tlb.entries[i].VirtualPage == virtualPage && tlb.entries[i].Present {
			return tlb.entries[i], true
		}
	}

	return TLBEntry{}, false
}

func (tlb *TLB) Insert(virtualAddr uint32, pte PageTableEntry) {
	virtualPage := virtualAddr & FrameAddressMask

	for i := range tlb.entries {
		if tlb.entries[i].VirtualPage == virtualPage {
			fifoOrder := tlb.entries[i].fifoOrder
			tlb.entries[i] = TLBEntry{
				VirtualPage:    virtualPage,
				PhysicalFrame:  pte.BaseAddress,
				Present:        pte.Present,
				ReadWrite:      pte.ReadWrite,
				UserSupervisor: pte.UserSupervisor,
				Dirty:          pte.Dirty,
				Global:         pte.Global,
				fifoOrder:      fifoOrder,
			}
			return
		}
	}

	entry := TLBEntry{
		VirtualPage:    virtualPage,
		PhysicalFrame:  pte.BaseAddress,
		Present:        pte.Present,
		ReadWrite:      pte.ReadWrite,
		UserSupervisor: pte.UserSupervisor,
		Dirty:          pte.Dirty,
		Global:         pte.Global,
		fifoOrder:      tlb.nextInsert,
	}
	tlb.nextInsert++

	if len(tlb.entries) < tlb.size {
		tlb.entries = append(tlb.entries, entry)
	} else {
		// FIFO
		oldestIdx := 0
		for i := 1; i < len(tlb.entries); i++ {
			if tlb.entries[i].fifoOrder < tlb.entries[oldestIdx].fifoOrder {
				oldestIdx = i
			}
		}
		tlb.entries[oldestIdx] = entry
	}
}

func (tlb *TLB) Flush() {
	tlb.entries = make([]TLBEntry, 0, tlb.size)
	tlb.nextInsert = 0
}

func (tlb *TLB) FlushPage(virtualAddr uint32) {
	virtualPage := virtualAddr & FrameAddressMask

	for i := 0; i < len(tlb.entries); i++ {
		if tlb.entries[i].VirtualPage == virtualPage {
			copy(tlb.entries[i:], tlb.entries[i+1:])
			tlb.entries = tlb.entries[:len(tlb.entries)-1]
			return
		}
	}
}

func (tlb *TLB) SetDirty(virtualAddr uint32) {
	virtualPage := virtualAddr & FrameAddressMask
	for i := range tlb.entries {
		if tlb.entries[i].VirtualPage == virtualPage {
			tlb.entries[i].Dirty = true
			return
		}
	}
}

func (tlb *TLB) Size() int {
	return len(tlb.entries)
}
