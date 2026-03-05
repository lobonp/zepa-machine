package assembler

import (
	"strings"
	"testing"
	"zepa-machine/core"
)

func runAssemblerFromReader(t *testing.T, source string) []byte {
	t.Helper()
	instrs, err := LoadAssemblyFromReader(strings.NewReader(source))
	if err != nil {
		t.Fatalf("LoadAssemblyFromReader error: %v", err)
	}
	bin, err := ConvertInstructionsToBinary(instrs)
	if err != nil {
		t.Fatalf("ConvertInstructionsToBinary error: %v", err)
	}
	return bin
}

func instructionWord(b []byte, index int) uint32 {
	off := index * 4
	return uint32(b[off])<<24 | uint32(b[off+1])<<16 | uint32(b[off+2])<<8 | uint32(b[off+3])
}

func TestParseRegisterAllValid(t *testing.T) {
	regs := map[string]core.Register{
		"W0": core.W0, "W1": core.W1, "W2": core.W2,
		"W3": core.W3, "W4": core.W4, "W5": core.W5,
		"PC": core.PC, "SP": core.SP, "IR": core.IR,
		"SR": core.SR, "MDR": core.MDR, "MAR": core.MAR,
		"LR": core.LR, "SSR": core.SSR,
		"CR0": core.CR0, "CR2": core.CR2, "CR3": core.CR3, "CR4": core.CR4,
		"EFLAGS": core.EFLAGS,
	}
	for name, expected := range regs {
		val, err := parseRegister(name)
		if err != nil {
			t.Fatalf("parseRegister(%s) error: %v", name, err)
		}
		if val != byte(expected) {
			t.Fatalf("parseRegister(%s) = %d, want %d", name, val, expected)
		}
	}
}

func TestParseRegisterLowercase(t *testing.T) {
	val, err := parseRegister("w0")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if val != byte(core.W0) {
		t.Fatalf("got %d, want %d", val, core.W0)
	}
}

func TestParseRegisterInvalid(t *testing.T) {
	invalids := []string{"W6", "R0", "AX", "", "123", "WW"}
	for _, r := range invalids {
		_, err := parseRegister(r)
		if err == nil {
			t.Fatalf("expected error for register '%s'", r)
		}
	}
}

func TestParseImmediateDecimal(t *testing.T) {
	cases := []struct {
		in  string
		out uint16
	}{
		{"0", 0},
		{"1", 1},
		{"255", 255},
		{"65535", 65535},
		{"#100", 100},
		{"#0", 0},
	}
	for _, c := range cases {
		val, err := parseImmediate(c.in)
		if err != nil {
			t.Fatalf("parseImmediate(%s) error: %v", c.in, err)
		}
		if val != c.out {
			t.Fatalf("parseImmediate(%s) = %d, want %d", c.in, val, c.out)
		}
	}
}

func TestParseImmediateHex(t *testing.T) {
	cases := []struct {
		in  string
		out uint16
	}{
		{"0x0", 0},
		{"0xFF", 255},
		{"0xFFFF", 65535},
		{"#0x10", 16},
		{"0X1A", 26},
	}
	for _, c := range cases {
		val, err := parseImmediate(c.in)
		if err != nil {
			t.Fatalf("parseImmediate(%s) error: %v", c.in, err)
		}
		if val != c.out {
			t.Fatalf("parseImmediate(%s) = %d, want %d", c.in, val, c.out)
		}
	}
}

func TestParseImmediateOverflow(t *testing.T) {
	_, err := parseImmediate("65536")
	if err == nil {
		t.Fatal("expected error for 65536")
	}
	_, err = parseImmediate("0x10000")
	if err == nil {
		t.Fatal("expected error for 0x10000")
	}
}

func TestParseImmediateInvalid(t *testing.T) {
	invalids := []string{"abc", "W0", "-1", "12.5", ""}
	for _, s := range invalids {
		_, err := parseImmediate(s)
		if err == nil {
			t.Fatalf("expected error for '%s'", s)
		}
	}
}

func TestProcessLineComment(t *testing.T) {
	if processLine("; full comment") != "" {
		t.Fatal("full comment should be empty")
	}
	if processLine("MV W0, 10 ; inline comment") != "MV W0 10" {
		t.Fatalf("got '%s'", processLine("MV W0, 10 ; inline comment"))
	}
}

