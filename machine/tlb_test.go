package machine

import "testing"

func TestNewTLBDefaultSize(t *testing.T) {
	tlb := NewTLB(0)
	if tlb.size != DefaultTLBSize {
		t.Fatalf("size = %d, want %d", tlb.size, DefaultTLBSize)
	}
}

func TestNewTLBNegativeSize(t *testing.T) {
	tlb := NewTLB(-5)
	if tlb.size != DefaultTLBSize {
		t.Fatalf("size = %d, want %d", tlb.size, DefaultTLBSize)
	}
}

func TestNewTLBCustomSize(t *testing.T) {
	tlb := NewTLB(8)
	if tlb.size != 8 {
		t.Fatalf("size = %d, want 8", tlb.size)
	}
	if tlb.Size() != 0 {
		t.Fatalf("entries = %d, want 0", tlb.Size())
	}
}

func TestNewTLBSizeOne(t *testing.T) {
	tlb := NewTLB(1)
	pte := PageTableEntry{Present: true, BaseAddress: 0x1000}
	tlb.Insert(0x1000, pte)
	if tlb.Size() != 1 {
		t.Fatalf("size = %d, want 1", tlb.Size())
	}
	tlb.Insert(0x2000, pte)
	if tlb.Size() != 1 {
		t.Fatalf("size = %d, want 1 after eviction", tlb.Size())
	}
	if _, found := tlb.Lookup(0x1000); found {
		t.Fatal("first entry should be evicted")
	}
	if _, found := tlb.Lookup(0x2000); !found {
		t.Fatal("second entry should be present")
	}
}

func TestTLBLookupEmptyTLB(t *testing.T) {
	tlb := NewTLB(4)
	if _, found := tlb.Lookup(0x1000); found {
		t.Fatal("expected miss on empty TLB")
	}
}

func TestTLBHitAndMiss(t *testing.T) {
	tlb := NewTLB(4)
	pte := PageTableEntry{
		Present:        true,
		ReadWrite:      true,
		UserSupervisor: false,
		BaseAddress:    0xABCD0000,
	}

	tlb.Insert(0x12345000, pte)
	entry, found := tlb.Lookup(0x12345000)
	if !found {
		t.Fatal("expected TLB hit")
	}
	if entry.PhysicalFrame != 0xABCD0000 {
		t.Fatalf("frame = 0x%X, want 0xABCD0000", entry.PhysicalFrame)
	}
	if entry.ReadWrite != pte.ReadWrite {
		t.Fatal("ReadWrite mismatch")
	}

	if _, found := tlb.Lookup(0x99999000); found {
		t.Fatal("expected miss on different address")
	}
}

func TestTLBInsertUpdate(t *testing.T) {
	tlb := NewTLB(4)
	pte1 := PageTableEntry{Present: true, ReadWrite: true, BaseAddress: 0x20000000}
	pte2 := PageTableEntry{Present: true, ReadWrite: false, BaseAddress: 0x30000000, Dirty: true}

	tlb.Insert(0x10000000, pte1)
	entry, _ := tlb.Lookup(0x10000000)
	if entry.PhysicalFrame != 0x20000000 || !entry.ReadWrite {
		t.Fatal("first insert mismatch")
	}

	tlb.Insert(0x10000000, pte2)
	entry, _ = tlb.Lookup(0x10000000)
	if entry.PhysicalFrame != 0x30000000 {
		t.Fatalf("frame = 0x%X, want 0x30000000", entry.PhysicalFrame)
	}
	if entry.ReadWrite {
		t.Fatal("expected ReadWrite=false after update")
	}
	if !entry.Dirty {
		t.Fatal("expected Dirty=true after update")
	}
	if tlb.Size() != 1 {
		t.Fatalf("size = %d, want 1", tlb.Size())
	}
}

func TestTLBInsertUpdatePreservesFIFOOrder(t *testing.T) {
	tlb := NewTLB(2)
	pte := PageTableEntry{Present: true, BaseAddress: 0x1000}

	tlb.Insert(0x01000000, pte)
	tlb.Insert(0x02000000, pte)

	tlb.Insert(0x01000000, PageTableEntry{Present: true, BaseAddress: 0x9000})

	tlb.Insert(0x03000000, pte)

	if _, found := tlb.Lookup(0x01000000); found {
		t.Fatal("first entry should be evicted (FIFO order preserved on update)")
	}
	if _, found := tlb.Lookup(0x02000000); !found {
		t.Fatal("second entry should remain")
	}
}

