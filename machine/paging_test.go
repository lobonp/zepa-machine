package machine

import (
	"testing"
	"zepa-machine/core"
)

func writeUint32LE(memory []byte, addr uint32, value uint32) {
	memory[addr] = byte(value)
	memory[addr+1] = byte(value >> 8)
	memory[addr+2] = byte(value >> 16)
	memory[addr+3] = byte(value >> 24)
}

func readUint32LE(memory []byte, addr uint32) uint32 {
	return uint32(memory[addr]) |
		uint32(memory[addr+1])<<8 |
		uint32(memory[addr+2])<<16 |
		uint32(memory[addr+3])<<24
}

func mapPage(t *testing.T, m *Machine, pdBase uint32, ptBase uint32, va uint32, frame uint32) {
	t.Helper()
	if !IsPageAligned(pdBase) || !IsPageAligned(ptBase) || !IsPageAligned(frame) {
		t.Fatalf("pdBase, ptBase and frame must be page-aligned")
	}
	m.SetPageDirectoryBase(pdBase)
	pdIndex := ExtractPDIndex(va)
	ptIndex := ExtractPTIndex(va)
	writeUint32LE(m.memory, pdBase+(pdIndex*4), EncodePDE(PageDirectoryEntry{
		Present:     true,
		ReadWrite:   true,
		BaseAddress: ptBase,
	}))
	writeUint32LE(m.memory, ptBase+(ptIndex*4), EncodePTE(PageTableEntry{
		Present:     true,
		ReadWrite:   true,
		BaseAddress: frame,
	}))
}

func mapPageWithFlags(t *testing.T, m *Machine, pdBase, ptBase, va, frame uint32, pdeFlags, pteFlags PageDirectoryEntry, pteEntry PageTableEntry) {
	t.Helper()
	m.SetPageDirectoryBase(pdBase)
	pdIndex := ExtractPDIndex(va)
	ptIndex := ExtractPTIndex(va)
	writeUint32LE(m.memory, pdBase+(pdIndex*4), EncodePDE(pdeFlags))
	pteEntry.BaseAddress = frame
	writeUint32LE(m.memory, ptBase+(ptIndex*4), EncodePTE(pteEntry))
}

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

func TestPDERoundTripAllFlagsSet(t *testing.T) {
	in := PageDirectoryEntry{
		Present:        true,
		ReadWrite:      true,
		UserSupervisor: true,
		WriteThrough:   true,
		CacheDisable:   true,
		Accessed:       true,
		Reserved:       true,
		PageSize:       true,
		Global:         true,
		Available:      0x7,
		BaseAddress:    0xFFFFF000,
	}
	if DecodePDE(EncodePDE(in)) != in {
		t.Fatal("PDE round trip failed with all flags set")
	}
}

func TestPDERoundTripAllFlagsClear(t *testing.T) {
	in := PageDirectoryEntry{}
	if DecodePDE(EncodePDE(in)) != in {
		t.Fatal("PDE round trip failed with all flags clear")
	}
}

func TestPDEBaseAddressMasked(t *testing.T) {
	in := PageDirectoryEntry{Present: true, BaseAddress: 0x12345678}
	out := DecodePDE(EncodePDE(in))
	if out.BaseAddress != 0x12345000 {
		t.Fatalf("BaseAddress = 0x%X, want 0x12345000", out.BaseAddress)
	}
}

func TestPDEAvailableBits(t *testing.T) {
	for _, avail := range []uint8{0, 1, 3, 5, 7} {
		in := PageDirectoryEntry{Present: true, Available: avail}
		out := DecodePDE(EncodePDE(in))
		if out.Available != avail {
			t.Fatalf("Available = %d, want %d", out.Available, avail)
		}
	}
}