func TestProcessLineLabel(t *testing.T) {
	if processLine("loop:") != "" {
		t.Fatal("label should be empty")
	}
	if processLine("  start:  ") != "" {
		t.Fatal("label with spaces should be empty")
	}
}

func TestProcessLineWhitespace(t *testing.T) {
	if processLine("") != "" {
		t.Fatal("empty line should be empty")
	}
	if processLine("   ") != "" {
		t.Fatal("whitespace-only should be empty")
	}
}

func TestProcessLineRemovesCommas(t *testing.T) {
	result := processLine("ADD W0, W1, W2")
	if result != "ADD W0 W1 W2" {
		t.Fatalf("got '%s'", result)
	}
}

func TestLoadAssemblyFromReaderEmpty(t *testing.T) {
	instrs, err := LoadAssemblyFromReader(strings.NewReader(""))
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(instrs) != 0 {
		t.Fatalf("got %d instructions, want 0", len(instrs))
	}
}

func TestLoadAssemblyFromReaderOnlyComments(t *testing.T) {
	src := "; comment1\n; comment2\n"
	instrs, err := LoadAssemblyFromReader(strings.NewReader(src))
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(instrs) != 0 {
		t.Fatalf("got %d instructions, want 0", len(instrs))
	}
}

func TestLoadAssemblyFromReaderOnlyLabels(t *testing.T) {
	src := "start:\nloop:\n"
	instrs, err := LoadAssemblyFromReader(strings.NewReader(src))
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(instrs) != 0 {
		t.Fatalf("got %d instructions, want 0", len(instrs))
	}
}

func TestLoadAssemblyFromReaderParsesInstructions(t *testing.T) {
	src := "MV W0, 10\nADD W1, W2, W3\nHALT\n"
	instrs, err := LoadAssemblyFromReader(strings.NewReader(src))
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(instrs) != 3 {
		t.Fatalf("got %d instructions, want 3", len(instrs))
	}
	if instrs[0][0] != "MV" || instrs[0][1] != "W0" || instrs[0][2] != "10" {
		t.Fatalf("unexpected first instruction: %v", instrs[0])
	}
	if instrs[2][0] != "HALT" {
		t.Fatalf("unexpected third instruction: %v", instrs[2])
	}
}

func TestConvertInstructionToBinaryEmpty(t *testing.T) {
	_, err := ConvertInstructionToBinary([]string{})
	if err == nil {
		t.Fatal("expected error for empty instruction")
	}
}

func TestConvertInstructionToBinaryInvalidOpcode(t *testing.T) {
	_, err := ConvertInstructionToBinary([]string{"INVALID"})
	if err == nil {
		t.Fatal("expected error for invalid opcode")
	}
}

func TestConvertInstructionToBinaryHalt(t *testing.T) {
	b, err := ConvertInstructionToBinary([]string{"HALT"})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	word := instructionWord(b, 0)
	opcode := (word >> 26) & 0x3F
	if opcode != uint32(core.HALT_OPCODE) {
		t.Fatalf("opcode = %d, want %d", opcode, core.HALT_OPCODE)
	}
}

func TestConvertInstructionToBinaryRet(t *testing.T) {
	b, err := ConvertInstructionToBinary([]string{"RET"})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	word := instructionWord(b, 0)
	opcode := (word >> 26) & 0x3F
	if opcode != uint32(core.RET_OPCODE) {
		t.Fatalf("opcode = %d, want %d", opcode, core.RET_OPCODE)
	}
}

func TestConvertInstructionToBinaryUdf(t *testing.T) {
	b, err := ConvertInstructionToBinary([]string{"UDF"})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	word := instructionWord(b, 0)
	opcode := (word >> 26) & 0x3F
	if opcode != uint32(core.UDF_OPCODE) {
		t.Fatalf("opcode = %d, want %d", opcode, core.UDF_OPCODE)
	}
}

