package machine

import (
	"bytes"
	"os"
	"testing"
	"zepa-machine/core"

	assembler "zepa-machine/cross-assembler"
)

func captureStdout(fn func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String()
}

func TestFetch(t *testing.T) {
	m := NewMachine(2048)
	m.mmu.Mode = ModeFlat
	m.memory[0] = 0b00111100
	m.memory[1] = 0b01000011
	m.memory[2] = 0b00001000
	m.memory[3] = 0b00000000
	m.fetch()

	if m.registers[core.IR] != 0b00111100010000110000100000000000 {
		t.Fatalf("IR = 0x%08X, want 0x3C430800", m.registers[core.IR])
	}
	if m.registers[core.PC] != 4 {
		t.Fatalf("PC = %d, want 4", m.registers[core.PC])
	}
}

func TestFetchSecondInstruction(t *testing.T) {
	m := NewMachine(2048)
	m.mmu.Mode = ModeFlat
	m.memory[4] = 0xFF
	m.memory[5] = 0xAA
	m.memory[6] = 0x55
	m.memory[7] = 0x00
	m.registers[core.PC] = 4
	m.fetch()

	if m.registers[core.IR] != 0xFFAA5500 {
		t.Fatalf("IR = 0x%08X, want 0xFFAA5500", m.registers[core.IR])
	}
	if m.registers[core.PC] != 8 {
		t.Fatalf("PC = %d, want 8", m.registers[core.PC])
	}
}

func TestFetchAllZeros(t *testing.T) {
	m := NewMachine(2048)
	m.mmu.Mode = ModeFlat
	m.fetch()
	if m.registers[core.IR] != 0 {
		t.Fatalf("IR = 0x%08X, want 0", m.registers[core.IR])
	}
}

func TestDecode(t *testing.T) {
	m := NewMachine(2048)
	m.mmu.Mode = ModeFlat
	m.memory[0] = 0b00111100
	m.memory[1] = 0b01000011
	m.memory[2] = 0b00001000
	m.memory[3] = 0b00000000
	m.fetch()
	decoded := m.decode()

	if decoded.rd != core.Register(2) {
		t.Fatalf("rd = %d, want 2", decoded.rd)
	}
	if decoded.rs1 != core.Register(3) {
		t.Fatalf("rs1 = %d, want 3", decoded.rs1)
	}
	if decoded.rs2 != core.Register(1) {
		t.Fatalf("rs2 = %d, want 1", decoded.rs2)
	}
	if decoded.funct5 != 0 {
		t.Fatalf("funct5 = %d, want 0", decoded.funct5)
	}
	if decoded.funct6 != 0 {
		t.Fatalf("funct6 = %d, want 0", decoded.funct6)
	}
}

func TestDecodeIType(t *testing.T) {
	m := NewMachine(2048)
	m.mmu.Mode = ModeFlat
	program, _ := assembler.ConvertInstructionsToBinary([][]string{{"MV", "W0", "#255"}})
	m.LoadProgram(program)
	m.fetch()
	decoded := m.decode()

	if decoded.rd != core.W0 {
		t.Fatalf("rd = %d, want W0", decoded.rd)
	}
	if decoded.immediate != 255 {
		t.Fatalf("immediate = %d, want 255", decoded.immediate)
	}
}

func TestMV(t *testing.T) {
	m := NewMachine(2048)
	m.execute(Instruction{opcode: (*Machine).mv, rd: core.W0, immediate: 0xFF})
	if m.registers[core.W0] != 0xFF {
		t.Fatalf("W0 = %d, want 255", m.registers[core.W0])
	}
}

func TestMVZero(t *testing.T) {
	m := NewMachine(2048)
	m.registers[core.W0] = 999
	m.execute(Instruction{opcode: (*Machine).mv, rd: core.W0, immediate: 0})
	if m.registers[core.W0] != 0 {
		t.Fatalf("W0 = %d, want 0", m.registers[core.W0])
	}
}

func TestMVMaxImmediate(t *testing.T) {
	m := NewMachine(2048)
	m.execute(Instruction{opcode: (*Machine).mv, rd: core.W0, immediate: 0xFFFF})
	if m.registers[core.W0] != 0xFFFF {
		t.Fatalf("W0 = %d, want 65535", m.registers[core.W0])
	}
}

func TestMVAllRegisters(t *testing.T) {
	m := NewMachine(2048)
	regs := []core.Register{core.W0, core.W1, core.W2, core.W3, core.W4, core.W5}
	for i, r := range regs {
		m.execute(Instruction{opcode: (*Machine).mv, rd: r, immediate: uint16(i + 10)})
	}
	for i, r := range regs {
		if m.registers[r] != uint32(i+10) {
			t.Fatalf("W%d = %d, want %d", i, m.registers[r], i+10)
		}
	}
}

func TestMVCR3FlushTLB(t *testing.T) {
	m := NewMachine(65536)
	m.tlb.Insert(0x1000, PageTableEntry{Present: true, ReadWrite: true, BaseAddress: 0x5000})
	if m.tlb.Size() != 1 {
		t.Fatalf("TLB size = %d, want 1", m.tlb.Size())
	}
	m.execute(Instruction{opcode: (*Machine).mv, rd: core.CR3, immediate: 0x2000})
	if m.tlb.Size() != 0 {
		t.Fatalf("TLB size after MV CR3 = %d, want 0", m.tlb.Size())
	}
	if got := m.GetPageDirectoryBase(); got != 0x2000 {
		t.Fatalf("CR3 = 0x%X, want 0x2000", got)
	}
}

func TestADD(t *testing.T) {
	m := NewMachine(2048)
	m.registers[core.W1] = 66
	m.registers[core.W2] = 3000
	m.execute(Instruction{opcode: (*Machine).add, rd: core.W0, rs1: core.W1, rs2: core.W2})
	if m.registers[core.W0] != 3066 {
		t.Fatalf("W0 = %d, want 3066", m.registers[core.W0])
	}
}

func TestADDZeros(t *testing.T) {
	m := NewMachine(2048)
	m.execute(Instruction{opcode: (*Machine).add, rd: core.W0, rs1: core.W1, rs2: core.W2})
	if m.registers[core.W0] != 0 {
		t.Fatalf("W0 = %d, want 0", m.registers[core.W0])
	}
}

