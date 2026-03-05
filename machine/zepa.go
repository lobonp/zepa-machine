// Package machines
package machine

import (
	"errors"
	"fmt"
	"zepa-machine/core"
	assembler "zepa-machine/cross-assembler"
)

type Operation func(m *Machine, inst Instruction)

const (
	opcodeLength = 6
	rdLength     = 5
	rs1Length    = 5
	rs2Length    = 5
	funct5Length = 5
	funct6Length = 6
	immediateLen = 16
	word         = 32

	// wRegisterSaveAreaBytes is the size of the W-register backup area appended
	// after the main memory. Each of the 6 W registers occupies 4 bytes (uint32, little-endian).
	wRegisterSaveAreaBytes = 6 * 4
)

var operations = map[byte]Operation{
	byte(core.MV_OPCODE):       (*Machine).mv,
	byte(core.ADD_OPCODE):      (*Machine).add,
	byte(core.SUB_OPCODE):      (*Machine).sub,
	byte(core.CMP_OPCODE):      (*Machine).cmp,
	byte(core.JUMP_OPCODE):     (*Machine).jump,
	byte(core.LOAD_OPCODE):     (*Machine).load,
	byte(core.STORE_OPCODE):    (*Machine).store,
	byte(core.HALT_OPCODE):     (*Machine).halt,
	byte(core.RET_OPCODE):      (*Machine).ret,
	byte(core.BEQ_OPCODE):      (*Machine).beq,
	byte(core.BLT_OPCODE):      (*Machine).blt,
	byte(core.BGT_OPCODE):      (*Machine).bgt,
	byte(core.UDF_OPCODE):      (*Machine).udf,
	byte(core.DISK2MEM_OPCODE): (*Machine).d2m,
}

type Instruction struct {
	opcode    func(m *Machine, inst Instruction)
	rd        core.Register
	rs1       core.Register
	rs2       core.Register
	funct5    byte
	funct6    byte
	immediate uint16
}

type Disk struct {
	programs [][]byte
}

type Machine struct {
	memory             []byte
	registers          map[core.Register]uint32
	evt                map[uint32]uint32
	disk               Disk
	halted             bool
	mmu                *MMU
	tlb                *TLB
	privMode           Privilege
	userMemoryLimit    uint32
	inException        bool
}

func (m *Machine) InitDisk() {
	m.disk.programs = make([][]byte, 0)
}

func (m *Machine) AddToDisk(program []byte) {
	m.disk.programs = append(m.disk.programs, program)
}

func (m *Machine) d2m(inst Instruction) {
	if len(m.disk.programs) > 0 {
		program := m.disk.programs[0]
		destVA := m.registers[core.W1]
		bytesWritten := uint32(0)

		for i, b := range program {
			offset := uint32(i)
			va := destVA + offset
			if va < destVA {
				m.registers[core.W3] = 0
				m.registers[core.W4] = bytesWritten
				m.disk.programs = m.disk.programs[1:]
				return
			}

			physical, err := m.translate(va, Write, m.privMode)
			if err != nil {
				m.registers[core.W3] = 0
				m.registers[core.W4] = bytesWritten
				m.disk.programs = m.disk.programs[1:]
				return
			}

			if err := m.memoryAccessFault(physical, true); err != nil {
				m.registers[core.W3] = 0
				m.registers[core.W4] = bytesWritten
				m.disk.programs = m.disk.programs[1:]
				return
			}

			m.memory[physical] = b
			bytesWritten++
		}

		m.registers[core.W4] = bytesWritten
		m.registers[core.W3] = 1
		m.disk.programs = m.disk.programs[1:]
	} else {
		m.registers[core.W3] = 0
		m.registers[core.W4] = 0
	}
}

func (m *Machine) mv(inst Instruction) {
	m.registers[inst.rd] = uint32(inst.immediate)
}

func (m *Machine) add(inst Instruction) {
	m.registers[inst.rd] = m.registers[inst.rs1] + m.registers[inst.rs2]
}

func (m *Machine) sub(inst Instruction) {
	m.registers[inst.rd] = m.registers[inst.rs1] - m.registers[inst.rs2]
}