func TestEncodeITypeJump(t *testing.T) {
	b, err := ConvertInstructionToBinary([]string{"JUMP", "42"})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	word := instructionWord(b, 0)
	opcode := (word >> 26) & 0x3F
	imm := (word >> 5) & 0xFFFF
	if opcode != uint32(core.JUMP_OPCODE) {
		t.Fatalf("opcode = %d, want %d", opcode, core.JUMP_OPCODE)
	}
	if imm != 42 {
		t.Fatalf("imm = %d, want 42", imm)
	}
}

func TestEncodeITypeMv(t *testing.T) {
	b, err := ConvertInstructionToBinary([]string{"MV", "W3", "255"})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	word := instructionWord(b, 0)
	opcode := (word >> 26) & 0x3F
	rd := (word >> 21) & 0x1F
	imm := (word >> 5) & 0xFFFF
	if opcode != uint32(core.MV_OPCODE) {
		t.Fatalf("opcode = %d, want %d", opcode, core.MV_OPCODE)
	}
	if rd != uint32(core.W3) {
		t.Fatalf("rd = %d, want %d", rd, core.W3)
	}
	if imm != 255 {
		t.Fatalf("imm = %d, want 255", imm)
	}
}

func TestEncodeITypeMvMaxImm(t *testing.T) {
	b, err := ConvertInstructionToBinary([]string{"MV", "W0", "65535"})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	word := instructionWord(b, 0)
	imm := (word >> 5) & 0xFFFF
	if imm != 65535 {
		t.Fatalf("imm = %d, want 65535", imm)
	}
}

func TestEncodeITypeMvZeroImm(t *testing.T) {
	b, err := ConvertInstructionToBinary([]string{"MV", "W0", "0"})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	word := instructionWord(b, 0)
	imm := (word >> 5) & 0xFFFF
	if imm != 0 {
		t.Fatalf("imm = %d, want 0", imm)
	}
}

func TestEncodeITypeLoad(t *testing.T) {
	b, err := ConvertInstructionToBinary([]string{"LOAD", "W1", "100"})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	word := instructionWord(b, 0)
	opcode := (word >> 26) & 0x3F
	rd := (word >> 21) & 0x1F
	imm := (word >> 5) & 0xFFFF
	if opcode != uint32(core.LOAD_OPCODE) {
		t.Fatalf("opcode = %d", opcode)
	}
	if rd != uint32(core.W1) {
		t.Fatalf("rd = %d", rd)
	}
	if imm != 100 {
		t.Fatalf("imm = %d", imm)
	}
}

func TestEncodeITypeStore(t *testing.T) {
	b, err := ConvertInstructionToBinary([]string{"STORE", "W2", "200"})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	word := instructionWord(b, 0)
	opcode := (word >> 26) & 0x3F
	rd := (word >> 21) & 0x1F
	imm := (word >> 5) & 0xFFFF
	if opcode != uint32(core.STORE_OPCODE) {
		t.Fatalf("opcode = %d", opcode)
	}
	if rd != uint32(core.W2) {
		t.Fatalf("rd = %d", rd)
	}
	if imm != 200 {
		t.Fatalf("imm = %d", imm)
	}
}

func TestEncodeITypeBranches(t *testing.T) {
	for _, op := range []string{"BEQ", "BLT", "BGT"} {
		b, err := ConvertInstructionToBinary([]string{op, "10"})
		if err != nil {
			t.Fatalf("%s error: %v", op, err)
		}
		word := instructionWord(b, 0)
		expected := opcodeMap[op]
		got := core.Opcode((word >> 26) & 0x3F)
		if got != expected {
			t.Fatalf("%s opcode = %d, want %d", op, got, expected)
		}
		imm := (word >> 5) & 0xFFFF
		if imm != 10 {
			t.Fatalf("%s imm = %d, want 10", op, imm)
		}
	}
}

func TestEncodeITypeD2M(t *testing.T) {
	b, err := ConvertInstructionToBinary([]string{"D2M", "W0", "500"})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	word := instructionWord(b, 0)
	opcode := (word >> 26) & 0x3F
	if opcode != uint32(core.DISK2MEM_OPCODE) {
		t.Fatalf("opcode = %d, want %d", opcode, core.DISK2MEM_OPCODE)
	}
}