func TestADDOverflow(t *testing.T) {
	m := NewMachine(2048)
	m.registers[core.W1] = 0xFFFFFFFF
	m.registers[core.W2] = 1
	m.execute(Instruction{opcode: (*Machine).add, rd: core.W0, rs1: core.W1, rs2: core.W2})
	if m.registers[core.W0] != 0 {
		t.Fatalf("W0 = 0x%X, want 0 (overflow wrap)", m.registers[core.W0])
	}
}

func TestADDSameRegister(t *testing.T) {
	m := NewMachine(2048)
	m.registers[core.W0] = 5
	m.execute(Instruction{opcode: (*Machine).add, rd: core.W0, rs1: core.W0, rs2: core.W0})
	if m.registers[core.W0] != 10 {
		t.Fatalf("W0 = %d, want 10", m.registers[core.W0])
	}
}

func TestADDLargeValues(t *testing.T) {
	m := NewMachine(2048)
	m.registers[core.W1] = 0x7FFFFFFF
	m.registers[core.W2] = 0x7FFFFFFF
	m.execute(Instruction{opcode: (*Machine).add, rd: core.W0, rs1: core.W1, rs2: core.W2})
	if m.registers[core.W0] != 0xFFFFFFFE {
		t.Fatalf("W0 = 0x%X, want 0xFFFFFFFE", m.registers[core.W0])
	}
}

func TestSUB(t *testing.T) {
	m := NewMachine(2048)
	m.registers[core.W1] = 30
	m.registers[core.W2] = 10
	m.execute(Instruction{opcode: (*Machine).sub, rd: core.W0, rs1: core.W1, rs2: core.W2})
	if m.registers[core.W0] != 20 {
		t.Fatalf("W0 = %d, want 20", m.registers[core.W0])
	}
}

func TestSUBZeros(t *testing.T) {
	m := NewMachine(2048)
	m.execute(Instruction{opcode: (*Machine).sub, rd: core.W0, rs1: core.W1, rs2: core.W2})
	if m.registers[core.W0] != 0 {
		t.Fatalf("W0 = %d, want 0", m.registers[core.W0])
	}
}

func TestSUBUnderflow(t *testing.T) {
	m := NewMachine(2048)
	m.registers[core.W1] = 0
	m.registers[core.W2] = 1
	m.execute(Instruction{opcode: (*Machine).sub, rd: core.W0, rs1: core.W1, rs2: core.W2})
	if m.registers[core.W0] != 0xFFFFFFFF {
		t.Fatalf("W0 = 0x%X, want 0xFFFFFFFF (underflow wrap)", m.registers[core.W0])
	}
}

func TestSUBEqual(t *testing.T) {
	m := NewMachine(2048)
	m.registers[core.W1] = 42
	m.registers[core.W2] = 42
	m.execute(Instruction{opcode: (*Machine).sub, rd: core.W0, rs1: core.W1, rs2: core.W2})
	if m.registers[core.W0] != 0 {
		t.Fatalf("W0 = %d, want 0", m.registers[core.W0])
	}
}

func TestSUBSameRegister(t *testing.T) {
	m := NewMachine(2048)
	m.registers[core.W0] = 100
	m.execute(Instruction{opcode: (*Machine).sub, rd: core.W0, rs1: core.W0, rs2: core.W0})
	if m.registers[core.W0] != 0 {
		t.Fatalf("W0 = %d, want 0", m.registers[core.W0])
	}
}

func TestCMPEqual(t *testing.T) {
	m := NewMachine(2048)
	m.registers[core.W0] = 5
	m.registers[core.W1] = 5
	m.execute(Instruction{opcode: (*Machine).cmp, rs1: core.W0, rs2: core.W1})
	if m.registers[core.SR] != 0 {
		t.Fatalf("SR = %d, want 0", m.registers[core.SR])
	}
	if m.registers[core.EFLAGS]&core.EFLAGS_Z == 0 {
		t.Fatal("EFLAGS_Z not set")
	}
	if m.registers[core.EFLAGS]&core.EFLAGS_L != 0 {
		t.Fatal("EFLAGS_L should not be set")
	}
	if m.registers[core.EFLAGS]&core.EFLAGS_G != 0 {
		t.Fatal("EFLAGS_G should not be set")
	}
}

func TestCMPGreater(t *testing.T) {
	m := NewMachine(2048)
	m.registers[core.W0] = 10
	m.registers[core.W1] = 3
	m.execute(Instruction{opcode: (*Machine).cmp, rs1: core.W0, rs2: core.W1})
	if m.registers[core.SR] != 2 {
		t.Fatalf("SR = %d, want 2", m.registers[core.SR])
	}
	if m.registers[core.EFLAGS]&core.EFLAGS_G == 0 {
		t.Fatal("EFLAGS_G not set")
	}
}

func TestCMPLess(t *testing.T) {
	m := NewMachine(2048)
	m.registers[core.W0] = 1
	m.registers[core.W1] = 100
	m.execute(Instruction{opcode: (*Machine).cmp, rs1: core.W0, rs2: core.W1})
	if m.registers[core.SR] != 1 {
		t.Fatalf("SR = %d, want 1", m.registers[core.SR])
	}
	if m.registers[core.EFLAGS]&core.EFLAGS_L == 0 {
		t.Fatal("EFLAGS_L not set")
	}
}

func TestCMPBothZero(t *testing.T) {
	m := NewMachine(2048)
	m.execute(Instruction{opcode: (*Machine).cmp, rs1: core.W0, rs2: core.W1})
	if m.registers[core.SR] != 0 {
		t.Fatalf("SR = %d, want 0", m.registers[core.SR])
	}
	if m.registers[core.EFLAGS]&core.EFLAGS_Z == 0 {
		t.Fatal("EFLAGS_Z not set for 0==0")
	}
}

func TestCMPMaxValues(t *testing.T) {
	m := NewMachine(2048)
	m.registers[core.W0] = 0xFFFFFFFF
	m.registers[core.W1] = 0xFFFFFFFF
	m.execute(Instruction{opcode: (*Machine).cmp, rs1: core.W0, rs2: core.W1})
	if m.registers[core.SR] != 0 {
		t.Fatalf("SR = %d, want 0", m.registers[core.SR])
	}
}

func TestCMPClearsOldFlags(t *testing.T) {
	m := NewMachine(2048)
	m.registers[core.EFLAGS] = core.EFLAGS_Z | core.EFLAGS_L | core.EFLAGS_G
	m.registers[core.W0] = 10
	m.registers[core.W1] = 3
	m.execute(Instruction{opcode: (*Machine).cmp, rs1: core.W0, rs2: core.W1})
	if m.registers[core.EFLAGS]&core.EFLAGS_Z != 0 {
		t.Fatal("EFLAGS_Z should be cleared")
	}
	if m.registers[core.EFLAGS]&core.EFLAGS_L != 0 {
		t.Fatal("EFLAGS_L should be cleared")
	}
}

