package machine

import (
	"testing"
	"zepa-machine/core"
)

func TestPDERoundTrip(t *testing.T) {
	in := PageDirectoryEntry{
		Present:        true,
		ReadWrite:      true,
		UserSupervisor: true,
		WriteThrough:   false,
		CacheDisable:   true,
		Accessed:       true,
		Reserved:       false,
		PageSize:       false,
		Global:         true,
		Available:      0x7,
		BaseAddress:    0x12345000,
	}

	encoded := EncodePDE(in)
	out := DecodePDE(encoded)

	if out != in {
		t.Fatalf("PDE round trip failed: got %+v want %+v", out, in)
	}
}

func TestPTERoundTrip(t *testing.T) {
	in := PageTableEntry{
		Present:        true,
		ReadWrite:      true,
		UserSupervisor: false,
		WriteThrough:   true,
		CacheDisable:   false,
		Accessed:       true,
		Dirty:          true,
		PageAttrTable:  false,
		Global:         true,
		Available:      0x5,
		BaseAddress:    0xABCD0000,
	}

	encoded := EncodePTE(in)
	out := DecodePTE(encoded)

	if out != in {
		t.Fatalf("PTE round trip failed: got %+v want %+v", out, in)
	}
}

func TestAddressExtraction(t *testing.T) {
	var va uint32 = 0xCAFEBABE

	if got, want := ExtractPDIndex(va), uint32(0x32B); got != want {
		t.Fatalf("ExtractPDIndex got %d want %d", got, want)
	}
	if got, want := ExtractPTIndex(va), uint32(0x3EB); got != want {
		t.Fatalf("ExtractPTIndex got %d want %d", got, want)
	}
	if got, want := ExtractPageOffset(va), uint32(0xABE); got != want {
		t.Fatalf("ExtractPageOffset got %d want %d", got, want)
	}
}

func TestPhysicalAddressAndAlignment(t *testing.T) {
	frame := uint32(0x12345000)
	offset := uint32(0x678)
	if got, want := MakePhysicalAddress(frame, offset), uint32(0x12345678); got != want {
		t.Fatalf("MakePhysicalAddress got 0x%X want 0x%X", got, want)
	}

	if !IsPageAligned(0x2000) {
		t.Fatalf("expected aligned address")
	}
	if IsPageAligned(0x2001) {
		t.Fatalf("expected non-aligned address")
	}
}

func TestPageTableWalkSuccess(t *testing.T) {
	m := NewMachine(65536)

	// Setup: Create valid PD and PT in memory
	pdBase := uint32(0x1000)
	ptBase := uint32(0x2000)
	pageFrame := uint32(0x3000)

	// Set CR3 to page directory base
	m.SetPageDirectoryBase(pdBase)

	// Create PDE pointing to PT
	pde := PageDirectoryEntry{
		Present:     true,
		ReadWrite:   true,
		BaseAddress: ptBase,
	}
	pdeValue := EncodePDE(pde)

	// Write PDE at index 0 (corresponds to VA 0x00000xxx)
	m.memory[pdBase] = byte(pdeValue)
	m.memory[pdBase+1] = byte(pdeValue >> 8)
	m.memory[pdBase+2] = byte(pdeValue >> 16)
	m.memory[pdBase+3] = byte(pdeValue >> 24)

	// Create PTE pointing to physical page
	pte := PageTableEntry{
		Present:     true,
		ReadWrite:   true,
		BaseAddress: pageFrame,
	}
	pteValue := EncodePTE(pte)

	// Write PTE at index 0 (corresponds to VA 0x00000000)
	m.memory[ptBase] = byte(pteValue)
	m.memory[ptBase+1] = byte(pteValue >> 8)
	m.memory[ptBase+2] = byte(pteValue >> 16)
	m.memory[ptBase+3] = byte(pteValue >> 24)

	// Perform page table walk
	va := uint32(0x00000000)
	returnedPTE, err := m.PageTableWalk(va)

	if err != nil {
		t.Fatalf("expected successful page table walk, got error: %v", err)
	}

	if returnedPTE.BaseAddress != pageFrame {
		t.Fatalf("got frame 0x%X, want 0x%X", returnedPTE.BaseAddress, pageFrame)
	}

	if !returnedPTE.Present {
		t.Fatal("expected PTE to be present")
	}
}