func TestTLBFIFOReplacement(t *testing.T) {
	tlb := NewTLB(2)
	pte := PageTableEntry{Present: true, BaseAddress: 0x00000000}

	tlb.Insert(0x01000000, pte)
	tlb.Insert(0x02000000, pte)
	if tlb.Size() != 2 {
		t.Fatalf("size = %d, want 2", tlb.Size())
	}

	tlb.Insert(0x03000000, pte)
	if _, found := tlb.Lookup(0x01000000); found {
		t.Fatal("first entry should be evicted")
	}
	if _, found := tlb.Lookup(0x02000000); !found {
		t.Fatal("second should remain")
	}
	if _, found := tlb.Lookup(0x03000000); !found {
		t.Fatal("third should be present")
	}

	tlb.Insert(0x04000000, pte)
	if _, found := tlb.Lookup(0x02000000); found {
		t.Fatal("second should be evicted")
	}
	if _, found := tlb.Lookup(0x03000000); !found {
		t.Fatal("third should remain")
	}
	if _, found := tlb.Lookup(0x04000000); !found {
		t.Fatal("fourth should be present")
	}
}

func TestTLBFIFOFullCycle(t *testing.T) {
	tlb := NewTLB(3)
	pte := PageTableEntry{Present: true, BaseAddress: 0x1000}
	for i := uint32(0); i < 6; i++ {
		tlb.Insert(i<<12, pte)
	}
	if tlb.Size() != 3 {
		t.Fatalf("size = %d, want 3", tlb.Size())
	}
	for i := uint32(0); i < 3; i++ {
		if _, found := tlb.Lookup(i << 12); found {
			t.Fatalf("entry %d should be evicted", i)
		}
	}
	for i := uint32(3); i < 6; i++ {
		if _, found := tlb.Lookup(i << 12); !found {
			t.Fatalf("entry %d should be present", i)
		}
	}
}

func TestTLBFlush(t *testing.T) {
	tlb := NewTLB(4)
	pte := PageTableEntry{Present: true, BaseAddress: 0x10000000}
	tlb.Insert(0x01000000, pte)
	tlb.Insert(0x02000000, pte)
	tlb.Insert(0x03000000, pte)

	tlb.Flush()
	if tlb.Size() != 0 {
		t.Fatalf("size = %d, want 0", tlb.Size())
	}

	addrs := []uint32{0x01000000, 0x02000000, 0x03000000}
	for _, a := range addrs {
		if _, found := tlb.Lookup(a); found {
			t.Fatalf("expected miss at 0x%X after flush", a)
		}
	}
}

func TestTLBFlushEmpty(t *testing.T) {
	tlb := NewTLB(4)
	tlb.Flush()
	if tlb.Size() != 0 {
		t.Fatalf("size = %d, want 0", tlb.Size())
	}
}

func TestTLBFlushResetsFIFOCounter(t *testing.T) {
	tlb := NewTLB(2)
	pte := PageTableEntry{Present: true, BaseAddress: 0x1000}
	tlb.Insert(0x01000000, pte)
	tlb.Insert(0x02000000, pte)
	tlb.Flush()

	tlb.Insert(0x03000000, pte)
	tlb.Insert(0x04000000, pte)
	tlb.Insert(0x05000000, pte)

	if _, found := tlb.Lookup(0x03000000); found {
		t.Fatal("third should be evicted after FIFO reset")
	}
}

func TestTLBFlushPage(t *testing.T) {
	tlb := NewTLB(4)
	pte := PageTableEntry{Present: true, BaseAddress: 0x10000000}
	tlb.Insert(0x01000000, pte)
	tlb.Insert(0x02000000, pte)
	tlb.Insert(0x03000000, pte)

	tlb.FlushPage(0x02000000)
	if tlb.Size() != 2 {
		t.Fatalf("size = %d, want 2", tlb.Size())
	}
	if _, found := tlb.Lookup(0x02000000); found {
		t.Fatal("expected miss on flushed page")
	}
	if _, found := tlb.Lookup(0x01000000); !found {
		t.Fatal("first should remain")
	}
	if _, found := tlb.Lookup(0x03000000); !found {
		t.Fatal("third should remain")
	}
}

