package machine

import (
	"bytes"
	"os"
	"testing"
	"zepa-machine/core"
)

func TestFetch(t *testing.T) {
	machine := NewMachine(2048)
	machine.mmu.Mode = ModeFlat
	machine.memory[0] = 0b00111100
	machine.memory[1] = 0b01000011
	machine.memory[2] = 0b00001000
	machine.memory[3] = 0b00000000
	machine.fetch()

	expectedInstruction := uint32(0b00111100010000110000100000000000)
	if machine.registers[core.IR] != expectedInstruction {
		t.Errorf("Expected 0b%032b, but got 0b%032b", expectedInstruction, machine.registers[core.IR])
	}

	if machine.registers[core.PC] != 4 {
		t.Errorf("Expected PC to be 4, but got %d", machine.registers[core.PC])
	}
}

func TestDecode(t *testing.T) {
	machine := NewMachine(2048)
	machine.mmu.Mode = ModeFlat // Desabilitar MMU para teste de decodificação
	machine.memory[0] = 0b00111100
	machine.memory[1] = 0b01000011
	machine.memory[2] = 0b00001000
	machine.memory[3] = 0b00000000
	machine.fetch()

	decodedInstruction := machine.decode()

	if decodedInstruction.rd != core.Register(2) {
		t.Errorf("Expected rd to be 2, but got %d", decodedInstruction.rd)
	}
	if decodedInstruction.rs1 != core.Register(3) {
		t.Errorf("Expected rs1 to be 3, but got %d", decodedInstruction.rs1)
	}
	if decodedInstruction.rs2 != core.Register(1) {
		t.Errorf("Expected rs2 to be 1, but got %d", decodedInstruction.rs2)
	}
	if decodedInstruction.funct5 != 0 {
		t.Errorf("Expected funct5 to be 0, but got %d", decodedInstruction.funct5)
	}
	if decodedInstruction.funct6 != 0 {
		t.Errorf("Expected funct6 to be 0, but got %d", decodedInstruction.funct6)
	}
}

func TestMV(t *testing.T) {
	machine := NewMachine(2048)
	inst := Instruction{opcode: (*Machine).mv, rd: core.W0, immediate: 0xFF}
	machine.execute(inst)

	if machine.registers[core.W0] != 0xFF {
		t.Errorf("Expected w0 to be 255, got %d", machine.registers[core.W0])
	}
}

func TestADD(t *testing.T) {
	machine := NewMachine(2048)
	machine.registers[core.W1] = 66
	machine.registers[core.W2] = 3000
	inst := Instruction{opcode: (*Machine).add, rd: core.W0, rs1: core.W1, rs2: core.W2}
	machine.execute(inst)

	if machine.registers[core.W0] != 3066 {
		t.Errorf("Expected w0 to be 3066, got %d", machine.registers[core.W0])
	}
}

func TestSUB(t *testing.T) {
	machine := NewMachine(2048)
	machine.registers[core.W1] = 30
	machine.registers[core.W2] = 10
	inst := Instruction{opcode: (*Machine).sub, rd: core.W0, rs1: core.W1, rs2: core.W2}
	machine.execute(inst)

	if machine.registers[core.W0] != 20 {
		t.Errorf("Expected w0 to be 20, got %d", machine.registers[core.W0])
	}
}

func TestJUMP(t *testing.T) {
	machine := NewMachine(2048)
	inst := Instruction{opcode: (*Machine).jump, immediate: 0xA}
	machine.execute(inst)

	if machine.registers[core.PC] != 0xA {
		t.Errorf("Expected pc to be 10, got %d", machine.registers[core.PC])
	}
}

func TestLOAD(t *testing.T) {
	machine := NewMachine(2048)
	machine.mmu.Mode = ModeFlat
	machine.memory[256] = 42
	inst := Instruction{opcode: (*Machine).load, rd: core.W1, immediate: 256}
	machine.execute(inst)

	if machine.registers[core.W1] != 42 {
		t.Errorf("Expected w1 to be 42, got %d", machine.registers[core.W1])
	}
}

func TestSTORE(t *testing.T) {
	machine := NewMachine(2048)
	machine.mmu.Mode = ModeFlat
	machine.registers[core.W1] = 65
	inst := Instruction{opcode: (*Machine).store, rd: core.W1, immediate: 100}
	machine.execute(inst)

	if machine.memory[100] != 65 {
		t.Errorf("Expected memory value to be 65, got %d", machine.memory[100])
	}
}

func TestBEQ(t *testing.T) {
	machine := NewMachine(2048)
	cmpInst := Instruction{opcode: (*Machine).cmp, rs1: core.W0, rs2: core.W1}
	inst := Instruction{opcode: (*Machine).beq, immediate: 0xA}

	machine.execute(cmpInst)
	machine.execute(inst)

	if machine.registers[core.PC] != 0xA {
		t.Errorf("Expected pc to be 10, got %d", machine.registers[core.PC])
	}

	machine.registers[core.PC] = 0
	machine.registers[core.W0] = 1

	machine.execute(cmpInst)
	machine.execute(inst)

	if machine.registers[core.PC] != 0 {
		t.Errorf("Expected pc to be 0, got %d", machine.registers[core.PC])
	}

	machine.registers[core.PC] = 0
	machine.registers[core.W1] = 0

	machine.execute(cmpInst)
	machine.execute(inst)

	if machine.registers[core.PC] != 0 {
		t.Errorf("Expected pc to be 0, got %d", machine.registers[core.PC])
	}
}

