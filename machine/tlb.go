package machine

const (
	DefaultTLBSize = 64
)

type TLBEntry struct {
	VirtualPage   uint32
	PhysicalFrame uint32
	Present       bool
	ReadWrite     bool
	UserSupervisor bool
	Dirty         bool
	Global        bool
}

type TLB struct {
	entries  []TLBEntry
	size     int
	nextSlot int
}

func NewTLB(size int) *TLB {
	if size <= 0 {
		size = DefaultTLBSize
	}
	return &TLB{
		entries:  make([]TLBEntry, 0, size),
		size:     size,
		nextSlot: 0,
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
			tlb.entries[i] = TLBEntry{
				VirtualPage:    virtualPage,
				PhysicalFrame:  pte.BaseAddress,
				Present:        pte.Present,
				ReadWrite:      pte.ReadWrite,
				UserSupervisor: pte.UserSupervisor,
				Dirty:          pte.Dirty,
				Global:         pte.Global,
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
	}

	// If TLB is not full, append
	if len(tlb.entries) < tlb.size {
		tlb.entries = append(tlb.entries, entry)
	} else {
		// Replace using FIFO policy
		tlb.entries[tlb.nextSlot] = entry
		tlb.nextSlot = (tlb.nextSlot + 1) % tlb.size
	}
}

// Flush clears all entries in the TLB.
func (tlb *TLB) Flush() {
	tlb.entries = make([]TLBEntry, 0, tlb.size)
	tlb.nextSlot = 0
}

// FlushPage removes a specific virtual page from the TLB.
func (tlb *TLB) FlushPage(virtualAddr uint32) {
	virtualPage := virtualAddr & FrameAddressMask

	for i := 0; i < len(tlb.entries); i++ {
		if tlb.entries[i].VirtualPage == virtualPage {
			// Remove entry by swapping with last and truncating
			lastIdx := len(tlb.entries) - 1
			tlb.entries[i] = tlb.entries[lastIdx]
			tlb.entries = tlb.entries[:lastIdx]
			
			// Adjust nextSlot if needed
			if tlb.nextSlot > 0 {
				tlb.nextSlot--
			}
			return
		}
	}
}

// Size returns the current number of entries in the TLB.
func (tlb *TLB) Size() int {
	return len(tlb.entries)
}