func TestTLBFlushPageFirst(t *testing.T) {
	tlb := NewTLB(4)
	pte := PageTableEntry{Present: true, BaseAddress: 0x1000}
	tlb.Insert(0x01000000, pte)
	tlb.Insert(0x02000000, pte)
	tlb.FlushPage(0x01000000)
	if tlb.Size() != 1 {
		t.Fatalf("size = %d, want 1", tlb.Size())
	}
	if _, found := tlb.Lookup(0x02000000); !found {
		t.Fatal("second should remain")
	}
}

func TestTLBFlushPageLast(t *testing.T) {
	tlb := NewTLB(4)
	pte := PageTableEntry{Present: true, BaseAddress: 0x1000}
	tlb.Insert(0x01000000, pte)
	tlb.Insert(0x02000000, pte)
	tlb.FlushPage(0x02000000)
	if tlb.Size() != 1 {
		t.Fatalf("size = %d, want 1", tlb.Size())
	}
}

func TestTLBFlushPageNotFound(t *testing.T) {
	tlb := NewTLB(4)
	pte := PageTableEntry{Present: true, BaseAddress: 0x1000}
	tlb.Insert(0x01000000, pte)
	tlb.FlushPage(0x99999000)
	if tlb.Size() != 1 {
		t.Fatalf("size = %d, want 1", tlb.Size())
	}
}

func TestTLBFlushPageEmpty(t *testing.T) {
	tlb := NewTLB(4)
	tlb.FlushPage(0x01000000)
	if tlb.Size() != 0 {
		t.Fatalf("size = %d, want 0", tlb.Size())
	}
}

func TestTLBFlushPageWithOffset(t *testing.T) {
	tlb := NewTLB(4)
	pte := PageTableEntry{Present: true, BaseAddress: 0x1000}
	tlb.Insert(0x01000ABC, pte)
	tlb.FlushPage(0x01000123)
	if tlb.Size() != 0 {
		t.Fatalf("size = %d, want 0 (same page different offset)", tlb.Size())
	}
}

func TestTLBSetDirty(t *testing.T) {
	tlb := NewTLB(4)
	pte := PageTableEntry{Present: true, BaseAddress: 0x1000, Dirty: false}
	tlb.Insert(0x01000000, pte)

	tlb.SetDirty(0x01000000)
	entry, _ := tlb.Lookup(0x01000000)
	if !entry.Dirty {
		t.Fatal("expected Dirty=true after SetDirty")
	}
}

func TestTLBSetDirtyNotFound(t *testing.T) {
	tlb := NewTLB(4)
	tlb.SetDirty(0x99999000)
}

func TestTLBSetDirtyAlreadyDirty(t *testing.T) {
	tlb := NewTLB(4)
	pte := PageTableEntry{Present: true, BaseAddress: 0x1000, Dirty: true}
	tlb.Insert(0x01000000, pte)
	tlb.SetDirty(0x01000000)
	entry, _ := tlb.Lookup(0x01000000)
	if !entry.Dirty {
		t.Fatal("should remain dirty")
	}
}

func TestTLBSetDirtyWithOffset(t *testing.T) {
	tlb := NewTLB(4)
	pte := PageTableEntry{Present: true, BaseAddress: 0x1000, Dirty: false}
	tlb.Insert(0x01000000, pte)
	tlb.SetDirty(0x01000ABC)
	entry, _ := tlb.Lookup(0x01000000)
	if !entry.Dirty {
		t.Fatal("SetDirty with offset should set dirty for the page")
	}
}

func TestTLBOffsetHandling(t *testing.T) {
	tlb := NewTLB(4)
	pte := PageTableEntry{Present: true, BaseAddress: 0xABCD0000}
	tlb.Insert(0x12345ABC, pte)

	entry, found := tlb.Lookup(0x12345123)
	if !found {
		t.Fatal("expected hit for same page different offset")
	}
	if entry.PhysicalFrame != 0xABCD0000 {
		t.Fatalf("frame = 0x%X, want 0xABCD0000", entry.PhysicalFrame)
	}

	entry, found = tlb.Lookup(0x12345000)
	if !found {
		t.Fatal("expected hit for page-aligned address")
	}
}