func TestCMPPreservesOtherEFLAGS(t *testing.T) {
	m := NewMachine(2048)
	m.registers[core.EFLAGS] = core.EFLAGS_IF | core.EFLAGS_CF
	m.registers[core.W0] = 5
	m.registers[core.W1] = 5
	m.execute(Instruction{opcode: (*Machine).cmp, rs1: core.W0, rs2: core.W1})
	if m.registers[core.EFLAGS]&core.EFLAGS_IF == 0 {
		t.Fatal("EFLAGS_IF should be preserved")
	}
	if m.registers[core.EFLAGS]&core.EFLAGS_CF == 0 {
		t.Fatal("EFLAGS_CF should be preserved")
	}
}

func TestJUMP(t *testing.T) {
	m := NewMachine(2048)
	m.execute(Instruction{opcode: (*Machine).jump, immediate: 0xA})
	if m.registers[core.PC] != 0xA {
		t.Fatalf("PC = %d, want 10", m.registers[core.PC])
	}
}

func TestJUMPToZero(t *testing.T) {
	m := NewMachine(2048)
	m.registers[core.PC] = 100
	m.execute(Instruction{opcode: (*Machine).jump, immediate: 0})
	if m.registers[core.PC] != 0 {
		t.Fatalf("PC = %d, want 0", m.registers[core.PC])
	}
}

func TestJUMPMaxImmediate(t *testing.T) {
	m := NewMachine(2048)
	m.execute(Instruction{opcode: (*Machine).jump, immediate: 0xFFFF})
	if m.registers[core.PC] != 0xFFFF {
		t.Fatalf("PC = %d, want 65535", m.registers[core.PC])
	}
}

func TestLOAD(t *testing.T) {
	m := NewMachine(2048)
	m.mmu.Mode = ModeFlat
	m.memory[256] = 42
	m.execute(Instruction{opcode: (*Machine).load, rd: core.W1, immediate: 256})
	if m.registers[core.W1] != 42 {
		t.Fatalf("W1 = %d, want 42", m.registers[core.W1])
	}
}

func TestLOADZeroAddress(t *testing.T) {
	m := NewMachine(2048)
	m.mmu.Mode = ModeFlat
	m.memory[0] = 0xAB
	m.execute(Instruction{opcode: (*Machine).load, rd: core.W0, immediate: 0})
	if m.registers[core.W0] != 0xAB {
		t.Fatalf("W0 = 0x%X, want 0xAB", m.registers[core.W0])
	}
}

func TestLOADMaxByte(t *testing.T) {
	m := NewMachine(2048)
	m.mmu.Mode = ModeFlat
	m.memory[100] = 0xFF
	m.execute(Instruction{opcode: (*Machine).load, rd: core.W0, immediate: 100})
	if m.registers[core.W0] != 0xFF {
		t.Fatalf("W0 = 0x%X, want 0xFF", m.registers[core.W0])
	}
}

func TestSTORE(t *testing.T) {
	m := NewMachine(2048)
	m.mmu.Mode = ModeFlat
	m.registers[core.W1] = 65
	m.execute(Instruction{opcode: (*Machine).store, rd: core.W1, immediate: 100})
	if m.memory[100] != 65 {
		t.Fatalf("memory[100] = %d, want 65", m.memory[100])
	}
}

func TestSTOREZeroValue(t *testing.T) {
	m := NewMachine(2048)
	m.mmu.Mode = ModeFlat
	m.memory[50] = 0xFF
	m.registers[core.W0] = 0
	m.execute(Instruction{opcode: (*Machine).store, rd: core.W0, immediate: 50})
	if m.memory[50] != 0 {
		t.Fatalf("memory[50] = %d, want 0", m.memory[50])
	}
}

func TestSTORETruncatesToByte(t *testing.T) {
	m := NewMachine(2048)
	m.mmu.Mode = ModeFlat
	m.registers[core.W0] = 0x1234
	m.execute(Instruction{opcode: (*Machine).store, rd: core.W0, immediate: 100})
	if m.memory[100] != 0x34 {
		t.Fatalf("memory[100] = 0x%X, want 0x34", m.memory[100])
	}
}

func TestBEQ(t *testing.T) {
	m := NewMachine(2048)
	m.execute(Instruction{opcode: (*Machine).cmp, rs1: core.W0, rs2: core.W1})
	m.execute(Instruction{opcode: (*Machine).beq, immediate: 0xA})
	if m.registers[core.PC] != 0xA {
		t.Fatalf("PC = %d, want 10 (equal)", m.registers[core.PC])
	}
}

func TestBEQNotTaken(t *testing.T) {
	m := NewMachine(2048)
	m.registers[core.W0] = 1
	m.execute(Instruction{opcode: (*Machine).cmp, rs1: core.W0, rs2: core.W1})
	m.execute(Instruction{opcode: (*Machine).beq, immediate: 0xA})
	if m.registers[core.PC] != 0 {
		t.Fatalf("PC = %d, want 0 (not equal)", m.registers[core.PC])
	}
}

func TestBLT(t *testing.T) {
	m := NewMachine(2048)
	m.registers[core.W1] = 1
	m.execute(Instruction{opcode: (*Machine).cmp, rs1: core.W0, rs2: core.W1})
	m.execute(Instruction{opcode: (*Machine).blt, immediate: 0xA})
	if m.registers[core.PC] != 0xA {
		t.Fatalf("PC = %d, want 10 (less than)", m.registers[core.PC])
	}
}

func TestBLTNotTakenEqual(t *testing.T) {
	m := NewMachine(2048)
	m.registers[core.W0] = 5
	m.registers[core.W1] = 5
	m.execute(Instruction{opcode: (*Machine).cmp, rs1: core.W0, rs2: core.W1})
	m.execute(Instruction{opcode: (*Machine).blt, immediate: 0xA})
	if m.registers[core.PC] != 0 {
		t.Fatalf("PC = %d, want 0 (equal, not less)", m.registers[core.PC])
	}
}