func (m *Machine) cmp(inst Instruction) {
	val1 := m.registers[inst.rs1]
	val2 := m.registers[inst.rs2]

	m.registers[core.SR] = 0
	m.registers[core.EFLAGS] &^= (core.EFLAGS_Z | core.EFLAGS_L | core.EFLAGS_G)

	if val1 == val2 {
		m.registers[core.SR] = 0
		m.registers[core.EFLAGS] |= core.EFLAGS_Z
	} else if val1 > val2 {
		m.registers[core.SR] = 2
		m.registers[core.EFLAGS] |= core.EFLAGS_G
	} else {
		m.registers[core.SR] = 1
		m.registers[core.EFLAGS] |= core.EFLAGS_L
	}
}

func (m *Machine) jump(inst Instruction) {
	m.registers[core.PC] = uint32(inst.immediate)
}

func (m *Machine) beq(inst Instruction) {
	if m.registers[core.SR] == 0 {
		m.registers[core.PC] = uint32(inst.immediate)
	}
}

func (m *Machine) blt(inst Instruction) {
	if m.registers[core.SR] == 1 {
		m.registers[core.PC] = uint32(inst.immediate)
	}
}

func (m *Machine) bgt(inst Instruction) {
	if m.registers[core.SR] == 2 {
		m.registers[core.PC] = uint32(inst.immediate)
	}
}

func (m *Machine) load(inst Instruction) {
	addr := uint32(inst.immediate)
	physical, err := m.translate(addr, Read, m.privMode)
	if m.handleFault(err) {
		return
	}

	if m.handleFault(m.memoryAccessFault(physical, false)) {
		return
	}

	m.registers[inst.rd] = uint32(m.memory[physical])
}

func (m *Machine) store(inst Instruction) {
	addr := uint32(inst.immediate)
	physical, err := m.translate(addr, Write, m.privMode)
	if m.handleFault(err) {
		return
	}

	if m.handleFault(m.memoryAccessFault(physical, true)) {
		return
	}

	m.memory[physical] = byte(m.registers[inst.rd])
}

func (m *Machine) handleFault(err error) bool {
	if err == nil {
		return false
	}

	var fault *core.FaultError
	if errors.As(err, &fault) && fault != nil {
		m.exception(fault.Code)
		return true
	}

	m.exception(core.EXC_UNDEFINED)
	return true
}

func (m *Machine) memoryAccessFault(addr uint32, isWrite bool) error {
	if addr >= uint32(len(m.memory)) {
		return &core.FaultError{Code: core.EXC_MEMORY_VIOLATION, Msg: "MEMORY_VIOLATION: ADDRESS OUT OF RANGE"}
	}

	// Keep exception handlers and register backup area as privileged memory (only enforce in user mode).
	if m.privMode == UserPrivilege && addr >= m.userMemoryLimit {
		return &core.FaultError{Code: core.EXC_PROTECTION_FAULT, Msg: "PROTECTION_FAULT: PRIVILEGED MEMORY"}
	}

	if isWrite && m.IsWriteProtectEnabled() {
		if !m.IsPagingEnabled() && m.mmu.Mode == ModeSegmented {
			segID := addr >> SegmentShift
			if segID < uint32(len(m.mmu.Segments)) {
				if m.mmu.Segments[segID].Protection&Write == 0 {
					return &core.FaultError{Code: core.EXC_PROTECTION_FAULT, Msg: "PROTECTION_FAULT: WRITE PROTECTED"}
				}
			}
		}
	}

	return nil
}

func (m *Machine) fetch() {
	instrStart := m.registers[core.PC]
	var completeInstruction uint32
	for i := 0; i < 4; i++ {
		currentInstructionAddress := m.registers[core.PC]
		physical, err := m.translate(currentInstructionAddress, Execute, m.privMode)
		if err != nil {
			m.registers[core.PC] = instrStart
			m.handleFault(err)
			return
		}

		if faultErr := m.memoryAccessFault(physical, false); faultErr != nil {
			m.registers[core.PC] = instrStart
			m.handleFault(faultErr)
			return
		}

		currentInstruction := m.memory[physical]
		completeInstruction |= uint32(currentInstruction) << (24 - 8*i)
		m.registers[core.PC] += 1
	}
	m.registers[core.IR] = completeInstruction
}