func TestEncodeRTypeAdd3Operands(t *testing.T) {
	b, err := ConvertInstructionToBinary([]string{"ADD", "W0", "W1", "W2"})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	word := instructionWord(b, 0)
	opcode := (word >> 26) & 0x3F
	rd := (word >> 21) & 0x1F
	rs1 := (word >> 16) & 0x1F
	rs2 := (word >> 11) & 0x1F
	if opcode != uint32(core.ADD_OPCODE) {
		t.Fatalf("opcode = %d", opcode)
	}
	if rd != uint32(core.W0) {
		t.Fatalf("rd = %d", rd)
	}
	if rs1 != uint32(core.W1) {
		t.Fatalf("rs1 = %d", rs1)
	}
	if rs2 != uint32(core.W2) {
		t.Fatalf("rs2 = %d", rs2)
	}
}

func TestEncodeRTypeAdd2Operands(t *testing.T) {
	b, err := ConvertInstructionToBinary([]string{"ADD", "W3", "W4"})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	word := instructionWord(b, 0)
	rd := (word >> 21) & 0x1F
	rs1 := (word >> 16) & 0x1F
	rs2 := (word >> 11) & 0x1F
	if rd != 0 {
		t.Fatalf("rd = %d, want 0", rd)
	}
	if rs1 != uint32(core.W3) {
		t.Fatalf("rs1 = %d", rs1)
	}
	if rs2 != uint32(core.W4) {
		t.Fatalf("rs2 = %d", rs2)
	}
}

func TestEncodeRTypeSub(t *testing.T) {
	b, err := ConvertInstructionToBinary([]string{"SUB", "W1", "W2", "W3"})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	word := instructionWord(b, 0)
	opcode := (word >> 26) & 0x3F
	if opcode != uint32(core.SUB_OPCODE) {
		t.Fatalf("opcode = %d", opcode)
	}
}

func TestEncodeRTypeCmp(t *testing.T) {
	b, err := ConvertInstructionToBinary([]string{"CMP", "W0", "W1"})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	word := instructionWord(b, 0)
	opcode := (word >> 26) & 0x3F
	if opcode != uint32(core.CMP_OPCODE) {
		t.Fatalf("opcode = %d", opcode)
	}
}

func TestEncodeRTypeWrongOperandCount(t *testing.T) {
	_, err := ConvertInstructionToBinary([]string{"ADD", "W0"})
	if err == nil {
		t.Fatal("expected error for 1 operand")
	}
	_, err = ConvertInstructionToBinary([]string{"ADD", "W0", "W1", "W2", "W3"})
	if err == nil {
		t.Fatal("expected error for 4 operands")
	}
}

func TestEncodeITypeWrongOperandCount(t *testing.T) {
	_, err := ConvertInstructionToBinary([]string{"MV", "W0", "10", "extra"})
	if err == nil {
		t.Fatal("expected error for 3 operands on I-Type")
	}
	_, err = ConvertInstructionToBinary([]string{"JUMP"})
	if err == nil {
		t.Fatal("expected error for JUMP with 0 operands")
	}
}

func TestEncodeRTypeLDLoadr(t *testing.T) {
	b, err := ConvertInstructionToBinary([]string{"LOADR", "W0", "W1"})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	word := instructionWord(b, 0)
	opcode := (word >> 26) & 0x3F
	rd := (word >> 21) & 0x1F
	rs1 := (word >> 16) & 0x1F
	if opcode != uint32(core.LOADR_OPCODE) {
		t.Fatalf("opcode = %d", opcode)
	}
	if rd != uint32(core.W0) {
		t.Fatalf("rd = %d", rd)
	}
	if rs1 != uint32(core.W1) {
		t.Fatalf("rs1 = %d", rs1)
	}
}

func TestEncodeRTypeLDStorer(t *testing.T) {
	b, err := ConvertInstructionToBinary([]string{"STORER", "W2", "W3"})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	word := instructionWord(b, 0)
	opcode := (word >> 26) & 0x3F
	if opcode != uint32(core.STORER_OPCODE) {
		t.Fatalf("opcode = %d", opcode)
	}
}