func TestPageTableWalkPDENotPresent(t *testing.T) {
	m := NewMachine(65536)

	pdBase := uint32(0x1000)
	m.SetPageDirectoryBase(pdBase)

	// Create non-present PDE
	pde := PageDirectoryEntry{
		Present: false,
	}
	pdeValue := EncodePDE(pde)

	// Write PDE at index 0
	m.memory[pdBase] = byte(pdeValue)
	m.memory[pdBase+1] = byte(pdeValue >> 8)
	m.memory[pdBase+2] = byte(pdeValue >> 16)
	m.memory[pdBase+3] = byte(pdeValue >> 24)

	va := uint32(0x00000000)
	_, err := m.PageTableWalk(va)

	if err == nil {
		t.Fatal("expected page fault for non-present PDE")
	}

	// Check that CR2 was updated
	if m.GetPageFaultAddress() != va {
		t.Fatalf("expected CR2=0x%X, got 0x%X", va, m.GetPageFaultAddress())
	}
}

func TestPageTableWalkPTENotPresent(t *testing.T) {
	m := NewMachine(65536)

	pdBase := uint32(0x1000)
	ptBase := uint32(0x2000)

	m.SetPageDirectoryBase(pdBase)

	// Create present PDE
	pde := PageDirectoryEntry{
		Present:     true,
		BaseAddress: ptBase,
	}
	pdeValue := EncodePDE(pde)

	m.memory[pdBase] = byte(pdeValue)
	m.memory[pdBase+1] = byte(pdeValue >> 8)
	m.memory[pdBase+2] = byte(pdeValue >> 16)
	m.memory[pdBase+3] = byte(pdeValue >> 24)

	// Create non-present PTE
	pte := PageTableEntry{
		Present: false,
	}
	pteValue := EncodePTE(pte)

	m.memory[ptBase] = byte(pteValue)
	m.memory[ptBase+1] = byte(pteValue >> 8)
	m.memory[ptBase+2] = byte(pteValue >> 16)
	m.memory[ptBase+3] = byte(pteValue >> 24)

	va := uint32(0x00000000)
	_, err := m.PageTableWalk(va)

	if err == nil {
		t.Fatal("expected page fault for non-present PTE")
	}

	// Check that CR2 was updated
	if m.GetPageFaultAddress() != va {
		t.Fatalf("expected CR2=0x%X, got 0x%X", va, m.GetPageFaultAddress())
	}
}

func TestTranslateAddressTLBMiss(t *testing.T) {
	m := NewMachine(65536)

	pdBase := uint32(0x1000)
	ptBase := uint32(0x2000)
	pageFrame := uint32(0x3000)

	m.SetPageDirectoryBase(pdBase)

	// Setup PD and PT
	pde := PageDirectoryEntry{
		Present:     true,
		ReadWrite:   true,
		BaseAddress: ptBase,
	}
	pdeValue := EncodePDE(pde)
	m.memory[pdBase] = byte(pdeValue)
	m.memory[pdBase+1] = byte(pdeValue >> 8)
	m.memory[pdBase+2] = byte(pdeValue >> 16)
	m.memory[pdBase+3] = byte(pdeValue >> 24)

	pte := PageTableEntry{
		Present:     true,
		ReadWrite:   true,
		BaseAddress: pageFrame,
	}
	pteValue := EncodePTE(pte)
	m.memory[ptBase] = byte(pteValue)
	m.memory[ptBase+1] = byte(pteValue >> 8)
	m.memory[ptBase+2] = byte(pteValue >> 16)
	m.memory[ptBase+3] = byte(pteValue >> 24)

	va := uint32(0x00000000)
	pa, err := m.TranslateAddress(va)

	if err != nil {
		t.Fatalf("expected successful translation, got error: %v", err)
	}

	// Physical address should be pageFrame + offset
	expectedPA := pageFrame
	if pa != expectedPA {
		t.Fatalf("got PA 0x%X, want 0x%X", pa, expectedPA)
	}

	// Verify TLB has entry
	if m.tlb.Size() != 1 {
		t.Fatalf("expected 1 TLB entry, got %d", m.tlb.Size())
	}
}