func (m *Machine) decodeRTypeInst(instruction uint32) Instruction {
	offsetOpcode := word - opcodeLength
	offSetRd := offsetOpcode - rdLength
	offSetRs1 := offSetRd - rs1Length
	offSetRs2 := offSetRs1 - rs2Length
	offSetFunct5 := offSetRs2 - funct5Length
	offSetFunct6 := offSetFunct5 - funct6Length

	opcode := instruction >> uint32(offsetOpcode) & core.OpCodeBitMask
	rd := (instruction >> uint32(offSetRd)) & core.RegisterBitMask
	rs1 := (instruction >> uint32(offSetRs1)) & core.RegisterBitMask
	rs2 := (instruction >> uint32(offSetRs2)) & core.RegisterBitMask
	funct5 := (instruction >> uint32(offSetFunct5)) & core.Funct5BitMask
	funct6 := (instruction >> uint32(offSetFunct6)) & core.Funct6BitMask

	operation := operations[byte(opcode)]

	return Instruction{
		opcode: operation,
		rd:     core.Register(rd),
		rs1:    core.Register(rs1),
		rs2:    core.Register(rs2),
		funct5: byte(funct5),
		funct6: byte(funct6),
	}
}

func (m *Machine) decodeITypeInst(instruction uint32) Instruction {
	offsetOpcode := word - opcodeLength
	offSetRdRs1 := offsetOpcode - rdLength
	offSetImmediate := offSetRdRs1 - immediateLen
	offSetFunct5 := offSetImmediate - funct5Length

	opcode := instruction >> uint32(offsetOpcode) & core.OpCodeBitMask
	rdRs1 := (instruction >> uint32(offSetRdRs1)) & core.RegisterBitMask
	immediate := (instruction >> uint32(offSetImmediate)) & core.ImmediateBitMask
	funct5 := (instruction >> uint32(offSetFunct5)) & core.Funct5BitMask

	operation := operations[byte(opcode)]

	return Instruction{
		opcode:    operation,
		rd:        core.Register(rdRs1),
		immediate: uint16(immediate),
		funct5:    byte(funct5),
	}
}

func (m *Machine) isEndOfProgram() bool {
	if m.registers[core.IR] == 0 {
		m.registers[core.PC] -= 4
		return true
	}
	return false
}

func (m *Machine) getOpcode(instruction uint32) core.Opcode {
	offsetOpcode := word - opcodeLength
	opcode := instruction >> uint32(offsetOpcode)

	return core.Opcode(opcode)
}

func (m *Machine) decode() Instruction {
	instruction := m.registers[core.IR]
	opcode := m.getOpcode(instruction)

	switch opcode {
	case core.ADD_OPCODE, core.SUB_OPCODE, core.CMP_OPCODE:
		return m.decodeRTypeInst(instruction)
	case core.MV_OPCODE, core.JUMP_OPCODE, core.LOAD_OPCODE, core.STORE_OPCODE,
		core.HALT_OPCODE, core.RET_OPCODE, core.BEQ_OPCODE, core.BGT_OPCODE, core.BLT_OPCODE, core.UDF_OPCODE:
		fallthrough
	default:
		return m.decodeITypeInst(instruction)
	}
}

func (m *Machine) execute(inst Instruction) {
	inst.opcode(m, inst)
}

func (m *Machine) Boot() {
	for !m.halted {
		m.fetch()
		if m.isEndOfProgram() {
			break
		}
		decodedInstruction := m.decode()
		m.execute(decodedInstruction)
	}
}

func (m *Machine) LoadProgram(program []byte) {
	copy(m.memory, program)
}

func (m *Machine) GetMemory() []byte {
	return m.memory
}

func (m *Machine) GetRegisters() map[core.Register]uint32 {
	return m.registers
}