func TestEncodeRTypeLDWrongOperandCount(t *testing.T) {
	_, err := ConvertInstructionToBinary([]string{"LOADR", "W0"})
	if err == nil {
		t.Fatal("expected error for 1 operand")
	}
	_, err = ConvertInstructionToBinary([]string{"LOADR", "W0", "W1", "W2"})
	if err == nil {
		t.Fatal("expected error for 3 operands")
	}
}

func TestEncodeITypeInvalidRegister(t *testing.T) {
	_, err := ConvertInstructionToBinary([]string{"MV", "R0", "10"})
	if err == nil {
		t.Fatal("expected error for invalid register")
	}
}

func TestEncodeITypeInvalidImmediate(t *testing.T) {
	_, err := ConvertInstructionToBinary([]string{"MV", "W0", "abc"})
	if err == nil {
		t.Fatal("expected error for non-numeric immediate")
	}
}

func TestEncodeRTypeInvalidRegister(t *testing.T) {
	_, err := ConvertInstructionToBinary([]string{"ADD", "X0", "W1", "W2"})
	if err == nil {
		t.Fatal("expected error for invalid register")
	}
}

func TestEncodeITypeHexImmediate(t *testing.T) {
	b, err := ConvertInstructionToBinary([]string{"MV", "W0", "0xFF"})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	word := instructionWord(b, 0)
	imm := (word >> 5) & 0xFFFF
	if imm != 0xFF {
		t.Fatalf("imm = 0x%X, want 0xFF", imm)
	}
}

func TestEncodeITypeHashImmediate(t *testing.T) {
	b, err := ConvertInstructionToBinary([]string{"MV", "W0", "#50"})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	word := instructionWord(b, 0)
	imm := (word >> 5) & 0xFFFF
	if imm != 50 {
		t.Fatalf("imm = %d, want 50", imm)
	}
}

func TestConvertInstructionsToBinaryMultiple(t *testing.T) {
	instrs := [][]string{
		{"MV", "W0", "5"},
		{"MV", "W1", "10"},
		{"ADD", "W2", "W0", "W1"},
		{"HALT"},
	}
	bin, err := ConvertInstructionsToBinary(instrs)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(bin) != 16 {
		t.Fatalf("len = %d, want 16", len(bin))
	}
	for i := 0; i < 4; i++ {
		w := instructionWord(bin, i)
		op := (w >> 26) & 0x3F
		switch i {
		case 0, 1:
			if op != uint32(core.MV_OPCODE) {
				t.Fatalf("instr %d opcode = %d, want MV", i, op)
			}
		case 2:
			if op != uint32(core.ADD_OPCODE) {
				t.Fatalf("instr %d opcode = %d, want ADD", i, op)
			}
		case 3:
			if op != uint32(core.HALT_OPCODE) {
				t.Fatalf("instr %d opcode = %d, want HALT", i, op)
			}
		}
	}
}