func TestTLBGlobalFlag(t *testing.T) {
	tlb := NewTLB(4)
	tlb.Insert(0x01000000, PageTableEntry{Present: true, BaseAddress: 0x10000000, Global: true})
	tlb.Insert(0x02000000, PageTableEntry{Present: true, BaseAddress: 0x20000000, Global: false})

	entry, _ := tlb.Lookup(0x01000000)
	if !entry.Global {
		t.Fatal("expected Global=true")
	}
	entry, _ = tlb.Lookup(0x02000000)
	if entry.Global {
		t.Fatal("expected Global=false")
	}
}

func TestTLBUserSupervisorFlag(t *testing.T) {
	tlb := NewTLB(4)
	tlb.Insert(0x01000000, PageTableEntry{Present: true, UserSupervisor: true, BaseAddress: 0x1000})
	tlb.Insert(0x02000000, PageTableEntry{Present: true, UserSupervisor: false, BaseAddress: 0x2000})

	e1, _ := tlb.Lookup(0x01000000)
	if !e1.UserSupervisor {
		t.Fatal("expected UserSupervisor=true")
	}
	e2, _ := tlb.Lookup(0x02000000)
	if e2.UserSupervisor {
		t.Fatal("expected UserSupervisor=false")
	}
}

func TestTLBFIFOAfterFlushPage(t *testing.T) {
	tlb := NewTLB(3)
	pte := PageTableEntry{Present: true, BaseAddress: 0x10000000}

	addrA := uint32(0x01000000)
	addrB := uint32(0x02000000)
	addrC := uint32(0x03000000)
	tlb.Insert(addrA, pte)
	tlb.Insert(addrB, pte)
	tlb.Insert(addrC, pte)

	tlb.FlushPage(addrB)

	addrD := uint32(0x04000000)
	tlb.Insert(addrD, pte)

	addrE := uint32(0x05000000)
	tlb.Insert(addrE, pte)

	if _, found := tlb.Lookup(addrA); found {
		t.Fatal("A should be evicted as oldest")
	}
	if _, found := tlb.Lookup(addrC); !found {
		t.Fatal("C should remain")
	}
	if _, found := tlb.Lookup(addrD); !found {
		t.Fatal("D should remain")
	}
	if _, found := tlb.Lookup(addrE); !found {
		t.Fatal("E should be present")
	}
}

func TestTLBNotPresentEntry(t *testing.T) {
	tlb := NewTLB(4)
	pte := PageTableEntry{Present: false, BaseAddress: 0x1000}
	tlb.Insert(0x01000000, pte)

	if _, found := tlb.Lookup(0x01000000); found {
		t.Fatal("non-present entry should not match on lookup")
	}
}

func TestTLBSizeAfterOperations(t *testing.T) {
	tlb := NewTLB(4)
	pte := PageTableEntry{Present: true, BaseAddress: 0x1000}

	if tlb.Size() != 0 {
		t.Fatal("should start empty")
	}
	tlb.Insert(0x01000000, pte)
	if tlb.Size() != 1 {
		t.Fatal("should be 1")
	}
	tlb.Insert(0x02000000, pte)
	if tlb.Size() != 2 {
		t.Fatal("should be 2")
	}
	tlb.FlushPage(0x01000000)
	if tlb.Size() != 1 {
		t.Fatal("should be 1 after flush page")
	}
	tlb.Flush()
	if tlb.Size() != 0 {
		t.Fatal("should be 0 after full flush")
	}
}

func TestTLBMultipleInsertsSameVirtualPage(t *testing.T) {
	tlb := NewTLB(4)
	pte1 := PageTableEntry{Present: true, BaseAddress: 0x1000}
	pte2 := PageTableEntry{Present: true, BaseAddress: 0x2000}
	pte3 := PageTableEntry{Present: true, BaseAddress: 0x3000}

	tlb.Insert(0x01000000, pte1)
	tlb.Insert(0x01000000, pte2)
	tlb.Insert(0x01000000, pte3)

	if tlb.Size() != 1 {
		t.Fatalf("size = %d, want 1", tlb.Size())
	}
	entry, _ := tlb.Lookup(0x01000000)
	if entry.PhysicalFrame != 0x3000 {
		t.Fatalf("frame = 0x%X, want 0x3000", entry.PhysicalFrame)
	}
}