func NewMachine(memoryBytes int) *Machine {
	handlerCode, err := assembler.ConvertInstructionsToBinary([][]string{
		{"HALT"}, // offset +0 : Default Handler
		{"RET"},  // offset +4 : Memory / Page / Protection Fault Handler
		{"RET"},  // offset +8 : Segmentation Fault Handler
	})
	if err != nil {
		fmt.Printf("%v\n", err)
	}

	segLimit := memoryBytes
	if segLimit > MaxSegmentSize {
		segLimit = MaxSegmentSize
	}
	handlerCodePA := uint32(0)
	if segLimit >= len(handlerCode) {
		handlerCodePA = uint32(segLimit - len(handlerCode))
	}

	machineMemory := memoryBytes + wRegisterSaveAreaBytes
	tlb := NewTLB(DefaultTLBSize)

	machine := &Machine{
		memory:    make([]byte, machineMemory),
		registers: make(map[core.Register]uint32),
		evt:       make(map[uint32]uint32),
		disk:      Disk{programs: make([][]byte, 0)},
		mmu: &MMU{
			Mode:       ModeSegmented,
			MemorySize: uint32(machineMemory),
			Segments: [NumSegments]Segment{
				{Base: 0, Limit: uint32(MaxSegmentSize), GrowsPositive: true, Protection: Read | Execute, Priv: KernelPrivilege},
				{Base: 16384, Limit: 16384, GrowsPositive: true, Protection: Read | Write, Priv: UserPrivilege},
				{Base: 49152, Limit: 16384, GrowsPositive: false, Protection: Read | Write, Priv: UserPrivilege},
				{Base: 32768, Limit: 16384, GrowsPositive: true, Protection: Read | Write, Priv: KernelPrivilege},
			},
			tlb: tlb,
		},
		tlb:                tlb,
		privMode:           KernelPrivilege,
		userMemoryLimit:    uint32(memoryBytes), // W-backup area starts here
	}

	machine.evt[core.EXC_UNDEFINED] = handlerCodePA
	machine.evt[core.EXC_MEMORY_VIOLATION] = handlerCodePA + 4
	machine.evt[core.EXC_SEGMENTATION_FAULT] = handlerCodePA + 8
	machine.evt[core.EXC_PAGE_FAULT] = handlerCodePA + 4
	machine.evt[core.EXC_PROTECTION_FAULT] = handlerCodePA + 4

	copy(machine.memory[handlerCodePA:], handlerCode)

	machine.registers[core.CR0] = 0
	machine.registers[core.CR2] = 0
	machine.registers[core.CR3] = 0
	machine.registers[core.CR4] = 0
	machine.registers[core.EFLAGS] = 0

	return machine
}

func (m *Machine) exception(code uint32) {
	fmt.Printf("Exception raised: Code %d\n", code)

	if m.inException {
		fmt.Printf("Double fault (code %d during handler) – halting\n", code)
		m.halted = true
		return
	}
	m.inException = true

	// Save W registers into memory as little-endian uint32 (4 bytes each).
	base := len(m.memory) - wRegisterSaveAreaBytes
	for i, reg := range []core.Register{core.W0, core.W1, core.W2, core.W3, core.W4, core.W5} {
		v := m.registers[reg]
		m.memory[base+i*4] = byte(v)
		m.memory[base+i*4+1] = byte(v >> 8)
		m.memory[base+i*4+2] = byte(v >> 16)
		m.memory[base+i*4+3] = byte(v >> 24)
	}

	// Save information
	m.registers[core.LR] = m.registers[core.PC]
	m.registers[core.SSR] = m.registers[core.SR]

	// Redirect to exception handler.
	// Handlers live in the kernel code segment (Seg0) and are always
	// reachable via normal segmentation, just like real hardware.
	m.registers[core.PC] = m.evt[code]
}

func (m *Machine) halt(inst Instruction) {
	// Set halted flag instead of exiting
	m.halted = true
}

func (m *Machine) ret(inst Instruction) {
	// Restore values
	m.registers[core.PC] = m.registers[core.LR]
	m.registers[core.SR] = m.registers[core.SSR]

	// Restore W registers from little-endian uint32 (4 bytes each).
	base := len(m.memory) - wRegisterSaveAreaBytes
	for i, reg := range []core.Register{core.W0, core.W1, core.W2, core.W3, core.W4, core.W5} {
		m.registers[reg] = uint32(m.memory[base+i*4]) |
			uint32(m.memory[base+i*4+1])<<8 |
			uint32(m.memory[base+i*4+2])<<16 |
			uint32(m.memory[base+i*4+3])<<24
	}

	// Reset link register and clear exception flag.
	m.registers[core.LR] = 0
	m.inException = false
}