func TestBLTNotTakenGreater(t *testing.T) {
	m := NewMachine(2048)
	m.registers[core.W0] = 10
	m.registers[core.W1] = 5
	m.execute(Instruction{opcode: (*Machine).cmp, rs1: core.W0, rs2: core.W1})
	m.execute(Instruction{opcode: (*Machine).blt, immediate: 0xA})
	if m.registers[core.PC] != 0 {
		t.Fatalf("PC = %d, want 0 (greater, not less)", m.registers[core.PC])
	}
}

func TestBGT(t *testing.T) {
	m := NewMachine(2048)
	m.registers[core.W0] = 1
	m.execute(Instruction{opcode: (*Machine).cmp, rs1: core.W0, rs2: core.W1})
	m.execute(Instruction{opcode: (*Machine).bgt, immediate: 0xA})
	if m.registers[core.PC] != 0xA {
		t.Fatalf("PC = %d, want 10 (greater than)", m.registers[core.PC])
	}
}

func TestBGTNotTakenEqual(t *testing.T) {
	m := NewMachine(2048)
	m.execute(Instruction{opcode: (*Machine).cmp, rs1: core.W0, rs2: core.W1})
	m.execute(Instruction{opcode: (*Machine).bgt, immediate: 0xA})
	if m.registers[core.PC] != 0 {
		t.Fatalf("PC = %d, want 0 (equal, not greater)", m.registers[core.PC])
	}
}

func TestBGTNotTakenLess(t *testing.T) {
	m := NewMachine(2048)
	m.registers[core.W1] = 10
	m.execute(Instruction{opcode: (*Machine).cmp, rs1: core.W0, rs2: core.W1})
	m.execute(Instruction{opcode: (*Machine).bgt, immediate: 0xA})
	if m.registers[core.PC] != 0 {
		t.Fatalf("PC = %d, want 0 (less, not greater)", m.registers[core.PC])
	}
}

func TestHalt(t *testing.T) {
	m := NewMachine(2048)
	m.execute(Instruction{opcode: (*Machine).halt})
	if !m.halted {
		t.Fatal("machine should be halted")
	}
}

func TestHaltIdempotent(t *testing.T) {
	m := NewMachine(2048)
	m.execute(Instruction{opcode: (*Machine).halt})
	m.execute(Instruction{opcode: (*Machine).halt})
	if !m.halted {
		t.Fatal("machine should remain halted")
	}
}

func TestRET(t *testing.T) {
	m := NewMachine(2048)
	m.execute(Instruction{opcode: (*Machine).ret})
	if m.registers[core.PC] != 0 {
		t.Fatalf("PC = %d, want 0", m.registers[core.PC])
	}
}

func TestRETRestoresPC(t *testing.T) {
	m := NewMachine(2048)
	m.registers[core.LR] = 0x100
	m.execute(Instruction{opcode: (*Machine).ret})
	if m.registers[core.PC] != 0x100 {
		t.Fatalf("PC = 0x%X, want 0x100", m.registers[core.PC])
	}
}

func TestRETRestoresSR(t *testing.T) {
	m := NewMachine(2048)
	m.mmu.Mode = ModeFlat
	m.registers[core.SR] = 42
	captureStdout(func() { m.exception(core.EXC_UNDEFINED) })
	if m.registers[core.SSR] != 42 {
		t.Fatalf("SSR = %d, want 42", m.registers[core.SSR])
	}
	m.registers[core.SR] = 0
	m.ret(Instruction{})
	if m.registers[core.SR] != 42 {
		t.Fatalf("SR = %d, want 42 after RET", m.registers[core.SR])
	}
}

func TestRETClearsLR(t *testing.T) {
	m := NewMachine(2048)
	m.registers[core.LR] = 0x200
	m.execute(Instruction{opcode: (*Machine).ret})
	if m.registers[core.LR] != 0 {
		t.Fatalf("LR = 0x%X, want 0", m.registers[core.LR])
	}
}

func TestRETClearsInException(t *testing.T) {
	m := NewMachine(2048)
	m.mmu.Mode = ModeFlat
	captureStdout(func() { m.exception(core.EXC_UNDEFINED) })
	if !m.inException {
		t.Fatal("inException should be true")
	}
	m.ret(Instruction{})
	if m.inException {
		t.Fatal("inException should be false after RET")
	}
}

func TestRETRestoresWRegisters(t *testing.T) {
	m := NewMachine(2048)
	m.mmu.Mode = ModeFlat
	expected := []uint32{10, 20, 30, 40, 50, 60}
	regs := []core.Register{core.W0, core.W1, core.W2, core.W3, core.W4, core.W5}
	for i, r := range regs {
		m.registers[r] = expected[i]
	}

	captureStdout(func() { m.exception(core.EXC_UNDEFINED) })
	for _, r := range regs {
		m.registers[r] = 0
	}
	m.ret(Instruction{})

	for i, r := range regs {
		if m.registers[r] != expected[i] {
			t.Fatalf("W%d = %d, want %d", i, m.registers[r], expected[i])
		}
	}
}

func TestRETWithoutExceptionDoesNotRestoreW(t *testing.T) {
	m := NewMachine(2048)
	m.registers[core.W0] = 99
	m.ret(Instruction{})
	if m.registers[core.W0] != 99 {
		t.Fatalf("W0 = %d, want 99 (RET without exception should not touch W regs)", m.registers[core.W0])
	}
}

func TestUndefinedException(t *testing.T) {
	m := NewMachine(2048)
	got := captureStdout(func() { m.execute(Instruction{opcode: (*Machine).udf}) })
	if got != "Exception raised: Code 0\n" {
		t.Fatalf("got %q, want exception code 0", got)
	}
}

func TestDoubleFaultHalts(t *testing.T) {
	m := NewMachine(2048)
	m.mmu.Mode = ModeFlat
	captureStdout(func() {
		m.exception(core.EXC_UNDEFINED)
		if m.halted {
			t.Fatal("should not halt on first exception")
		}
		m.exception(core.EXC_PAGE_FAULT)
	})
	if !m.halted {
		t.Fatal("should halt on double fault")
	}
}

func TestExceptionSavesPC(t *testing.T) {
	m := NewMachine(2048)
	m.mmu.Mode = ModeFlat
	m.registers[core.PC] = 0x100
	captureStdout(func() { m.exception(core.EXC_UNDEFINED) })
	if m.registers[core.LR] != 0x100 {
		t.Fatalf("LR = 0x%X, want 0x100", m.registers[core.LR])
	}
}

