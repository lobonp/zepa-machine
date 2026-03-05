package machine

import "testing"

func TestTLBHitAndMiss(t *testing.T) {
	tlb := NewTLB(4)

	virtualAddr := uint32(0x12345000)
	pte := PageTableEntry{
		Present:        true,
		ReadWrite:      true,
		UserSupervisor: false,
		BaseAddress:    0xABCD0000,
		Dirty:          false,
		Global:         false,
	}

	if _, found := tlb.Lookup(virtualAddr); found {
		t.Fatal("expected TLB miss on empty TLB")
	}

	tlb.Insert(virtualAddr, pte)
	entry, found := tlb.Lookup(virtualAddr)
	if !found {
		t.Fatal("expected TLB hit after insert")
	}
	if entry.PhysicalFrame != pte.BaseAddress {
		t.Fatalf("got frame 0x%X, want 0x%X", entry.PhysicalFrame, pte.BaseAddress)
	}
	if entry.ReadWrite != pte.ReadWrite {
		t.Fatal("ReadWrite flag mismatch")
	}

	if _, found := tlb.Lookup(0x99999000); found {
		t.Fatal("expected TLB miss on different address")
	}
}

func TestTLBInsertUpdate(t *testing.T) {
	tlb := NewTLB(4)

	virtualAddr := uint32(0x10000000)
	pte1 := PageTableEntry{
		Present:     true,
		ReadWrite:   true,
		BaseAddress: 0x20000000,
	}
	pte2 := PageTableEntry{
		Present:     true,
		ReadWrite:   false,
		BaseAddress: 0x30000000,
		Dirty:       true,
	}

	// Insert first mapping
	tlb.Insert(virtualAddr, pte1)
	entry, _ := tlb.Lookup(virtualAddr)
	if entry.PhysicalFrame != 0x20000000 {
		t.Fatalf("got frame 0x%X, want 0x20000000", entry.PhysicalFrame)
	}
	if !entry.ReadWrite {
		t.Fatal("expected ReadWrite=true")
	}

	// Update same virtual address
	tlb.Insert(virtualAddr, pte2)
	entry, _ = tlb.Lookup(virtualAddr)
	if entry.PhysicalFrame != 0x30000000 {
		t.Fatalf("got frame 0x%X, want 0x30000000", entry.PhysicalFrame)
	}
	if entry.ReadWrite {
		t.Fatal("expected ReadWrite=false after update")
	}
	if !entry.Dirty {
		t.Fatal("expected Dirty=true after update")
	}

	// Ensure TLB size didn't grow from update
	if tlb.Size() != 1 {
		t.Fatalf("expected 1 entry, got %d", tlb.Size())
	}
}

func TestTLBFIFOReplacement(t *testing.T) {
	tlb := NewTLB(2)

	pte := PageTableEntry{
		Present:     true,
		BaseAddress: 0x00000000,
	}

	// Fill TLB
	tlb.Insert(0x01000000, pte)
	tlb.Insert(0x02000000, pte)

	if tlb.Size() != 2 {
		t.Fatalf("expected size 2, got %d", tlb.Size())
	}

	// Verify both are present
	if _, found := tlb.Lookup(0x01000000); !found {
		t.Fatal("expected first entry in TLB")
	}
	if _, found := tlb.Lookup(0x02000000); !found {
		t.Fatal("expected second entry in TLB")
	}

	// Insert third entry, should evict first (FIFO)
	tlb.Insert(0x03000000, pte)

	if tlb.Size() != 2 {
		t.Fatalf("expected size 2 after replacement, got %d", tlb.Size())
	}

	// First should be evicted
	if _, found := tlb.Lookup(0x01000000); found {
		t.Fatal("expected first entry to be evicted")
	}

	// Second and third should remain
	if _, found := tlb.Lookup(0x02000000); !found {
		t.Fatal("expected second entry to remain")
	}
	if _, found := tlb.Lookup(0x03000000); !found {
		t.Fatal("expected third entry to be present")
	}

	// Insert fourth, should evict second
	tlb.Insert(0x04000000, pte)
	if _, found := tlb.Lookup(0x02000000); found {
		t.Fatal("expected second entry to be evicted")
	}
	if _, found := tlb.Lookup(0x03000000); !found {
		t.Fatal("expected third entry to remain")
	}
	if _, found := tlb.Lookup(0x04000000); !found {
		t.Fatal("expected fourth entry to be present")
	}
}