func TestConvertInstructionsToBinaryWithError(t *testing.T) {
	instrs := [][]string{
		{"MV", "W0", "5"},
		{"INVALID"},
	}
	_, err := ConvertInstructionsToBinary(instrs)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRunAssemblerFromReaderHaltOnly(t *testing.T) {
	bin := runAssemblerFromReader(t, "HALT\n")
	if len(bin) != 4 {
		t.Fatalf("len = %d, want 4", len(bin))
	}
	op := (instructionWord(bin, 0) >> 26) & 0x3F
	if op != uint32(core.HALT_OPCODE) {
		t.Fatalf("opcode = %d", op)
	}
}

func TestRunAssemblerFromReaderMvAndHalt(t *testing.T) {
	bin := runAssemblerFromReader(t, "MV W0, 42\nHALT\n")
	if len(bin) != 8 {
		t.Fatalf("len = %d, want 8", len(bin))
	}
	w := instructionWord(bin, 0)
	imm := (w >> 5) & 0xFFFF
	if imm != 42 {
		t.Fatalf("imm = %d, want 42", imm)
	}
}

func TestRunAssemblerFromReaderWithComments(t *testing.T) {
	src := "; setup\nMV W0, 1 ; load one\n; end\nHALT\n"
	bin := runAssemblerFromReader(t, src)
	if len(bin) != 8 {
		t.Fatalf("len = %d, want 8", len(bin))
	}
}

func TestRunAssemblerFromReaderWithLabels(t *testing.T) {
	src := "start:\nMV W0, 5\nloop:\nHALT\n"
	bin := runAssemblerFromReader(t, src)
	if len(bin) != 8 {
		t.Fatalf("len = %d, want 8", len(bin))
	}
}

func TestRunAssemblerFromReaderAllITypeOpcodes(t *testing.T) {
	iTypes := map[string]core.Opcode{
		"MV W0, 1":    core.MV_OPCODE,
		"JUMP 5":      core.JUMP_OPCODE,
		"LOAD W0, 10": core.LOAD_OPCODE,
		"STORE W0, 5": core.STORE_OPCODE,
		"HALT":        core.HALT_OPCODE,
		"RET":         core.RET_OPCODE,
		"BEQ 3":       core.BEQ_OPCODE,
		"BLT 4":       core.BLT_OPCODE,
		"BGT 5":       core.BGT_OPCODE,
		"UDF":         core.UDF_OPCODE,
		"D2M W0, 1":   core.DISK2MEM_OPCODE,
	}
	for src, expected := range iTypes {
		bin := runAssemblerFromReader(t, src+"\n")
		w := instructionWord(bin, 0)
		op := core.Opcode((w >> 26) & 0x3F)
		if op != expected {
			t.Fatalf("%s: opcode = %d, want %d", src, op, expected)
		}
	}
}

func TestRunAssemblerFromReaderAllRTypeOpcodes(t *testing.T) {
	rTypes := map[string]core.Opcode{
		"ADD W0, W1, W2": core.ADD_OPCODE,
		"SUB W0, W1, W2": core.SUB_OPCODE,
		"CMP W0, W1":     core.CMP_OPCODE,
	}
	for src, expected := range rTypes {
		bin := runAssemblerFromReader(t, src+"\n")
		w := instructionWord(bin, 0)
		op := core.Opcode((w >> 26) & 0x3F)
		if op != expected {
			t.Fatalf("%s: opcode = %d, want %d", src, op, expected)
		}
	}
}

func TestRunAssemblerFromReaderAllRTypeLDOpcodes(t *testing.T) {
	ldTypes := map[string]core.Opcode{
		"LOADR W0, W1":  core.LOADR_OPCODE,
		"STORER W2, W3": core.STORER_OPCODE,
	}
	for src, expected := range ldTypes {
		bin := runAssemblerFromReader(t, src+"\n")
		w := instructionWord(bin, 0)
		op := core.Opcode((w >> 26) & 0x3F)
		if op != expected {
			t.Fatalf("%s: opcode = %d, want %d", src, op, expected)
		}
	}
}

func TestRunAssemblerFileNotFound(t *testing.T) {
	_, err := RunAssembler("/nonexistent/path.asm")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestRunAssemblerSampleFiles(t *testing.T) {
	samples := []string{
		"../asm/samples/add_and_mv.asm",
		"../asm/samples/add_sub_halt.asm",
		"../asm/samples/add_two_number.asm",
		"../asm/samples/cmp_and_jump.asm",
		"../asm/samples/cmp_two_numbers.asm",
		"../asm/samples/cmp_without_jump.asm",
		"../asm/samples/immediate_halt.asm",
		"../asm/samples/loadr_storer.asm",
		"../asm/samples/memory_rw_.asm",
		"../asm/samples/mov_and_halt.asm",
		"../asm/samples/multiply_two_numbers.asm",
		"../asm/samples/simple_jump.asm",
		"../asm/samples/store_load_halt.asm",
		"../asm/samples/store_load.asm",
		"../asm/samples/sub_cmp_bigger.asm",
		"../asm/samples/paging_enable.asm",
		"../asm/samples/paging_two_frames.asm",
	}
	for _, path := range samples {
		bin, err := RunAssembler(path)
		if err != nil {
			t.Fatalf("RunAssembler(%s) error: %v", path, err)
		}
		if len(bin) == 0 {
			t.Fatalf("RunAssembler(%s) returned empty", path)
		}
		if len(bin)%4 != 0 {
			t.Fatalf("RunAssembler(%s) len=%d not multiple of 4", path, len(bin))
		}
	}
}

func TestEncodeITypeAllRegisters(t *testing.T) {
	regs := []string{"W0", "W1", "W2", "W3", "W4", "W5"}
	for _, r := range regs {
		b, err := ConvertInstructionToBinary([]string{"MV", r, "1"})
		if err != nil {
			t.Fatalf("MV %s, 1 error: %v", r, err)
		}
		word := instructionWord(b, 0)
		rd := (word >> 21) & 0x1F
		expected := RegisterMap[r]
		if rd != uint32(expected) {
			t.Fatalf("MV %s: rd = %d, want %d", r, rd, expected)
		}
	}
}

func TestEncodeRTypeAllRegisterPositions(t *testing.T) {
	b, err := ConvertInstructionToBinary([]string{"ADD", "W5", "W4", "W3"})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	word := instructionWord(b, 0)
	rd := (word >> 21) & 0x1F
	rs1 := (word >> 16) & 0x1F
	rs2 := (word >> 11) & 0x1F
	if rd != uint32(core.W5) || rs1 != uint32(core.W4) || rs2 != uint32(core.W3) {
		t.Fatalf("rd=%d rs1=%d rs2=%d", rd, rs1, rs2)
	}
}

func TestRoundTripMvAddHalt(t *testing.T) {
	src := "MV W0, 5\nMV W1, 3\nADD W2, W0, W1\nHALT\n"
	bin := runAssemblerFromReader(t, src)
	if len(bin) != 16 {
		t.Fatalf("len = %d, want 16", len(bin))
	}

	w0 := instructionWord(bin, 0)
	if (w0>>26)&0x3F != uint32(core.MV_OPCODE) {
		t.Fatal("instr 0 not MV")
	}
	if (w0>>5)&0xFFFF != 5 {
		t.Fatal("instr 0 imm != 5")
	}

	w2 := instructionWord(bin, 2)
	if (w2>>26)&0x3F != uint32(core.ADD_OPCODE) {
		t.Fatal("instr 2 not ADD")
	}

	w3 := instructionWord(bin, 3)
	if (w3>>26)&0x3F != uint32(core.HALT_OPCODE) {
		t.Fatal("instr 3 not HALT")
	}
}

func TestJumpWithOnlyImmediate(t *testing.T) {
	b, err := ConvertInstructionToBinary([]string{"JUMP", "0"})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	word := instructionWord(b, 0)
	imm := (word >> 5) & 0xFFFF
	rd := (word >> 21) & 0x1F
	if imm != 0 {
		t.Fatalf("imm = %d", imm)
	}
	if rd != 0 {
		t.Fatalf("rd = %d, want 0", rd)
	}
}

func TestHaltIgnoresExtraOperands(t *testing.T) {
	b, err := ConvertInstructionToBinary([]string{"HALT"})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	word := instructionWord(b, 0)
	if word != uint32(core.HALT_OPCODE)<<26 {
		t.Fatalf("unexpected HALT encoding: 0x%08X", word)
	}
}

func TestFunct5Funct6DefaultToZero(t *testing.T) {
	b, err := ConvertInstructionToBinary([]string{"ADD", "W0", "W1", "W2"})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	word := instructionWord(b, 0)
	funct5 := (word >> 6) & 0x1F
	funct6 := word & 0x3F
	if funct5 != 0 {
		t.Fatalf("funct5 = %d, want 0", funct5)
	}
	if funct6 != 0 {
		t.Fatalf("funct6 = %d, want 0", funct6)
	}
}

func TestITypeFunct5OverlapsImmediate(t *testing.T) {
	b, err := ConvertInstructionToBinary([]string{"MV", "W0", "0"})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	word := instructionWord(b, 0)
	lowBits := word & 0x1F
	if lowBits != 0 {
		t.Fatalf("low 5 bits = 0x%X, want 0 (funct5 default)", lowBits)
	}
}

func TestCaseSensitivityOpcode(t *testing.T) {
	src := "mv w0, 5\nhalt\n"
	instrs, err := LoadAssemblyFromReader(strings.NewReader(src))
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	bin, err := ConvertInstructionsToBinary(instrs)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(bin) != 8 {
		t.Fatalf("len = %d, want 8", len(bin))
	}
}