func TestExceptionAfterRETDoesNotDoubleFault(t *testing.T) {
	m := NewMachine(2048)
	m.mmu.Mode = ModeFlat
	captureStdout(func() {
		m.exception(core.EXC_UNDEFINED)
		m.ret(Instruction{})
		m.exception(core.EXC_UNDEFINED)
	})
	if m.halted {
		t.Fatal("should not halt after RET clears inException")
	}
}

func TestExceptionSetsInException(t *testing.T) {
	m := NewMachine(2048)
	m.mmu.Mode = ModeFlat
	captureStdout(func() { m.exception(core.EXC_MEMORY_VIOLATION) })
	if !m.inException {
		t.Fatal("inException should be true after exception")
	}
}

func TestSegmentFaultException(t *testing.T) {
	m := NewMachine(2048)
	got := captureStdout(func() {
		m.execute(Instruction{opcode: (*Machine).load, immediate: 0xFFFF})
	})
	if got != "Exception raised: Code 2\n" {
		t.Fatalf("got %q, want exception code 2", got)
	}
}

func TestPageFaultException(t *testing.T) {
	m := NewMachine(2048)
	m.mmu.Mode = ModeFlat
	m.EnablePaging()
	m.SetPageDirectoryBase(0x400)
	writeUint32LE(m.memory, 0x400, EncodePDE(PageDirectoryEntry{Present: false}))
	got := captureStdout(func() {
		m.execute(Instruction{opcode: (*Machine).load, immediate: 0x100})
	})
	if got != "Exception raised: Code 3\n" {
		t.Fatalf("got %q, want exception code 3", got)
	}
}

func TestProtectionFaultPrivilegedMemory(t *testing.T) {
	m := NewMachine(2048)
	m.registers[core.W1] = 99
	got := captureStdout(func() {
		m.execute(Instruction{opcode: (*Machine).store, rd: core.W1, immediate: 2048})
	})
	if got != "Exception raised: Code 4\n" {
		t.Fatalf("got %q, want exception code 4", got)
	}
}

func TestMemoryAccessFaultOutOfRange(t *testing.T) {
	m := NewMachine(2048)
	m.mmu.Mode = ModeFlat
	got := captureStdout(func() {
		m.execute(Instruction{opcode: (*Machine).load, rd: core.W0, immediate: 0xFFFF})
	})
	if got != "Exception raised: Code 1\n" {
		t.Fatalf("got %q, want exception code 1", got)
	}
}

func TestWriteProtectEnforced(t *testing.T) {
	m := NewMachine(65536)
	m.mmu.Mode = ModeFlat
	m.EnableWriteProtect()
	m.mmu.Mode = ModeSegmented

	m.registers[core.W1] = 65
	got := captureStdout(func() {
		m.execute(Instruction{opcode: (*Machine).store, rd: core.W1, immediate: 100})
	})
	if got != "Exception raised: Code 4\n" {
		t.Fatalf("got %q, want exception code 4", got)
	}
}

func TestTranslateValidFlat(t *testing.T) {
	m := NewMachine(2048)
	m.mmu.Mode = ModeFlat
	pa, err := m.translate(100, Read, KernelPrivilege)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pa != 100 {
		t.Fatalf("PA = %d, want 100", pa)
	}
}

func TestTranslateInvalidLimit(t *testing.T) {
	m := NewMachine(2048)
	m.mmu.Mode = ModeSegmented
	_, err := m.translate(0x4000, Read, KernelPrivilege)
	if err == nil {
		t.Fatal("expected error for limit violation")
	}
}

func TestTranslateBypassesPagingDuringException(t *testing.T) {
	m := NewMachine(65536)
	m.mmu.Mode = ModeFlat
	m.EnablePaging()
	m.inException = true
	pa, err := m.translate(0x100, Read, KernelPrivilege)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pa != 0x100 {
		t.Fatalf("PA = 0x%X, want 0x100", pa)
	}
}

func TestFetchWithMMU(t *testing.T) {
	m := NewMachine(2048)
	m.mmu.Mode = ModeSegmented
	m.memory[0] = 0b00110100
	m.fetch()
	if m.registers[core.PC] != 4 {
		t.Fatalf("PC = %d, want 4", m.registers[core.PC])
	}
}

func TestLoadWithSegmentationFault(t *testing.T) {
	m := NewMachine(2048)
	m.mmu.Mode = ModeSegmented
	got := captureStdout(func() {
		m.execute(Instruction{opcode: (*Machine).load, rd: core.W1, immediate: 0x4000})
	})
	if got != "Exception raised: Code 2\n" {
		t.Fatalf("got %q, want exception code 2", got)
	}
}

func TestLOADR(t *testing.T) {
	m := NewMachine(65536)
	m.mmu.Mode = ModeFlat
	m.memory[0xA000] = 0xBE
	m.registers[core.W2] = 0xA000
	m.execute(Instruction{opcode: (*Machine).loadr, rd: core.W1, rs1: core.W2})
	if m.registers[core.W1] != 0xBE {
		t.Fatalf("W1 = 0x%X, want 0xBE", m.registers[core.W1])
	}
}

func TestLOADRZeroAddress(t *testing.T) {
	m := NewMachine(2048)
	m.mmu.Mode = ModeFlat
	m.memory[0] = 0x42
	m.registers[core.W0] = 0
	m.execute(Instruction{opcode: (*Machine).loadr, rd: core.W1, rs1: core.W0})
	if m.registers[core.W1] != 0x42 {
		t.Fatalf("W1 = 0x%X, want 0x42", m.registers[core.W1])
	}
}

func TestSTORER(t *testing.T) {
	m := NewMachine(65536)
	m.mmu.Mode = ModeFlat
	m.registers[core.W0] = 0xCA
	m.registers[core.W3] = 0xB000
	m.execute(Instruction{opcode: (*Machine).storer, rd: core.W0, rs1: core.W3})
	if m.memory[0xB000] != 0xCA {
		t.Fatalf("memory[0xB000] = 0x%X, want 0xCA", m.memory[0xB000])
	}
}

func TestSTORERZeroAddress(t *testing.T) {
	m := NewMachine(2048)
	m.mmu.Mode = ModeFlat
	m.registers[core.W0] = 0x99
	m.registers[core.W1] = 0
	m.execute(Instruction{opcode: (*Machine).storer, rd: core.W0, rs1: core.W1})
	if m.memory[0] != 0x99 {
		t.Fatalf("memory[0] = 0x%X, want 0x99", m.memory[0])
	}
}