func TestTLBFlush(t *testing.T) {
	tlb := NewTLB(4)

	pte := PageTableEntry{
		Present:     true,
		BaseAddress: 0x10000000,
	}

	// Insert multiple entries
	tlb.Insert(0x01000000, pte)
	tlb.Insert(0x02000000, pte)
	tlb.Insert(0x03000000, pte)

	if tlb.Size() != 3 {
		t.Fatalf("expected 3 entries, got %d", tlb.Size())
	}

	// Flush all
	tlb.Flush()

	if tlb.Size() != 0 {
		t.Fatalf("expected 0 entries after flush, got %d", tlb.Size())
	}

	// Verify all entries are gone
	if _, found := tlb.Lookup(0x01000000); found {
		t.Fatal("expected miss after flush")
	}
	if _, found := tlb.Lookup(0x02000000); found {
		t.Fatal("expected miss after flush")
	}
	if _, found := tlb.Lookup(0x03000000); found {
		t.Fatal("expected miss after flush")
	}
}

func TestTLBFlushPage(t *testing.T) {
	tlb := NewTLB(4)

	pte := PageTableEntry{
		Present:     true,
		BaseAddress: 0x10000000,
	}

	// Insert three entries
	tlb.Insert(0x01000000, pte)
	tlb.Insert(0x02000000, pte)
	tlb.Insert(0x03000000, pte)

	if tlb.Size() != 3 {
		t.Fatalf("expected 3 entries, got %d", tlb.Size())
	}

	// Flush middle entry
	tlb.FlushPage(0x02000000)

	if tlb.Size() != 2 {
		t.Fatalf("expected 2 entries after FlushPage, got %d", tlb.Size())
	}

	// Verify specific entry is gone
	if _, found := tlb.Lookup(0x02000000); found {
		t.Fatal("expected miss on flushed page")
	}

	// Verify others remain
	if _, found := tlb.Lookup(0x01000000); !found {
		t.Fatal("expected first entry to remain")
	}
	if _, found := tlb.Lookup(0x03000000); !found {
		t.Fatal("expected third entry to remain")
	}

	// Flush non-existent page should not crash
	tlb.FlushPage(0x99999000)
	if tlb.Size() != 2 {
		t.Fatalf("expected size unchanged, got %d", tlb.Size())
	}
}

func TestTLBOffsetHandling(t *testing.T) {
	tlb := NewTLB(4)

	pte := PageTableEntry{
		Present:     true,
		BaseAddress: 0xABCD0000,
	}

	// Insert with non-zero offset (should be masked)
	virtualAddr := uint32(0x12345ABC)
	tlb.Insert(virtualAddr, pte)

	// Lookup with different offset, same page
	entry, found := tlb.Lookup(0x12345123)
	if !found {
		t.Fatal("expected TLB hit for same page with different offset")
	}
	if entry.PhysicalFrame != 0xABCD0000 {
		t.Fatalf("got frame 0x%X, want 0xABCD0000", entry.PhysicalFrame)
	}

	// Lookup with page-aligned address
	entry, found = tlb.Lookup(0x12345000)
	if !found {
		t.Fatal("expected TLB hit for page-aligned address")
	}
	if entry.PhysicalFrame != 0xABCD0000 {
		t.Fatalf("got frame 0x%X, want 0xABCD0000", entry.PhysicalFrame)
	}
}

func TestTLBGlobalFlag(t *testing.T) {
	tlb := NewTLB(4)

	pteGlobal := PageTableEntry{
		Present:     true,
		BaseAddress: 0x10000000,
		Global:      true,
	}
	pteLocal := PageTableEntry{
		Present:     true,
		BaseAddress: 0x20000000,
		Global:      false,
	}

	tlb.Insert(0x01000000, pteGlobal)
	tlb.Insert(0x02000000, pteLocal)

	entry, found := tlb.Lookup(0x01000000)
	if !found || !entry.Global {
		t.Fatal("expected global entry with Global=true")
	}

	entry, found = tlb.Lookup(0x02000000)
	if !found || entry.Global {
		t.Fatal("expected local entry with Global=false")
	}
}

func TestTLBFIFOAfterFlushPage(t *testing.T) {
	tlb := NewTLB(3)

	pte := PageTableEntry{Present: true, BaseAddress: 0x10000000}

	// Initial FIFO order: A, B, C
	addrA := uint32(0x01000000)
	addrB := uint32(0x02000000)
	addrC := uint32(0x03000000)
	tlb.Insert(addrA, pte)
	tlb.Insert(addrB, pte)
	tlb.Insert(addrC, pte)

	// Remove middle entry (B). Remaining oldest should still be A.
	tlb.FlushPage(addrB)

	// Insert D. TLB is not full yet.
	addrD := uint32(0x04000000)
	tlb.Insert(addrD, pte)

	// Insert E. TLB is full; FIFO must evict A (oldest remaining).
	addrE := uint32(0x05000000)
	tlb.Insert(addrE, pte)

	if _, found := tlb.Lookup(addrA); found {
		t.Fatal("expected A to be evicted as oldest entry")
	}
	if _, found := tlb.Lookup(addrC); !found {
		t.Fatal("expected C to remain")
	}
	if _, found := tlb.Lookup(addrD); !found {
		t.Fatal("expected D to remain")
	}
	if _, found := tlb.Lookup(addrE); !found {
		t.Fatal("expected E to be present")
	}
}
