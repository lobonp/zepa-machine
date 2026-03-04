package machine

import "testing"

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