func TestLOADRSegmentFault(t *testing.T) {
	m := NewMachine(2048)
	m.registers[core.W0] = 0xFFFFFFFF
	got := captureStdout(func() {
		m.execute(Instruction{opcode: (*Machine).loadr, rd: core.W1, rs1: core.W0})
	})
	if got == "" {
		t.Fatal("expected exception output")
	}
}

func TestSTORERSegmentFault(t *testing.T) {
	m := NewMachine(2048)
	m.registers[core.W0] = 0xAB
	m.registers[core.W1] = 0xFFFFFFFF
	got := captureStdout(func() {
		m.execute(Instruction{opcode: (*Machine).storer, rd: core.W0, rs1: core.W1})
	})
	if got == "" {
		t.Fatal("expected exception output")
	}
}

func TestD2MEmptyDisk(t *testing.T) {
	m := NewMachine(2048)
	m.execute(Instruction{opcode: (*Machine).d2m})
	if m.registers[core.W3] != 0 {
		t.Fatalf("W3 = %d, want 0", m.registers[core.W3])
	}
	if m.registers[core.W4] != 0 {
		t.Fatalf("W4 = %d, want 0", m.registers[core.W4])
	}
}

func TestD2MSuccess(t *testing.T) {
	m := NewMachine(65536)
	m.mmu.Mode = ModeFlat
	m.AddToDisk([]byte{0xDE, 0xAD, 0xBE, 0xEF})
	m.registers[core.W1] = 0x100
	m.execute(Instruction{opcode: (*Machine).d2m})
	if m.registers[core.W3] != 1 {
		t.Fatalf("W3 = %d, want 1", m.registers[core.W3])
	}
	if m.registers[core.W4] != 4 {
		t.Fatalf("W4 = %d, want 4", m.registers[core.W4])
	}
	if m.memory[0x100] != 0xDE || m.memory[0x101] != 0xAD || m.memory[0x102] != 0xBE || m.memory[0x103] != 0xEF {
		t.Fatal("disk data not written correctly")
	}
}

func TestD2MConsumesProgram(t *testing.T) {
	m := NewMachine(65536)
	m.mmu.Mode = ModeFlat
	m.AddToDisk([]byte{0x01})
	m.AddToDisk([]byte{0x02})
	m.registers[core.W1] = 0x100
	m.execute(Instruction{opcode: (*Machine).d2m})
	if m.memory[0x100] != 0x01 {
		t.Fatalf("first program byte = 0x%X, want 0x01", m.memory[0x100])
	}
	m.registers[core.W1] = 0x200
	m.execute(Instruction{opcode: (*Machine).d2m})
	if m.memory[0x200] != 0x02 {
		t.Fatalf("second program byte = 0x%X, want 0x02", m.memory[0x200])
	}
}

func TestD2MEmptyProgram(t *testing.T) {
	m := NewMachine(65536)
	m.mmu.Mode = ModeFlat
	m.AddToDisk([]byte{})
	m.registers[core.W1] = 0x100
	m.execute(Instruction{opcode: (*Machine).d2m})
	if m.registers[core.W3] != 1 {
		t.Fatalf("W3 = %d, want 1", m.registers[core.W3])
	}
	if m.registers[core.W4] != 0 {
		t.Fatalf("W4 = %d, want 0", m.registers[core.W4])
	}
}

func TestIsEndOfProgram(t *testing.T) {
	m := NewMachine(2048)
	m.registers[core.IR] = 0
	m.registers[core.PC] = 8
	if !m.isEndOfProgram() {
		t.Fatal("should detect end of program when IR=0")
	}
	if m.registers[core.PC] != 4 {
		t.Fatalf("PC = %d, want 4", m.registers[core.PC])
	}
}

func TestIsNotEndOfProgram(t *testing.T) {
	m := NewMachine(2048)
	m.registers[core.IR] = 0x12345678
	if m.isEndOfProgram() {
		t.Fatal("should not detect end of program when IR != 0")
	}
}

func TestBootHaltProgram(t *testing.T) {
	m := NewMachine(2048)
	m.mmu.Mode = ModeFlat
	program, _ := assembler.ConvertInstructionsToBinary([][]string{{"HALT"}})
	m.LoadProgram(program)
	m.Boot()
	if !m.halted {
		t.Fatal("machine should halt after HALT")
	}
}

func TestBootMVAndHalt(t *testing.T) {
	m := NewMachine(2048)
	m.mmu.Mode = ModeFlat
	program, _ := assembler.ConvertInstructionsToBinary([][]string{
		{"MV", "W0", "#42"},
		{"HALT"},
	})
	m.LoadProgram(program)
	m.Boot()
	if m.registers[core.W0] != 42 {
		t.Fatalf("W0 = %d, want 42", m.registers[core.W0])
	}
}

func TestBootAddProgram(t *testing.T) {
	m := NewMachine(2048)
	m.mmu.Mode = ModeFlat
	program, _ := assembler.ConvertInstructionsToBinary([][]string{
		{"MV", "W1", "#10"},
		{"MV", "W2", "#20"},
		{"ADD", "W0", "W1", "W2"},
		{"HALT"},
	})
	m.LoadProgram(program)
	m.Boot()
	if m.registers[core.W0] != 30 {
		t.Fatalf("W0 = %d, want 30", m.registers[core.W0])
	}
}

func TestBootSubProgram(t *testing.T) {
	m := NewMachine(2048)
	m.mmu.Mode = ModeFlat
	program, _ := assembler.ConvertInstructionsToBinary([][]string{
		{"MV", "W1", "#50"},
		{"MV", "W2", "#30"},
		{"SUB", "W0", "W1", "W2"},
		{"HALT"},
	})
	m.LoadProgram(program)
	m.Boot()
	if m.registers[core.W0] != 20 {
		t.Fatalf("W0 = %d, want 20", m.registers[core.W0])
	}
}

func TestBootJumpSkipsInstructions(t *testing.T) {
	m := NewMachine(2048)
	m.mmu.Mode = ModeFlat
	program, _ := assembler.ConvertInstructionsToBinary([][]string{
		{"MV", "W0", "#1"},
		{"JUMP", "#12"},
		{"MV", "W0", "#99"},
		{"HALT"},
	})
	m.LoadProgram(program)
	m.Boot()
	if m.registers[core.W0] != 1 {
		t.Fatalf("W0 = %d, want 1", m.registers[core.W0])
	}
}