func TestPDEAvailableOverflow(t *testing.T) {
	in := PageDirectoryEntry{Present: true, Available: 0xFF}
	out := DecodePDE(EncodePDE(in))
	if out.Available != 0x7 {
		t.Fatalf("Available = %d, want 7 (masked to 3 bits)", out.Available)
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
	if DecodePTE(EncodePTE(in)) != in {
		t.Fatalf("PTE round trip failed")
	}
}

func TestPTERoundTripAllFlagsSet(t *testing.T) {
	in := PageTableEntry{
		Present:        true,
		ReadWrite:      true,
		UserSupervisor: true,
		WriteThrough:   true,
		CacheDisable:   true,
		Accessed:       true,
		Dirty:          true,
		PageAttrTable:  true,
		Global:         true,
		Available:      0x7,
		BaseAddress:    0xFFFFF000,
	}
	if DecodePTE(EncodePTE(in)) != in {
		t.Fatal("PTE round trip failed with all flags set")
	}
}

func TestPTERoundTripAllFlagsClear(t *testing.T) {
	in := PageTableEntry{}
	if DecodePTE(EncodePTE(in)) != in {
		t.Fatal("PTE round trip failed with all flags clear")
	}
}

func TestPTEBaseAddressMasked(t *testing.T) {
	in := PageTableEntry{Present: true, BaseAddress: 0xABCDE999}
	out := DecodePTE(EncodePTE(in))
	if out.BaseAddress != 0xABCDE000 {
		t.Fatalf("BaseAddress = 0x%X, want 0xABCDE000", out.BaseAddress)
	}
}

func TestPTEDirtyBitOnly(t *testing.T) {
	in := PageTableEntry{Present: true, Dirty: true, BaseAddress: 0x1000}
	out := DecodePTE(EncodePTE(in))
	if !out.Dirty {
		t.Fatal("Dirty should be set")
	}
	if out.Accessed {
		t.Fatal("Accessed should not be set")
	}
}

func TestAddressExtraction(t *testing.T) {
	var va uint32 = 0xCAFEBABE
	if got, want := ExtractPDIndex(va), uint32(0x32B); got != want {
		t.Fatalf("PDIndex = %d, want %d", got, want)
	}
	if got, want := ExtractPTIndex(va), uint32(0x3EB); got != want {
		t.Fatalf("PTIndex = %d, want %d", got, want)
	}
	if got, want := ExtractPageOffset(va), uint32(0xABE); got != want {
		t.Fatalf("PageOffset = %d, want %d", got, want)
	}
}

func TestAddressExtractionZero(t *testing.T) {
	if ExtractPDIndex(0) != 0 {
		t.Fatal("PDIndex(0) != 0")
	}
	if ExtractPTIndex(0) != 0 {
		t.Fatal("PTIndex(0) != 0")
	}
	if ExtractPageOffset(0) != 0 {
		t.Fatal("PageOffset(0) != 0")
	}
}

func TestAddressExtractionMaxAddress(t *testing.T) {
	va := uint32(0xFFFFFFFF)
	if ExtractPDIndex(va) != 0x3FF {
		t.Fatalf("PDIndex = %d, want 1023", ExtractPDIndex(va))
	}
	if ExtractPTIndex(va) != 0x3FF {
		t.Fatalf("PTIndex = %d, want 1023", ExtractPTIndex(va))
	}
	if ExtractPageOffset(va) != 0xFFF {
		t.Fatalf("PageOffset = %d, want 4095", ExtractPageOffset(va))
	}
}

func TestAddressExtractionBoundary(t *testing.T) {
	va := uint32(0x00400000)
	if ExtractPDIndex(va) != 1 {
		t.Fatalf("PDIndex = %d, want 1", ExtractPDIndex(va))
	}
	if ExtractPTIndex(va) != 0 {
		t.Fatalf("PTIndex = %d, want 0", ExtractPTIndex(va))
	}
}

func TestAddressExtractionFirstPTEntry(t *testing.T) {
	va := uint32(0x00001000)
	if ExtractPDIndex(va) != 0 {
		t.Fatalf("PDIndex = %d, want 0", ExtractPDIndex(va))
	}
	if ExtractPTIndex(va) != 1 {
		t.Fatalf("PTIndex = %d, want 1", ExtractPTIndex(va))
	}
}

func TestPhysicalAddressAndAlignment(t *testing.T) {
	if got := MakePhysicalAddress(0x12345000, 0x678); got != 0x12345678 {
		t.Fatalf("MakePhysicalAddress = 0x%X, want 0x12345678", got)
	}
	if !IsPageAligned(0x2000) {
		t.Fatal("0x2000 should be aligned")
	}
	if IsPageAligned(0x2001) {
		t.Fatal("0x2001 should not be aligned")
	}
}

func TestMakePhysicalAddressZero(t *testing.T) {
	if got := MakePhysicalAddress(0, 0); got != 0 {
		t.Fatalf("MakePhysicalAddress(0,0) = 0x%X, want 0", got)
	}
}

func TestMakePhysicalAddressMasks(t *testing.T) {
	if got := MakePhysicalAddress(0xFFFFF123, 0x1FFF); got != MakePhysicalAddress(0xFFFFF000, 0x0FFF) {
		t.Fatalf("masking failed: got 0x%X", got)
	}
}

func TestIsPageAlignedEdgeCases(t *testing.T) {
	if !IsPageAligned(0) {
		t.Fatal("0 should be page-aligned")
	}
	if !IsPageAligned(0x1000) {
		t.Fatal("0x1000 should be page-aligned")
	}
	if IsPageAligned(0xFFF) {
		t.Fatal("0xFFF should not be page-aligned")
	}
	if !IsPageAligned(0xFFFFF000) {
		t.Fatal("0xFFFFF000 should be page-aligned")
	}
}

func TestPageFaultError(t *testing.T) {
	pf := &PageFault{VirtualAddress: 0xDEAD, Reason: "test", IsWrite: false}
	got := pf.Error()
	if got != "test (read access at 0xDEAD)" {
		t.Fatalf("Error() = %q", got)
	}

	pf.IsWrite = true
	got = pf.Error()
	if got != "test (write access at 0xDEAD)" {
		t.Fatalf("Error() = %q", got)
	}
}

func TestPageTableWalkSuccess(t *testing.T) {
	m := NewMachine(65536)
	pdBase := uint32(0x1000)
	ptBase := uint32(0x2000)
	pageFrame := uint32(0x3000)
	m.SetPageDirectoryBase(pdBase)

	writeUint32LE(m.memory, pdBase, EncodePDE(PageDirectoryEntry{
		Present: true, ReadWrite: true, BaseAddress: ptBase,
	}))
	writeUint32LE(m.memory, ptBase, EncodePTE(PageTableEntry{
		Present: true, ReadWrite: true, BaseAddress: pageFrame,
	}))

	pte, err := m.PageTableWalk(0x00000000, Read)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pte.BaseAddress != pageFrame {
		t.Fatalf("frame = 0x%X, want 0x%X", pte.BaseAddress, pageFrame)
	}
	if !pte.Present {
		t.Fatal("PTE should be present")
	}
}

func TestPageTableWalkSetsAccessedBits(t *testing.T) {
	m := NewMachine(65536)
	pdBase := uint32(0x1000)
	ptBase := uint32(0x2000)
	m.SetPageDirectoryBase(pdBase)

	writeUint32LE(m.memory, pdBase, EncodePDE(PageDirectoryEntry{
		Present: true, ReadWrite: true, BaseAddress: ptBase,
	}))
	writeUint32LE(m.memory, ptBase, EncodePTE(PageTableEntry{
		Present: true, ReadWrite: true, BaseAddress: 0x3000,
	}))

	m.PageTableWalk(0x00000000, Read)

	pde := DecodePDE(readUint32LE(m.memory, pdBase))
	if !pde.Accessed {
		t.Fatal("PDE.Accessed should be set after walk")
	}
	pte := DecodePTE(readUint32LE(m.memory, ptBase))
	if !pte.Accessed {
		t.Fatal("PTE.Accessed should be set after walk")
	}
}

func TestPageTableWalkSetsDirtyOnWrite(t *testing.T) {
	m := NewMachine(65536)
	pdBase := uint32(0x1000)
	ptBase := uint32(0x2000)
	m.SetPageDirectoryBase(pdBase)

	writeUint32LE(m.memory, pdBase, EncodePDE(PageDirectoryEntry{
		Present: true, ReadWrite: true, BaseAddress: ptBase,
	}))
	writeUint32LE(m.memory, ptBase, EncodePTE(PageTableEntry{
		Present: true, ReadWrite: true, BaseAddress: 0x3000,
	}))

	m.PageTableWalk(0x00000000, Write)

	pte := DecodePTE(readUint32LE(m.memory, ptBase))
	if !pte.Dirty {
		t.Fatal("PTE.Dirty should be set after write walk")
	}
}

func TestPageTableWalkDoesNotSetDirtyOnRead(t *testing.T) {
	m := NewMachine(65536)
	pdBase := uint32(0x1000)
	ptBase := uint32(0x2000)
	m.SetPageDirectoryBase(pdBase)

	writeUint32LE(m.memory, pdBase, EncodePDE(PageDirectoryEntry{
		Present: true, ReadWrite: true, BaseAddress: ptBase,
	}))
	writeUint32LE(m.memory, ptBase, EncodePTE(PageTableEntry{
		Present: true, ReadWrite: true, BaseAddress: 0x3000,
	}))

	m.PageTableWalk(0x00000000, Read)

	pte := DecodePTE(readUint32LE(m.memory, ptBase))
	if pte.Dirty {
		t.Fatal("PTE.Dirty should not be set after read walk")
	}
}

func TestPageTableWalkPDENotPresent(t *testing.T) {
	m := NewMachine(65536)
	m.SetPageDirectoryBase(0x1000)
	writeUint32LE(m.memory, 0x1000, EncodePDE(PageDirectoryEntry{Present: false}))

	_, err := m.PageTableWalk(0x00000000, Read)
	if err == nil {
		t.Fatal("expected page fault for non-present PDE")
	}
	if m.GetPageFaultAddress() != 0 {
		t.Fatalf("CR2 = 0x%X, want 0", m.GetPageFaultAddress())
	}
}

func TestPageTableWalkPTENotPresent(t *testing.T) {
	m := NewMachine(65536)
	m.SetPageDirectoryBase(0x1000)
	writeUint32LE(m.memory, 0x1000, EncodePDE(PageDirectoryEntry{
		Present: true, BaseAddress: 0x2000,
	}))
	writeUint32LE(m.memory, 0x2000, EncodePTE(PageTableEntry{Present: false}))

	_, err := m.PageTableWalk(0x00000000, Read)
	if err == nil {
		t.Fatal("expected page fault for non-present PTE")
	}
	if m.GetPageFaultAddress() != 0 {
		t.Fatalf("CR2 = 0x%X, want 0", m.GetPageFaultAddress())
	}
}

func TestPageTableWalkUserAccessToKernelPDE(t *testing.T) {
	m := NewMachine(65536)
	m.privMode = UserPrivilege
	m.SetPageDirectoryBase(0x1000)
	writeUint32LE(m.memory, 0x1000, EncodePDE(PageDirectoryEntry{
		Present: true, UserSupervisor: false, BaseAddress: 0x2000,
	}))

	_, err := m.PageTableWalk(0x00000000, Read)
	if err == nil {
		t.Fatal("expected protection fault for user access to kernel PDE")
	}
}

func TestPageTableWalkUserAccessToKernelPTE(t *testing.T) {
	m := NewMachine(65536)
	m.privMode = UserPrivilege
	m.SetPageDirectoryBase(0x1000)
	writeUint32LE(m.memory, 0x1000, EncodePDE(PageDirectoryEntry{
		Present: true, UserSupervisor: true, BaseAddress: 0x2000,
	}))
	writeUint32LE(m.memory, 0x2000, EncodePTE(PageTableEntry{
		Present: true, UserSupervisor: false, BaseAddress: 0x3000,
	}))

	_, err := m.PageTableWalk(0x00000000, Read)
	if err == nil {
		t.Fatal("expected protection fault for user access to kernel PTE")
	}
}

func TestPageTableWalkPDOutOfBounds(t *testing.T) {
	m := NewMachine(4096)
	m.SetPageDirectoryBase(0xFFFF0000)

	_, err := m.PageTableWalk(0x00000000, Read)
	if err == nil {
		t.Fatal("expected page fault for PD out of bounds")
	}
}

func TestPageTableWalkPTOutOfBounds(t *testing.T) {
	m := NewMachine(8192)
	m.SetPageDirectoryBase(0x1000)
	writeUint32LE(m.memory, 0x1000, EncodePDE(PageDirectoryEntry{
		Present: true, BaseAddress: 0xFFFF0000,
	}))

	_, err := m.PageTableWalk(0x00000000, Read)
	if err == nil {
		t.Fatal("expected page fault for PT out of bounds")
	}
}

func TestTranslateAddressTLBMiss(t *testing.T) {
	m := NewMachine(65536)
	mapPage(t, m, 0x1000, 0x2000, 0x00000000, 0x3000)

	pa, err := m.TranslateAddress(0x00000000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pa != 0x3000 {
		t.Fatalf("PA = 0x%X, want 0x3000", pa)
	}
	if m.tlb.Size() != 1 {
		t.Fatalf("TLB size = %d, want 1", m.tlb.Size())
	}
}

func TestTranslateAddressTLBHit(t *testing.T) {
	m := NewMachine(65536)
	mapPage(t, m, 0x1000, 0x2000, 0x00000000, 0x3000)

	pa1, _ := m.TranslateAddress(0x00000000)
	size1 := m.tlb.Size()
	pa2, _ := m.TranslateAddress(0x00000000)

	if pa1 != pa2 {
		t.Fatalf("PAs differ: 0x%X vs 0x%X", pa1, pa2)
	}
	if m.tlb.Size() != size1 {
		t.Fatalf("TLB size changed: %d -> %d", size1, m.tlb.Size())
	}
}

func TestTranslateAddressWithOffset(t *testing.T) {
	m := NewMachine(65536)
	mapPage(t, m, 0x1000, 0x2000, 0x00000ABC, 0x3000)

	pa, err := m.TranslateAddress(0x00000ABC)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pa != 0x3000+0xABC {
		t.Fatalf("PA = 0x%X, want 0x%X", pa, 0x3000+0xABC)
	}
}

func TestTranslateAddressPreservesFlags(t *testing.T) {
	m := NewMachine(65536)
	m.SetPageDirectoryBase(0x1000)
	writeUint32LE(m.memory, 0x1000, EncodePDE(PageDirectoryEntry{
		Present: true, ReadWrite: true, BaseAddress: 0x2000,
	}))
	writeUint32LE(m.memory, 0x2000, EncodePTE(PageTableEntry{
		Present: true, ReadWrite: false, UserSupervisor: true, Dirty: true, BaseAddress: 0x3000,
	}))

	m.TranslateAddress(0x00000000)

	entry, found := m.tlb.Lookup(0x00000000)
	if !found {
		t.Fatal("expected TLB hit")
	}
	if entry.ReadWrite {
		t.Fatal("expected ReadWrite=false")
	}
	if !entry.UserSupervisor {
		t.Fatal("expected UserSupervisor=true")
	}
	if !entry.Dirty {
		t.Fatal("expected Dirty=true")
	}
}

func TestTranslateAddressRejectsUserAccessToKernelPage(t *testing.T) {
	m := NewMachine(65536)
	m.mmu.Mode = ModeFlat
	m.EnablePaging()
	m.privMode = UserPrivilege
	m.SetPageDirectoryBase(0x1000)

	writeUint32LE(m.memory, 0x1000, EncodePDE(PageDirectoryEntry{
		Present: true, ReadWrite: true, UserSupervisor: false, BaseAddress: 0x2000,
	}))
	writeUint32LE(m.memory, 0x2000, EncodePTE(PageTableEntry{
		Present: true, ReadWrite: true, UserSupervisor: false, BaseAddress: 0x3000,
	}))

	_, err := m.TranslateAddress(0x00000000)
	if err == nil {
		t.Fatal("expected protection fault")
	}
}

func TestTranslateAddressTLBHitStillChecksUserAccess(t *testing.T) {
	m := NewMachine(65536)
	m.mmu.Mode = ModeFlat
	m.EnablePaging()
	m.SetPageDirectoryBase(0x1000)

	writeUint32LE(m.memory, 0x1000, EncodePDE(PageDirectoryEntry{
		Present: true, ReadWrite: true, UserSupervisor: false, BaseAddress: 0x2000,
	}))
	writeUint32LE(m.memory, 0x2000, EncodePTE(PageTableEntry{
		Present: true, ReadWrite: true, UserSupervisor: false, BaseAddress: 0x3000,
	}))

	m.privMode = KernelPrivilege
	if _, err := m.TranslateAddress(0x44); err != nil {
		t.Fatalf("kernel translation failed: %v", err)
	}

	m.privMode = UserPrivilege
	if _, err := m.TranslateAddress(0x44); err == nil {
		t.Fatal("expected protection fault on TLB hit for user access")
	}
}

func TestTranslateAddressTLBHitReadOnlyWrite(t *testing.T) {
	m := NewMachine(65536)
	m.mmu.Mode = ModeFlat
	m.EnablePaging()
	m.SetPageDirectoryBase(0x1000)

	writeUint32LE(m.memory, 0x1000, EncodePDE(PageDirectoryEntry{
		Present: true, ReadWrite: true, BaseAddress: 0x2000,
	}))
	writeUint32LE(m.memory, 0x2000, EncodePTE(PageTableEntry{
		Present: true, ReadWrite: false, BaseAddress: 0x3000,
	}))

	m.TranslateAddress(0x00000000)

	_, err := m.mmu.TranslateWithPaging(0x00000000, Write, KernelPrivilege, m.GetPageDirectoryBase(), m.memory, m.SetPageFaultAddress)
	if err == nil {
		t.Fatal("expected protection fault for write to read-only page via TLB hit")
	}
}

func TestTranslateUsesPagingWhenEnabled(t *testing.T) {
	m := NewMachine(65536)
	m.mmu.Mode = ModeFlat
	m.EnablePaging()
	mapPage(t, m, 0x1000, 0x2000, 0x123, 0x4000)

	pa, err := m.translate(0x123, Read, m.privMode)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pa != 0x4123 {
		t.Fatalf("PA = 0x%X, want 0x4123", pa)
	}
}

func TestTranslateBypassesPagingWhenDisabled(t *testing.T) {
	m := NewMachine(65536)
	m.mmu.Mode = ModeFlat
	pa, err := m.translate(0x123, Read, m.privMode)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pa != 0x123 {
		t.Fatalf("PA = 0x%X, want 0x123", pa)
	}
}

func TestLoadStoreWithPagingEnabled(t *testing.T) {
	m := NewMachine(65536)
	m.mmu.Mode = ModeFlat
	m.EnablePaging()
	pdBase := uint32(0x1000)
	ptBase := uint32(0x2000)
	loadFrame := uint32(0x3000)
	storeFrame := uint32(0x4000)

	loadVA := uint32(0x100)
	mapPage(t, m, pdBase, ptBase, loadVA, loadFrame)
	storeVA := uint32(0x1204)
	mapPage(t, m, pdBase, ptBase, storeVA, storeFrame)

	m.memory[MakePhysicalAddress(loadFrame, ExtractPageOffset(loadVA))] = 77
	m.execute(Instruction{opcode: (*Machine).load, rd: core.W1, immediate: uint16(loadVA)})
	if m.registers[core.W1] != 77 {
		t.Fatalf("load via paging = %d, want 77", m.registers[core.W1])
	}

	m.registers[core.W2] = 99
	m.execute(Instruction{opcode: (*Machine).store, rd: core.W2, immediate: uint16(storeVA)})
	if m.memory[MakePhysicalAddress(storeFrame, ExtractPageOffset(storeVA))] != 99 {
		t.Fatal("store via paging failed")
	}
}

func TestSegmentationAndPagingCombined(t *testing.T) {
	m := NewMachine(65536)
	m.mmu.Mode = ModeSegmented
	m.EnablePaging()

	pdBase := uint32(0x5000)
	ptBase := uint32(0x6000)
	frame := uint32(0x7000)
	m.SetPageDirectoryBase(pdBase)

	va := uint32(0x100)
	pdIndex := ExtractPDIndex(va)
	ptIndex := ExtractPTIndex(va)

	writeUint32LE(m.memory, pdBase+(pdIndex*4), EncodePDE(PageDirectoryEntry{
		Present: true, ReadWrite: true, BaseAddress: ptBase,
	}))
	writeUint32LE(m.memory, ptBase+(ptIndex*4), EncodePTE(PageTableEntry{
		Present: true, ReadWrite: true, BaseAddress: frame,
	}))

	offset := ExtractPageOffset(va)
	m.memory[frame+offset] = 0xAB

	m.execute(Instruction{opcode: (*Machine).load, rd: core.W1, immediate: uint16(va)})
	if m.registers[core.W1] != 0xAB {
		t.Fatalf("seg+paging LOAD = 0x%X, want 0xAB", m.registers[core.W1])
	}
}

func TestDirtyBitPersistOnTLBHitWrite(t *testing.T) {
	m := NewMachine(65536)
	m.mmu.Mode = ModeFlat
	m.EnablePaging()

	pdBase := uint32(0x1000)
	ptBase := uint32(0x2000)
	pageFrame := uint32(0x3000)
	m.SetPageDirectoryBase(pdBase)

	writeUint32LE(m.memory, pdBase, EncodePDE(PageDirectoryEntry{
		Present: true, ReadWrite: true, BaseAddress: ptBase,
	}))
	writeUint32LE(m.memory, ptBase, EncodePTE(PageTableEntry{
		Present: true, ReadWrite: true, BaseAddress: pageFrame, Dirty: false,
	}))

	va := uint32(0x0000)
	m.mmu.TranslateWithPaging(va, Write, KernelPrivilege, pdBase, m.memory, m.SetPageFaultAddress)

	writeUint32LE(m.memory, ptBase, EncodePTE(PageTableEntry{
		Present: true, ReadWrite: true, BaseAddress: pageFrame, Dirty: false,
	}))
	m.tlb.Flush()
	m.tlb.Insert(va, PageTableEntry{Present: true, ReadWrite: true, BaseAddress: pageFrame, Dirty: false})

	m.mmu.TranslateWithPaging(va, Write, KernelPrivilege, pdBase, m.memory, m.SetPageFaultAddress)

	entry, found := m.tlb.Lookup(va)
	if !found {
		t.Fatal("expected TLB entry")
	}
	if !entry.Dirty {
		t.Fatal("TLB Dirty should be set")
	}
	pte := DecodePTE(readUint32LE(m.memory, ptBase))
	if !pte.Dirty {
		t.Fatal("PTE Dirty should be persisted")
	}
}

func TestWriteToReadOnlyPageFault(t *testing.T) {
	m := NewMachine(65536)
	m.mmu.Mode = ModeFlat
	m.EnablePaging()
	m.SetPageDirectoryBase(0x1000)

	writeUint32LE(m.memory, 0x1000, EncodePDE(PageDirectoryEntry{
		Present: true, ReadWrite: true, BaseAddress: 0x2000,
	}))
	writeUint32LE(m.memory, 0x2000, EncodePTE(PageTableEntry{
		Present: true, ReadWrite: false, BaseAddress: 0x3000,
	}))

	_, err := m.mmu.TranslateWithPaging(0x0, Write, KernelPrivilege, 0x1000, m.memory, m.SetPageFaultAddress)
	if err == nil {
		t.Fatal("expected protection fault for write to read-only page")
	}
}

func TestMultiplePageMappings(t *testing.T) {
	m := NewMachine(65536)
	m.mmu.Mode = ModeFlat
	m.EnablePaging()
	pdBase := uint32(0x1000)
	ptBase := uint32(0x2000)

	m.SetPageDirectoryBase(pdBase)
	writeUint32LE(m.memory, pdBase, EncodePDE(PageDirectoryEntry{
		Present: true, ReadWrite: true, BaseAddress: ptBase,
	}))

	frames := []uint32{0x4000, 0x5000, 0x6000}
	for i, frame := range frames {
		writeUint32LE(m.memory, ptBase+uint32(i*4), EncodePTE(PageTableEntry{
			Present: true, ReadWrite: true, BaseAddress: frame,
		}))
		m.memory[frame] = byte(0xA0 + i)
	}

	for i, frame := range frames {
		va := uint32(i) * PageSize
		pa, err := m.TranslateAddress(va)
		if err != nil {
			t.Fatalf("page %d translation failed: %v", i, err)
		}
		if pa != frame {
			t.Fatalf("page %d: PA = 0x%X, want 0x%X", i, pa, frame)
		}
	}
}

func TestMMUFlatMode(t *testing.T) {
	mmu := &MMU{Mode: ModeFlat}
	pa, err := mmu.Translate(0x12345, Read, KernelPrivilege)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pa != 0x12345 {
		t.Fatalf("PA = 0x%X, want 0x12345", pa)
	}
}

func TestMMUSegmentedValid(t *testing.T) {
	mmu := &MMU{
		Mode: ModeSegmented,
		Segments: [NumSegments]Segment{
			{Base: 0, Limit: 4096, GrowsPositive: true, Protection: Read | Execute, Priv: KernelPrivilege},
		},
		MemorySize: 65536,
	}
	pa, err := mmu.Translate(100, Read, KernelPrivilege)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pa != 100 {
		t.Fatalf("PA = %d, want 100", pa)
	}
}

func TestMMUSegmentedInvalidSegmentID(t *testing.T) {
	mmu := &MMU{Mode: ModeSegmented, MemorySize: 65536}
	_, err := mmu.Translate(0x10000000, Read, KernelPrivilege)
	if err == nil {
		t.Fatal("expected segmentation fault for invalid segment ID")
	}
}

func TestMMUSegmentedPrivilegeViolation(t *testing.T) {
	mmu := &MMU{
		Mode: ModeSegmented,
		Segments: [NumSegments]Segment{
			{Base: 0, Limit: 4096, GrowsPositive: true, Protection: Read, Priv: KernelPrivilege},
		},
		MemorySize: 65536,
	}
	_, err := mmu.Translate(100, Read, UserPrivilege)
	if err == nil {
		t.Fatal("expected protection fault for user access to kernel segment")
	}
}

func TestMMUSegmentedProtectionViolation(t *testing.T) {
	mmu := &MMU{
		Mode: ModeSegmented,
		Segments: [NumSegments]Segment{
			{Base: 0, Limit: 4096, GrowsPositive: true, Protection: Read, Priv: KernelPrivilege},
		},
		MemorySize: 65536,
	}
	_, err := mmu.Translate(100, Write, KernelPrivilege)
	if err == nil {
		t.Fatal("expected protection fault for write to read-only segment")
	}
}

func TestMMUSegmentedLimitViolation(t *testing.T) {
	mmu := &MMU{
		Mode: ModeSegmented,
		Segments: [NumSegments]Segment{
			{Base: 0, Limit: 10, GrowsPositive: true, Protection: Read, Priv: KernelPrivilege},
		},
		MemorySize: 65536,
	}
	_, err := mmu.Translate(11, Read, KernelPrivilege)
	if err == nil {
		t.Fatal("expected segmentation fault for offset >= limit")
	}
}

func TestMMUSegmentedNegativeGrowth(t *testing.T) {
	mmu := &MMU{
		Mode: ModeSegmented,
		Segments: [NumSegments]Segment{
			{},
			{},
			{Base: 49152, Limit: 16384, GrowsPositive: false, Protection: Read | Write, Priv: UserPrivilege},
		},
		MemorySize: 65536,
	}
	va := uint32(2<<SegmentShift) | uint32(MaxSegmentSize-100)
	pa, err := mmu.Translate(va, Read, UserPrivilege)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pa != 49152-100 {
		t.Fatalf("PA = %d, want %d", pa, 49152-100)
	}
}

func TestMMUSegmentedMemoryBoundsCheck(t *testing.T) {
	mmu := &MMU{
		Mode: ModeSegmented,
		Segments: [NumSegments]Segment{
			{Base: 60000, Limit: 4096, GrowsPositive: true, Protection: Read, Priv: KernelPrivilege},
		},
		MemorySize: 100,
	}
	_, err := mmu.Translate(50, Read, KernelPrivilege)
	if err == nil {
		t.Fatal("expected memory violation for out-of-range physical address")
	}
}

func TestPersistDirtyBitInvalidMemory(t *testing.T) {
	mmu := &MMU{Mode: ModeFlat}
	mmu.ensureTLB()
	memory := make([]byte, 10)
	mmu.persistDirtyBit(0, 0xFFFF0000, memory)
}

func TestPersistDirtyBitPDENotPresent(t *testing.T) {
	mmu := &MMU{Mode: ModeFlat}
	mmu.ensureTLB()
	memory := make([]byte, 65536)
	writeUint32LE(memory, 0x1000, EncodePDE(PageDirectoryEntry{Present: false}))
	mmu.persistDirtyBit(0, 0x1000, memory)
}

func TestPersistDirtyBitAlreadyDirty(t *testing.T) {
	mmu := &MMU{Mode: ModeFlat}
	mmu.ensureTLB()
	memory := make([]byte, 65536)
	writeUint32LE(memory, 0x1000, EncodePDE(PageDirectoryEntry{
		Present: true, BaseAddress: 0x2000,
	}))
	writeUint32LE(memory, 0x2000, EncodePTE(PageTableEntry{
		Present: true, Dirty: true, BaseAddress: 0x3000,
	}))
	mmu.persistDirtyBit(0, 0x1000, memory)
	pte := DecodePTE(readUint32LE(memory, 0x2000))
	if !pte.Dirty {
		t.Fatal("Dirty should remain true")
	}
}

func TestEnsureTLBCreatesDefault(t *testing.T) {
	mmu := &MMU{Mode: ModeFlat}
	mmu.ensureTLB()
	if mmu.tlb == nil {
		t.Fatal("ensureTLB should create TLB")
	}
}

func TestEnsureTLBDoesNotReplace(t *testing.T) {
	tlb := NewTLB(4)
	mmu := &MMU{Mode: ModeFlat, tlb: tlb}
	mmu.ensureTLB()
	if mmu.tlb != tlb {
		t.Fatal("ensureTLB should not replace existing TLB")
	}
}