func TestBLT(t *testing.T) {
	machine := NewMachine(2048)
	machine.registers[core.W1] = 1
	cmpInst := Instruction{opcode: (*Machine).cmp, rs1: core.W0, rs2: core.W1}
	inst := Instruction{opcode: (*Machine).blt, immediate: 0xA}

	machine.execute(cmpInst)
	machine.execute(inst)

	if machine.registers[core.PC] != 0xA {
		t.Errorf("Expected pc to be 10, got %d", machine.registers[core.PC])
	}

	machine.registers[core.PC] = 0
	machine.registers[core.W0] = 1

	machine.execute(cmpInst)
	machine.execute(inst)

	if machine.registers[core.PC] != 0 {
		t.Errorf("Expected pc to be 0, got %d", machine.registers[core.PC])
	}

	machine.registers[core.PC] = 0
	machine.registers[core.W0] = 1
	machine.registers[core.W1] = 0

	machine.execute(cmpInst)
	machine.execute(inst)

	if machine.registers[core.PC] != 0 {
		t.Errorf("Expected pc to be 0, got %d", machine.registers[core.PC])
	}
}

func TestBGT(t *testing.T) {
	machine := NewMachine(2048)
	machine.registers[core.W0] = 1
	cmpInst := Instruction{opcode: (*Machine).cmp, rs1: core.W0, rs2: core.W1}
	inst := Instruction{opcode: (*Machine).bgt, immediate: 0xA}

	machine.execute(cmpInst)
	machine.execute(inst)

	if machine.registers[core.PC] != 0xA {
		t.Errorf("Expected pc to be 10, got %d", machine.registers[core.PC])
	}

	machine.registers[core.PC] = 0
	machine.registers[core.W0] = 0

	machine.execute(cmpInst)
	machine.execute(inst)

	if machine.registers[core.PC] != 0 {
		t.Errorf("Expected pc to be 0, got %d", machine.registers[core.PC])
	}

	machine.registers[core.PC] = 0
	machine.registers[core.W0] = 0
	machine.registers[core.W1] = 1

	machine.execute(cmpInst)
	machine.execute(inst)

	if machine.registers[core.PC] != 0 {
		t.Errorf("Expected pc to be 0, got %d", machine.registers[core.PC])
	}
}

func TestRET(t *testing.T) {
	machine := NewMachine(2048)
	inst := Instruction{opcode: (*Machine).ret}

	machine.execute(inst)

	if machine.registers[core.PC] != 0 {
		t.Errorf("Expected pc to be 0, got %d", machine.registers[core.PC])
	}
}

func TestUndefinedException(t *testing.T) {
	machine := NewMachine(2048)
	inst := Instruction{opcode: (*Machine).udf}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	machine.execute(inst)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	expected := "Exception raised: Code 0\n"

	if got != expected {
		t.Errorf("Expected: %s, got %s", expected, got)
	}
}

func TestSegmentFaultException(t *testing.T) {
	machine := NewMachine(2048)
	inst := Instruction{opcode: (*Machine).load, immediate: 0xFFFF}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	machine.execute(inst)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	expected := "Exception raised: Code 2\n"

	if got != expected {
		t.Errorf("Expected: %s, got %s", expected, got)
	}
}

func TestTranslateValid(t *testing.T) {
	machine := NewMachine(2048)
	pa, err := machine.translate(100, Read, KernelPrivilege)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if pa != 100 {
		t.Errorf("Expected PA 100, got %d", pa)
	}
}

func TestTranslateInvalidLimit(t *testing.T) {
	machine := NewMachine(2048)
	machine.mmu.Mode = ModeSegmented
	_, err := machine.translate(0x4000, Read, KernelPrivilege)
	if err == nil {
		t.Error("Expected error for limit violation")
	}
}

func TestFetchWithMMU(t *testing.T) {
	machine := NewMachine(2048)
	machine.mmu.Mode = ModeSegmented
	machine.memory[0] = 0b00110100
	machine.fetch()
}

func TestLoadWithSegmentationFault(t *testing.T) {
	machine := NewMachine(2048)
	machine.mmu.Mode = ModeSegmented
	inst := Instruction{opcode: (*Machine).load, rd: core.W1, immediate: 0x4000}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	machine.execute(inst)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	expected := "Exception raised: Code 2\n"

	if got != expected {
		t.Errorf("Expected: %s, got %s", expected, got)
	}
}

func TestPageFaultException(t *testing.T) {
	machine := NewMachine(2048)
	machine.mmu.Mode = ModeFlat
	machine.EnablePaging()
	machine.SetPageDirectoryBase(0x400)
	writeUint32LE(machine.memory, 0x400, EncodePDE(PageDirectoryEntry{Present: false}))
	inst := Instruction{opcode: (*Machine).load, immediate: 0x100}

	// Get default exit
	old := os.Stdout // Save stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	machine.execute(inst)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	expected := "Exception raised: Code 3\n"

	if got != expected {
		t.Errorf("Expected: %s, got %s", expected, got)
	}
}

func TestProtectionFaultException(t *testing.T) {
	machine := NewMachine(2048)
	machine.registers[core.W1] = 99
	inst := Instruction{opcode: (*Machine).store, rd: core.W1, immediate: 2048}

	// Get default exit
	old := os.Stdout // Save stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	machine.execute(inst)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	expected := "Exception raised: Code 4\n"

	if got != expected {
		t.Errorf("Expected: %s, got %s", expected, got)
	}
}