func TestBootCmpBeqTaken(t *testing.T) {
	m := NewMachine(2048)
	m.mmu.Mode = ModeFlat
	program, _ := assembler.ConvertInstructionsToBinary([][]string{
		{"MV", "W0", "#5"},
		{"MV", "W1", "#5"},
		{"CMP", "W0", "W1"},
		{"BEQ", "#20"},
		{"MV", "W2", "#99"},
		{"HALT"},
	})
	m.LoadProgram(program)
	m.Boot()
	if m.registers[core.W2] != 0 {
		t.Fatalf("W2 = %d, want 0 (BEQ should skip MV W2)", m.registers[core.W2])
	}
}

func TestBootCmpBgtTaken(t *testing.T) {
	m := NewMachine(2048)
	m.mmu.Mode = ModeFlat
	program, _ := assembler.ConvertInstructionsToBinary([][]string{
		{"MV", "W0", "#10"},
		{"MV", "W1", "#3"},
		{"CMP", "W0", "W1"},
		{"BGT", "#20"},
		{"MV", "W2", "#99"},
		{"HALT"},
	})
	m.LoadProgram(program)
	m.Boot()
	if m.registers[core.W2] != 0 {
		t.Fatalf("W2 = %d, want 0 (BGT should skip MV W2)", m.registers[core.W2])
	}
}

func TestBootCmpBltTaken(t *testing.T) {
	m := NewMachine(2048)
	m.mmu.Mode = ModeFlat
	program, _ := assembler.ConvertInstructionsToBinary([][]string{
		{"MV", "W0", "#1"},
		{"MV", "W1", "#10"},
		{"CMP", "W0", "W1"},
		{"BLT", "#20"},
		{"MV", "W2", "#99"},
		{"HALT"},
	})
	m.LoadProgram(program)
	m.Boot()
	if m.registers[core.W2] != 0 {
		t.Fatalf("W2 = %d, want 0 (BLT should skip MV W2)", m.registers[core.W2])
	}
}

func TestBootEmptyMemoryStops(t *testing.T) {
	m := NewMachine(2048)
	m.mmu.Mode = ModeFlat
	m.Boot()
	if m.registers[core.PC] != 0 {
		t.Fatalf("PC = %d, want 0", m.registers[core.PC])
	}
}

func TestLOADRAssembler(t *testing.T) {
	m := NewMachine(65536)
	m.mmu.Mode = ModeFlat
	m.memory[0xC042] = 0x77
	program, err := assembler.ConvertInstructionsToBinary([][]string{
		{"MV", "W1", "#0xC042"},
		{"LOADR", "W2", "W1"},
		{"HALT"},
	})
	if err != nil {
		t.Fatalf("assembly failed: %v", err)
	}
	m.LoadProgram(program)
	m.Boot()
	if m.registers[core.W2] != 0x77 {
		t.Fatalf("W2 = 0x%X, want 0x77", m.registers[core.W2])
	}
}

func TestSTORERAssembler(t *testing.T) {
	m := NewMachine(65536)
	m.mmu.Mode = ModeFlat
	program, err := assembler.ConvertInstructionsToBinary([][]string{
		{"MV", "W0", "#0xAB"},
		{"MV", "W1", "#0xD000"},
		{"STORER", "W0", "W1"},
		{"HALT"},
	})
	if err != nil {
		t.Fatalf("assembly failed: %v", err)
	}
	m.LoadProgram(program)
	m.Boot()
	if m.memory[0xD000] != 0xAB {
		t.Fatalf("memory[0xD000] = 0x%X, want 0xAB", m.memory[0xD000])
	}
}

func TestNewMachineInitialization(t *testing.T) {
	m := NewMachine(4096)
	if len(m.memory) != 4096+wRegisterSaveAreaBytes {
		t.Fatalf("memory size = %d, want %d", len(m.memory), 4096+wRegisterSaveAreaBytes)
	}
	if m.halted {
		t.Fatal("should not be halted initially")
	}
	if m.inException {
		t.Fatal("should not be in exception initially")
	}
	if m.privMode != KernelPrivilege {
		t.Fatal("should start in kernel mode")
	}
	if m.registers[core.CR0] != 0 {
		t.Fatalf("CR0 = 0x%X, want 0", m.registers[core.CR0])
	}
	if m.registers[core.EFLAGS] != 0 {
		t.Fatalf("EFLAGS = 0x%X, want 0", m.registers[core.EFLAGS])
	}
}

func TestNewMachineHasEVT(t *testing.T) {
	m := NewMachine(4096)
	codes := []uint32{core.EXC_UNDEFINED, core.EXC_MEMORY_VIOLATION, core.EXC_SEGMENTATION_FAULT, core.EXC_PAGE_FAULT, core.EXC_PROTECTION_FAULT}
	for _, code := range codes {
		if _, ok := m.evt[code]; !ok {
			t.Fatalf("EVT missing handler for code %d", code)
		}
	}
}

func TestControlRegisters(t *testing.T) {
	m := NewMachine(2048)

	m.EnablePaging()
	if !m.IsPagingEnabled() {
		t.Fatal("paging should be enabled")
	}
	m.DisablePaging()
	if m.IsPagingEnabled() {
		t.Fatal("paging should be disabled")
	}

	m.EnableProtectedMode()
	if !m.IsProtectedModeEnabled() {
		t.Fatal("protected mode should be enabled")
	}

	m.EnableWriteProtect()
	if !m.IsWriteProtectEnabled() {
		t.Fatal("write protect should be enabled")
	}

	m.EnablePSE()
	if !m.IsPSEEnabled() {
		t.Fatal("PSE should be enabled")
	}

	m.EnablePGE()
	if !m.IsPGEEnabled() {
		t.Fatal("PGE should be enabled")
	}
}

func TestInterruptFlags(t *testing.T) {
	m := NewMachine(2048)
	if m.AreInterruptsEnabled() {
		t.Fatal("interrupts should be disabled initially")
	}
	m.EnableInterrupts()
	if !m.AreInterruptsEnabled() {
		t.Fatal("interrupts should be enabled")
	}
	m.DisableInterrupts()
	if m.AreInterruptsEnabled() {
		t.Fatal("interrupts should be disabled again")
	}
}

func TestIOPL(t *testing.T) {
	m := NewMachine(2048)
	m.SetIOPL(3)
	if m.GetIOPL() != 3 {
		t.Fatalf("IOPL = %d, want 3", m.GetIOPL())
	}
	m.SetIOPL(0)
	if m.GetIOPL() != 0 {
		t.Fatalf("IOPL = %d, want 0", m.GetIOPL())
	}
	m.SetIOPL(0xFF)
	if m.GetIOPL() != 3 {
		t.Fatalf("IOPL = %d, want 3 (masked to 2 bits)", m.GetIOPL())
	}
}