func TestTranslateAddressTLBHit(t *testing.T) {
	m := NewMachine(65536)

	pdBase := uint32(0x1000)
	ptBase := uint32(0x2000)
	pageFrame := uint32(0x3000)

	m.SetPageDirectoryBase(pdBase)

	// Setup PD and PT
	pde := PageDirectoryEntry{
		Present:     true,
		ReadWrite:   true,
		BaseAddress: ptBase,
	}
	pdeValue := EncodePDE(pde)
	m.memory[pdBase] = byte(pdeValue)
	m.memory[pdBase+1] = byte(pdeValue >> 8)
	m.memory[pdBase+2] = byte(pdeValue >> 16)
	m.memory[pdBase+3] = byte(pdeValue >> 24)

	pte := PageTableEntry{
		Present:     true,
		ReadWrite:   true,
		BaseAddress: pageFrame,
	}
	pteValue := EncodePTE(pte)
	m.memory[ptBase] = byte(pteValue)
	m.memory[ptBase+1] = byte(pteValue >> 8)
	m.memory[ptBase+2] = byte(pteValue >> 16)
	m.memory[ptBase+3] = byte(pteValue >> 24)

	va := uint32(0x00000000)

	// First translation: TLB miss
	pa1, err := m.TranslateAddress(va)
	if err != nil {
		t.Fatalf("first translation failed: %v", err)
	}

	initialTLBSize := m.tlb.Size()

	// Second translation: should be TLB hit (TLB size unchanged)
	pa2, err := m.TranslateAddress(va)
	if err != nil {
		t.Fatalf("second translation failed: %v", err)
	}

	if pa1 != pa2 {
		t.Fatalf("PAs differ: first 0x%X, second 0x%X", pa1, pa2)
	}

	if m.tlb.Size() != initialTLBSize {
		t.Fatalf("expected TLB size unchanged at %d, got %d", initialTLBSize, m.tlb.Size())
	}
}

func TestTranslateAddressWithOffset(t *testing.T) {
	m := NewMachine(65536)

	pdBase := uint32(0x1000)
	ptBase := uint32(0x2000)
	pageFrame := uint32(0x3000)

	m.SetPageDirectoryBase(pdBase)

	// Setup PD and PT
	pde := PageDirectoryEntry{
		Present:     true,
		ReadWrite:   true,
		BaseAddress: ptBase,
	}
	pdeValue := EncodePDE(pde)
	m.memory[pdBase] = byte(pdeValue)
	m.memory[pdBase+1] = byte(pdeValue >> 8)
	m.memory[pdBase+2] = byte(pdeValue >> 16)
	m.memory[pdBase+3] = byte(pdeValue >> 24)

	pte := PageTableEntry{
		Present:     true,
		ReadWrite:   true,
		BaseAddress: pageFrame,
	}
	pteValue := EncodePTE(pte)
	m.memory[ptBase] = byte(pteValue)
	m.memory[ptBase+1] = byte(pteValue >> 8)
	m.memory[ptBase+2] = byte(pteValue >> 16)
	m.memory[ptBase+3] = byte(pteValue >> 24)

	// VA with offset within page
	va := uint32(0x00000ABC)
	pa, err := m.TranslateAddress(va)

	if err != nil {
		t.Fatalf("expected successful translation, got error: %v", err)
	}

	// PA should preserve the offset
	offset := ExtractPageOffset(va)
	expectedPA := pageFrame + offset
	if pa != expectedPA {
		t.Fatalf("got PA 0x%X, want 0x%X", pa, expectedPA)
	}
}

func TestTranslateAddressPreservesFlags(t *testing.T) {
	m := NewMachine(65536)

	pdBase := uint32(0x1000)
	ptBase := uint32(0x2000)
	pageFrame := uint32(0x3000)

	m.SetPageDirectoryBase(pdBase)

	// Setup PD and PT
	pde := PageDirectoryEntry{
		Present:     true,
		ReadWrite:   true,
		BaseAddress: ptBase,
	}
	pdeValue := EncodePDE(pde)
	m.memory[pdBase] = byte(pdeValue)
	m.memory[pdBase+1] = byte(pdeValue >> 8)
	m.memory[pdBase+2] = byte(pdeValue >> 16)
	m.memory[pdBase+3] = byte(pdeValue >> 24)

	// PTE with specific flags
	pte := PageTableEntry{
		Present:        true,
		ReadWrite:      false,
		UserSupervisor: true,
		Dirty:          true,
		Global:         false,
		BaseAddress:    pageFrame,
	}
	pteValue := EncodePTE(pte)
	m.memory[ptBase] = byte(pteValue)
	m.memory[ptBase+1] = byte(pteValue >> 8)
	m.memory[ptBase+2] = byte(pteValue >> 16)
	m.memory[ptBase+3] = byte(pteValue >> 24)

	va := uint32(0x00000000)
	pa, err := m.TranslateAddress(va)

	if err != nil {
		t.Fatalf("expected successful translation, got error: %v", err)
	}

	if pa != pageFrame {
		t.Fatalf("got PA 0x%X, want 0x%X", pa, pageFrame)
	}

	// Verify TLB cached flags correctly
	tlbEntry, found := m.tlb.Lookup(va)
	if !found {
		t.Fatal("expected TLB hit")
	}

	if tlbEntry.ReadWrite {
		t.Fatal("expected ReadWrite=false in TLB")
	}
	if !tlbEntry.UserSupervisor {
		t.Fatal("expected UserSupervisor=true in TLB")
	}
	if !tlbEntry.Dirty {
		t.Fatal("expected Dirty=true in TLB")
	}
}

