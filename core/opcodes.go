package core

type (
	Opcode     uint8
	Register   uint8
	FaultError struct {
		Code uint32
		Msg  string
	}
)

const (
	// Define registers
	W0 Register = iota
	W1
	W2
	W3
	W4
	W5
	PC
	SP
	IR
	SR
	MDR
	MAR
	LR
	SSR

	// Control registers
	CR0    // Control Register 0: system control flags
	CR2    // Control Register 2: page fault linear address
	CR3    // Control Register 3: page directory base register
	CR4    // Control Register 4: architecture extensions
	EFLAGS // Extended flags register
)

const (
	// Define opcodes for different instructions
	MV_OPCODE Opcode = iota
	ADD_OPCODE
	SUB_OPCODE
	CMP_OPCODE
	JUMP_OPCODE
	LOAD_OPCODE
	STORE_OPCODE
	FETCH_OPCODE
	HALT_OPCODE
	RET_OPCODE
	BEQ_OPCODE
	BLT_OPCODE
	BGT_OPCODE
	UDF_OPCODE
	DISK2MEM_OPCODE Opcode = 28
)

const (
	OpCodeBitMask    = 0x3F
	RegisterBitMask  = 0x1F
	ImmediateBitMask = 0xFFFF
	Funct5BitMask    = 0x1F
	Funct6BitMask    = 0x3F
)

// Exception codes
const (
	EXC_UNDEFINED          = 0
	EXC_MEMORY_VIOLATION   = 1
	EXC_SEGMENTATION_FAULT = 2
	EXC_PAGE_FAULT         = 3
	EXC_PROTECTION_FAULT   = 4
)

// CR0 control bits
const (
	CR0_PE = 1 << 0  // Protection Enable: enables protected mode
	CR0_WP = 1 << 16 // Write Protect: enables write protection in supervisor mode
	CR0_PG = 1 << 31 // Paging: enables paging
)

// CR3 control bits
const (
	CR3_PDBR_MASK = 0xFFFFF000 // Page Directory Base Register mask (bits 31-12)
)

// CR4 control bits
const (
	CR4_PSE = 1 << 4 // Page Size Extension: enables 4MB pages
	CR4_PGE = 1 << 7 // Page Global Enable: enables global pages in TLB
)

// EFLAGS bits
const (
	EFLAGS_CF        = 1 << 0  // Carry Flag
	EFLAGS_Z         = 1 << 6  // Zero Flag
	EFLAGS_L         = 1 << 7  // Less Than Flag
	EFLAGS_G         = 1 << 8  // Greater Than Flag
	EFLAGS_IF        = 1 << 9  // Interrupt Enable Flag
	EFLAGS_OF        = 1 << 11 // Overflow Flag
	EFLAGS_IOPL_MASK = 3 << 12 // I/O Privilege Level mask (bits 13-12)
)

func (e *FaultError) Error() string {
	return e.Msg
}