func TestIOPLPreservesOtherFlags(t *testing.T) {
	m := NewMachine(2048)
	m.EnableInterrupts()
	m.SetIOPL(2)
	if !m.AreInterruptsEnabled() {
		t.Fatal("SetIOPL should not clear IF")
	}
}

func TestGetMemoryAndRegisters(t *testing.T) {
	m := NewMachine(2048)
	m.registers[core.W0] = 42
	if m.GetRegisters()[core.W0] != 42 {
		t.Fatal("GetRegisters failed")
	}
	if len(m.GetMemory()) != 2048+wRegisterSaveAreaBytes {
		t.Fatal("GetMemory length mismatch")
	}
}

func TestLoadProgram(t *testing.T) {
	m := NewMachine(2048)
	data := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	m.LoadProgram(data)
	for i, b := range data {
		if m.memory[i] != b {
			t.Fatalf("memory[%d] = 0x%X, want 0x%X", i, m.memory[i], b)
		}
	}
}

func TestDisk(t *testing.T) {
	m := NewMachine(2048)
	m.InitDisk()
	m.AddToDisk([]byte{1, 2, 3})
	m.AddToDisk([]byte{4, 5})
	if len(m.disk.programs) != 2 {
		t.Fatalf("disk programs = %d, want 2", len(m.disk.programs))
	}
}

func TestHandleFaultNilError(t *testing.T) {
	m := NewMachine(2048)
	if m.handleFault(nil) {
		t.Fatal("handleFault(nil) should return false")
	}
}

func TestHandleFaultNonFaultError(t *testing.T) {
	m := NewMachine(2048)
	captureStdout(func() {
		if !m.handleFault(os.ErrNotExist) {
			t.Fatal("handleFault with non-FaultError should still return true")
		}
	})
}

func TestUserModeCannotAccessPrivilegedMemory(t *testing.T) {
	m := NewMachine(4096)
	m.mmu.Mode = ModeFlat
	m.privMode = UserPrivilege
	err := m.memoryAccessFault(m.userMemoryLimit, false)
	if err == nil {
		t.Fatal("user mode should not access privileged memory")
	}
}

func TestKernelModeCanAccessAllMemory(t *testing.T) {
	m := NewMachine(4096)
	m.mmu.Mode = ModeFlat
	m.privMode = KernelPrivilege
	err := m.memoryAccessFault(m.userMemoryLimit, false)
	if err != nil {
		t.Fatalf("kernel mode should access all memory, got %v", err)
	}
}

func TestMemoryAccessFaultBelowLimit(t *testing.T) {
	m := NewMachine(4096)
	m.privMode = UserPrivilege
	err := m.memoryAccessFault(0, false)
	if err != nil {
		t.Fatalf("user mode should access addr 0, got %v", err)
	}
}

func TestSetPageDirectoryBase(t *testing.T) {
	m := NewMachine(2048)
	m.SetPageDirectoryBase(0x12345678)
	if got := m.GetPageDirectoryBase(); got != 0x12345000 {
		t.Fatalf("PDBR = 0x%X, want 0x12345000 (masked)", got)
	}
}

func TestSetPageFaultAddress(t *testing.T) {
	m := NewMachine(2048)
	m.SetPageFaultAddress(0xDEADBEEF)
	if m.GetPageFaultAddress() != 0xDEADBEEF {
		t.Fatalf("CR2 = 0x%X, want 0xDEADBEEF", m.GetPageFaultAddress())
	}
}

func TestGetOpcode(t *testing.T) {
	m := NewMachine(2048)
	inst := uint32(core.HALT_OPCODE) << 26
	if m.getOpcode(inst) != core.HALT_OPCODE {
		t.Fatalf("opcode = %d, want %d", m.getOpcode(inst), core.HALT_OPCODE)
	}
}

func TestDecodeRTypeAllFields(t *testing.T) {
	m := NewMachine(2048)
	inst := uint32(core.ADD_OPCODE)<<26 |
		uint32(core.W3)<<21 |
		uint32(core.W4)<<16 |
		uint32(core.W5)<<11 |
		uint32(0x1F)<<6 |
		uint32(0x3F)
	decoded := m.decodeRTypeInst(inst)
	if decoded.rd != core.W3 {
		t.Fatalf("rd = %d, want W3", decoded.rd)
	}
	if decoded.rs1 != core.W4 {
		t.Fatalf("rs1 = %d, want W4", decoded.rs1)
	}
	if decoded.rs2 != core.W5 {
		t.Fatalf("rs2 = %d, want W5", decoded.rs2)
	}
	if decoded.funct5 != 0x1F {
		t.Fatalf("funct5 = 0x%X, want 0x1F", decoded.funct5)
	}
	if decoded.funct6 != 0x3F {
		t.Fatalf("funct6 = 0x%X, want 0x3F", decoded.funct6)
	}
}

func TestDecodeITypeAllFields(t *testing.T) {
	m := NewMachine(2048)
	inst := uint32(core.MV_OPCODE)<<26 |
		uint32(core.W5)<<21 |
		uint32(0xABCD)<<5 |
		uint32(0x1F)
	decoded := m.decodeITypeInst(inst)
	if decoded.rd != core.W5 {
		t.Fatalf("rd = %d, want W5", decoded.rd)
	}
	if decoded.immediate != 0xABCD {
		t.Fatalf("immediate = 0x%X, want 0xABCD", decoded.immediate)
	}
	if decoded.funct5 != 0x1F {
		t.Fatalf("funct5 = 0x%X, want 0x1F", decoded.funct5)
	}
}

func TestFetchFaultResetsPC(t *testing.T) {
	m := NewMachine(2048)
	m.mmu.Mode = ModeFlat
	m.registers[core.PC] = 0xFFFF
	captureStdout(func() { m.fetch() })
	expected := m.evt[core.EXC_MEMORY_VIOLATION]
	if m.registers[core.PC] != expected {
		t.Fatalf("PC = 0x%X, want 0x%X (handler address)", m.registers[core.PC], expected)
	}
	if m.registers[core.LR] != 0xFFFF {
		t.Fatalf("LR = 0x%X, want 0xFFFF (saved PC)", m.registers[core.LR])
	}
}