func (m *Machine) udf(inst Instruction) {
	m.exception(core.EXC_UNDEFINED)
}

func (m *Machine) translate(va uint32, access AccessType, priv Privilege) (uint32, error) {
	segmentedAddress, err := m.mmu.Translate(va, access, priv)
	if err != nil {
		return 0, err
	}

	if !m.IsPagingEnabled() {
		return segmentedAddress, nil
	}

	return m.mmu.TranslateWithPaging(segmentedAddress, access, priv, m.GetPageDirectoryBase(), m.memory, m.SetPageFaultAddress)
}

// CR0 helper functions
func (m *Machine) IsPagingEnabled() bool {
	return (m.registers[core.CR0] & core.CR0_PG) != 0
}

func (m *Machine) IsProtectedModeEnabled() bool {
	return (m.registers[core.CR0] & core.CR0_PE) != 0
}

func (m *Machine) IsWriteProtectEnabled() bool {
	return (m.registers[core.CR0] & core.CR0_WP) != 0
}

func (m *Machine) EnablePaging() {
	m.registers[core.CR0] |= core.CR0_PG
}

func (m *Machine) DisablePaging() {
	m.registers[core.CR0] &^= core.CR0_PG
}

func (m *Machine) EnableProtectedMode() {
	m.registers[core.CR0] |= core.CR0_PE
}

func (m *Machine) EnableWriteProtect() {
	m.registers[core.CR0] |= core.CR0_WP
}

// CR3 helper functions
func (m *Machine) GetPageDirectoryBase() uint32 {
	return m.registers[core.CR3] & core.CR3_PDBR_MASK
}

func (m *Machine) SetPageDirectoryBase(physAddr uint32) {
	m.registers[core.CR3] = physAddr & core.CR3_PDBR_MASK
}

// CR4 helper functions
func (m *Machine) IsPSEEnabled() bool {
	return (m.registers[core.CR4] & core.CR4_PSE) != 0
}

func (m *Machine) IsPGEEnabled() bool {
	return (m.registers[core.CR4] & core.CR4_PGE) != 0
}

func (m *Machine) EnablePSE() {
	m.registers[core.CR4] |= core.CR4_PSE
}

func (m *Machine) EnablePGE() {
	m.registers[core.CR4] |= core.CR4_PGE
}

func (m *Machine) TranslateAddress(va uint32) (uint32, error) {
	return m.mmu.TranslateWithPaging(va, Read, m.privMode, m.GetPageDirectoryBase(), m.memory, m.SetPageFaultAddress)
}

func (m *Machine) PageTableWalk(va uint32) (PageTableEntry, error) {
	return m.mmu.PageTableWalk(va, m.privMode, m.GetPageDirectoryBase(), m.memory, m.SetPageFaultAddress)
}

// CR2 helper functions
func (m *Machine) SetPageFaultAddress(addr uint32) {
	m.registers[core.CR2] = addr
}

func (m *Machine) GetPageFaultAddress() uint32 {
	return m.registers[core.CR2]
}

// EFLAGS helper functions
func (m *Machine) AreInterruptsEnabled() bool {
	return (m.registers[core.EFLAGS] & core.EFLAGS_IF) != 0
}

func (m *Machine) EnableInterrupts() {
	m.registers[core.EFLAGS] |= core.EFLAGS_IF
}

func (m *Machine) DisableInterrupts() {
	m.registers[core.EFLAGS] &^= core.EFLAGS_IF
}

func (m *Machine) GetIOPL() uint8 {
	return uint8((m.registers[core.EFLAGS] & core.EFLAGS_IOPL_MASK) >> 12)
}

func (m *Machine) SetIOPL(level uint8) {
	m.registers[core.EFLAGS] = (m.registers[core.EFLAGS] &^ uint32(core.EFLAGS_IOPL_MASK)) | (uint32(level&3) << 12)
}