func writeUint32LE(memory []byte, addr uint32, value uint32) {
	memory[addr] = byte(value)
	memory[addr+1] = byte(value >> 8)
	memory[addr+2] = byte(value >> 16)
	memory[addr+3] = byte(value >> 24)
}

func mapPage(t *testing.T, m *Machine, pdBase uint32, ptBase uint32, va uint32, frame uint32) {
	t.Helper()

	if !IsPageAligned(pdBase) || !IsPageAligned(ptBase) || !IsPageAligned(frame) {
		t.Fatalf("pdBase, ptBase and frame must be page-aligned")
	}

	m.SetPageDirectoryBase(pdBase)

	pdIndex := ExtractPDIndex(va)
	ptIndex := ExtractPTIndex(va)

	pde := EncodePDE(PageDirectoryEntry{
		Present:     true,
		ReadWrite:   true,
		BaseAddress: ptBase,
	})
	pdeAddr := pdBase + (pdIndex * 4)
	writeUint32LE(m.memory, pdeAddr, pde)

	pte := EncodePTE(PageTableEntry{
		Present:     true,
		ReadWrite:   true,
		BaseAddress: frame,
	})
	pteAddr := ptBase + (ptIndex * 4)
	writeUint32LE(m.memory, pteAddr, pte)
}

func TestTranslateUsesPagingWhenEnabled(t *testing.T) {
	m := NewMachine(65536)
	m.mmu.Mode = ModeFlat
	m.EnablePaging()

	va := uint32(0x00000123)
	frame := uint32(0x00004000)
	mapPage(t, m, 0x00001000, 0x00002000, va, frame)

	pa, err := m.translate(va, Read, m.privMode)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if want := uint32(0x00004123); pa != want {
		t.Fatalf("translate with paging got 0x%X want 0x%X", pa, want)
	}

	if m.tlb.Size() != 1 {
		t.Fatalf("expected TLB size 1, got %d", m.tlb.Size())
	}
}

func TestTranslateBypassesPagingWhenDisabled(t *testing.T) {
	m := NewMachine(65536)
	m.mmu.Mode = ModeFlat

	va := uint32(0x00000123)
	pa, err := m.translate(va, Read, m.privMode)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pa != va {
		t.Fatalf("translate without paging got 0x%X want 0x%X", pa, va)
	}
}

func TestLoadStoreWithPagingEnabled(t *testing.T) {
	m := NewMachine(65536)
	m.mmu.Mode = ModeFlat
	m.EnablePaging()

	pdBase := uint32(0x00001000)
	ptBase := uint32(0x00002000)

	loadVA := uint32(0x00000100)
	loadFrame := uint32(0x00003000)
	mapPage(t, m, pdBase, ptBase, loadVA, loadFrame)

	storeVA := uint32(0x00001204)
	storeFrame := uint32(0x00004000)
	mapPage(t, m, pdBase, ptBase, storeVA, storeFrame)

	loadPA := MakePhysicalAddress(loadFrame, ExtractPageOffset(loadVA))
	m.memory[loadPA] = 77

	loadInst := Instruction{opcode: (*Machine).load, rd: core.W1, immediate: uint16(loadVA)}
	m.execute(loadInst)
	if m.registers[core.W1] != 77 {
		t.Fatalf("load via paging got %d want 77", m.registers[core.W1])
	}

	m.registers[core.W2] = 99
	storeInst := Instruction{opcode: (*Machine).store, rd: core.W2, immediate: uint16(storeVA)}
	m.execute(storeInst)

	storePA := MakePhysicalAddress(storeFrame, ExtractPageOffset(storeVA))
	if m.memory[storePA] != 99 {
		t.Fatalf("store via paging got %d want 99", m.memory[storePA])
	}
}
