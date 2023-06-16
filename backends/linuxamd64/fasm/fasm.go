package fasm

import (
	mir "github.com/padeir0/pir/backends/linuxamd64/mir"
	mirc "github.com/padeir0/pir/backends/linuxamd64/mir/class"
	FT "github.com/padeir0/pir/backends/linuxamd64/mir/flowkind"
	IT "github.com/padeir0/pir/backends/linuxamd64/mir/instrkind"
	T "github.com/padeir0/pir/types"

	"fmt"
	"strconv"
)

func Generate(P *mir.Program) *FasmProgram {
	output := &fasmProgram{
		executable: []*fasmProc{genWrite(P), genRead(P), genError(P)},
		data:       []*fasmData{},
		entry:      genEntry(P),
	}
	for _, sy := range P.Symbols {
		if !sy.Builtin {
			if sy.Proc != nil {
				proc := genProc(P, sy.Proc)
				output.executable = append(output.executable, proc)
			}
			if sy.Mem != nil {
				mem := genMem(sy.Mem)
				output.data = append(output.data, mem)
			}
		}
	}
	return &FasmProgram{
		Contents: output.String(),
		Name:     P.Name,
	}
}

type FasmProgram struct {
	Name     string
	Contents string
}

type fasmProgram struct {
	entry      []*amd64Instr
	executable []*fasmProc
	data       []*fasmData
}

func (this *fasmProgram) String() string {
	output := "format ELF64 executable 3\n"
	output += "\nsegment readable writable\n"
	for _, decData := range this.data {
		output += decData.String()
	}
	output += "\nsegment readable executable\n"
	output += "entry $\n"
	for _, instr := range this.entry {
		output += "\t" + instr.String() + "\n"
	}
	output += "; ---- end \n"
	for _, fproc := range this.executable {
		output += fproc.String()
	}
	return output
}

type fasmData struct {
	label    string
	content  string
	declared bool
}

func (this *fasmData) String() string {
	if this.declared {
		return this.label + " db " + this.content + "\n"
	}
	return this.label + " rb " + this.content + "\n"
}

type fasmProc struct {
	label  string
	blocks []*fasmBlock // order matters
}

func (this *fasmProc) String() string {
	output := ""
	for _, b := range this.blocks {
		output += b.String() + "\n"
	}
	return output
}

type fasmBlock struct {
	label string
	code  []*amd64Instr // order matters ofc
}

func (this *fasmBlock) String() string {
	output := this.label + ":\n"
	for _, instr := range this.code {
		output += "\t" + instr.String() + "\n"
	}
	return output
}

type InstrType int

type amd64Instr struct {
	Instr string
	Op1   string
	Op2   string
}

func (this *amd64Instr) String() string {
	if this.Instr == "" {
		return "???"
	}
	if this.Op1 == "" {
		return this.Instr
	}
	if this.Op2 == "" {
		return this.Instr + "\t" + this.Op1
	}
	return this.Instr + "\t" + this.Op1 + ", " + this.Op2
}

const (
	Add = "add"
	Sub = "sub"
	Neg = "neg"
	//signed
	IMul = "imul"
	IDiv = "idiv"
	//unsigned
	Mul = "mul"
	Div = "div"

	Xor = "xor"

	Mov   = "mov"
	Movsx = "movsx"
	Push  = "push"
	Pop   = "pop"

	And = "and"
	Or  = "or"

	Cmp = "cmp"

	Sete  = "sete"
	Setne = "setne"
	// signed
	Setg  = "setg"
	Setge = "setge"
	Setl  = "setl"
	Setle = "setle"
	// unsigned
	Seta  = "seta"
	Setae = "setae"
	Setb  = "setb"
	Setbe = "setbe"

	Jmp = "jmp"
	Je  = "je"
	Jne = "jne"
	Jg  = "jg"
	Jge = "jge"
	Jl  = "jl"
	Jle = "jle"

	Call    = "call"
	Syscall = "syscall"
	Ret     = "ret"
)

type register struct {
	QWord string
	DWord string
	Word  string
	Byte  string
}

// we use this three as scratch space, IDIV already needs rax and rdx,
// and very rarely will a block need more than 3~5 registers
var RAX = &register{QWord: "rax", DWord: "eax", Word: "ax", Byte: "al"}
var RDX = &register{QWord: "rdx", DWord: "edx", Word: "dx", Byte: "dl"}
var RBX = &register{QWord: "rbx", DWord: "ebx", Word: "bx", Byte: "bl"}

var Registers = []*register{
	{QWord: "r15", DWord: "r15d", Word: "r15w", Byte: "r15b"},
	{QWord: "r14", DWord: "r14d", Word: "r14w", Byte: "r14b"},
	{QWord: "r13", DWord: "r13d", Word: "r13w", Byte: "r13b"},
	{QWord: "r12", DWord: "r12d", Word: "r12w", Byte: "r12b"},

	{QWord: "r11", DWord: "r11d", Word: "r11w", Byte: "r11b"},
	{QWord: "r10", DWord: "r10d", Word: "r10w", Byte: "r10b"},
	{QWord: "r9", DWord: "r9d", Word: "r9w", Byte: "r9b"},
	{QWord: "r8", DWord: "r8d", Word: "r8w", Byte: "r8b"},

	{QWord: "rdi", DWord: "edi", Word: "di", Byte: "dil"},
	{QWord: "rsi", DWord: "esi", Word: "si", Byte: "sil"},
	{QWord: "rcx", DWord: "ecx", Word: "cx", Byte: "cl"},
}

// read[ptr, int] int
func genRead(P *mir.Program) *fasmProc {
	ptrArg := mir.Operand{Class: mirc.CallerInterproc, Num: 0, Type: T.T_Ptr}
	sizeArg := mir.Operand{Class: mirc.CallerInterproc, Num: 1, Type: T.T_I64}
	ptr := convertOperand(P, ptrArg, 0, 0, 0)
	size := convertOperand(P, sizeArg, 0, 0, 0)
	amountRead := ptr
	return &fasmProc{
		label: "_read",
		blocks: []*fasmBlock{
			{label: "_read", code: []*amd64Instr{
				unary(Push, "rbp"),
				bin(Mov, "rbp", "rsp"),

				bin(Mov, "rdx", size),
				bin(Mov, "rsi", ptr),
				bin(Mov, "rdi", "0"), // STDERR
				bin(Mov, "rax", "0"), // WRITE
				{Instr: Syscall},

				bin(Mov, amountRead, "rax"),

				bin(Mov, "rsp", "rbp"),
				unary(Pop, "rbp"),
				{Instr: Ret},
			}},
		},
	}
}

// write[ptr, int]
func genWrite(P *mir.Program) *fasmProc {
	ptrArg := mir.Operand{Class: mirc.CallerInterproc, Num: 0, Type: T.T_Ptr}
	sizeArg := mir.Operand{Class: mirc.CallerInterproc, Num: 1, Type: T.T_I64}
	ptr := convertOperand(P, ptrArg, 0, 0, 0)
	size := convertOperand(P, sizeArg, 0, 0, 0)
	return &fasmProc{
		label: "_write",
		blocks: []*fasmBlock{
			{label: "_write", code: []*amd64Instr{
				unary(Push, "rbp"),
				bin(Mov, "rbp", "rsp"),

				bin(Mov, "rdx", size),
				bin(Mov, "rsi", ptr),
				bin(Mov, "rdi", "1"), // STDOUT
				bin(Mov, "rax", "1"), // WRITE
				{Instr: Syscall},

				bin(Mov, "rsp", "rbp"),
				unary(Pop, "rbp"),
				{Instr: Ret},
			}},
		},
	}
}

// error[ptr, int]
func genError(P *mir.Program) *fasmProc {
	ptrArg := mir.Operand{Class: mirc.CallerInterproc, Num: 0, Type: T.T_Ptr}
	sizeArg := mir.Operand{Class: mirc.CallerInterproc, Num: 1, Type: T.T_I64}
	ptr := convertOperand(P, ptrArg, 0, 0, 0)
	size := convertOperand(P, sizeArg, 0, 0, 0)
	return &fasmProc{
		label: "_error",
		blocks: []*fasmBlock{
			{label: "_error", code: []*amd64Instr{
				unary(Push, "rbp"),
				bin(Mov, "rbp", "rsp"),

				bin(Mov, "rdx", size),
				bin(Mov, "rsi", ptr),
				bin(Mov, "rdi", "2"), // STDERR
				bin(Mov, "rax", "1"), // WRITE
				{Instr: Syscall},

				bin(Mov, "rsp", "rbp"),
				unary(Pop, "rbp"),
				{Instr: Ret},
			}},
		},
	}
}

func genMem(mem *mir.MemoryDecl) *fasmData {
	if mem.Data == "" {
		return &fasmData{
			label:    mem.Label,
			content:  strconv.FormatUint(mem.Size, 10),
			declared: false,
		}
	}
	return &fasmData{
		label:    mem.Label,
		content:  convertString(mem.Data),
		declared: true,
	}
}

func convertString(original string) string {
	s := original[1 : len(original)-1] // removes quotes
	output := "'"
	for i := 0; i < len(s); i++ {
		r := s[i]
		if r == '\\' {
			i++
			r = s[i]
			switch r {
			case 'n':
				output += "', 0xA, '"
			case 't':
				output += "', 0x9, '"
			case 'r':
				output += "', 0xD, '"
			case '\'':
				output += "', 0x27, '"
			case '\\':
				output += "', 0x5C, '"
			default:
				output += string(r)
			}
		} else {
			output += string(r)
		}
	}
	output += "'"
	return output
}

func genEntry(P *mir.Program) []*amd64Instr {
	entry := P.Symbols[P.Entry]
	if entry == nil || entry.Proc == nil {
		panic("nil entrypoint")
	}
	return []*amd64Instr{
		unary(Call, entry.Proc.Label),
		bin(Xor, "rdi", "rdi"), // EXIT CODE 0
		bin(Mov, "rax", "60"),  // EXIT
		{Instr: Syscall},
	}
}

func genProc(P *mir.Program, proc *mir.Procedure) *fasmProc {
	stackReserve := 8 * (proc.NumOfVars + proc.NumOfSpills + proc.NumOfMaxCalleeArguments)
	init := &fasmBlock{
		label: proc.Label,
		code: []*amd64Instr{
			unary(Push, "rbp"),
			bin(Mov, "rbp", "rsp"),
			bin(Sub, "rsp", strconv.FormatInt(int64(stackReserve), 10)),
		},
	}
	fproc := &fasmProc{label: proc.Label, blocks: []*fasmBlock{init}}
	proc.ResetBlocks()
	fproc.blocks = append(fproc.blocks, genBlocks(P, proc, proc.FirstBlock())...)
	return fproc
}

func genBlocks(P *mir.Program, proc *mir.Procedure, start *mir.BasicBlock) []*fasmBlock {
	trueBranches := []*mir.BasicBlock{}
	falseBlocks := genFalseBranches(P, proc, start, &trueBranches)
	for _, tBlock := range trueBranches {
		out := genBlocks(P, proc, tBlock)
		falseBlocks = append(falseBlocks, out...)
	}
	return falseBlocks
}

func genFalseBranches(P *mir.Program, proc *mir.Procedure, block *mir.BasicBlock, trueBranches *[]*mir.BasicBlock) []*fasmBlock {
	if block.Visited {
		panic("no blocks should be already visited: " + block.Label)
	}
	block.Visited = true
	fb := genCode(P, proc, block)

	// should generate Jmp only for true branches and Jmps that point to
	// already visited blocks
	switch block.Out.T {
	case FT.Jmp:
		t := proc.GetBlock(block.Out.True)
		if t.Visited {
			jmp := unary(Jmp, t.Label)
			fb.code = append(fb.code, jmp)
			return []*fasmBlock{fb}
		}
		out := genFalseBranches(P, proc, t, trueBranches)
		out = append([]*fasmBlock{fb}, out...)
		return out
	case FT.If:
		t := proc.GetBlock(block.Out.True)
		if !t.Visited {
			*trueBranches = append(*trueBranches, t)
		}
		jmp := genCondJmp(P, proc, t, block.Out.V[0])
		fb.code = append(fb.code, jmp...)
		f := proc.GetBlock(block.Out.False)
		out := genFalseBranches(P, proc, f, trueBranches)
		out = append([]*fasmBlock{fb}, out...)
		return out
	case FT.Exit:
		exit := genExit(P, proc, block.Out.V[0])
		fb.code = append(fb.code, exit...)
		return []*fasmBlock{fb}
	case FT.Return:
		ret := genRet()
		fb.code = append(fb.code, ret...)
		return []*fasmBlock{fb}
	}
	panic("Invalid flow: " + block.Out.String())
}

func genCondJmp(P *mir.Program, proc *mir.Procedure, block *mir.BasicBlock, op mir.Operand) []*amd64Instr {
	newOp := convertOperandProc(P, proc, op)
	if op.Class == mirc.Lit || op.Class == mirc.Static {
		rbx := genReg(RBX, op.Type)
		return []*amd64Instr{
			bin(Mov, rbx, newOp),
			bin(Cmp, rbx, "1"),
			unary(Je, block.Label),
		}
	}
	return []*amd64Instr{
		bin(Cmp, newOp, "1"),
		unary(Je, block.Label),
	}
}

func genRet() []*amd64Instr {
	return []*amd64Instr{
		bin(Mov, "rsp", "rbp"),
		unary(Pop, "rbp"),
		{Instr: Ret},
	}
}

func genExit(P *mir.Program, proc *mir.Procedure, op mir.Operand) []*amd64Instr {
	exitCode := convertOperandProc(P, proc, op)
	return []*amd64Instr{
		bin(Xor, "rdi", "rdi"),
		bin(Mov, "dil", exitCode), // EXIT CODE
		bin(Mov, "rax", "60"),     // EXIT
		{Instr: Syscall},
	}
}

func genCode(P *mir.Program, proc *mir.Procedure, block *mir.BasicBlock) *fasmBlock {
	output := make([]*amd64Instr, len(block.Code))[:0]
	for _, instr := range block.Code {
		output = append(output, genInstr(P, proc, instr)...)
	}
	return &fasmBlock{label: block.Label, code: output}
}

func genInstr(P *mir.Program, proc *mir.Procedure, instr mir.Instr) []*amd64Instr {
	switch instr.T {
	case IT.Add, IT.Mult, IT.Or, IT.And:
		return genSimpleBin(P, proc, instr)
	case IT.Sub:
		return genSub(P, proc, instr)
	case IT.Eq, IT.Diff, IT.Less, IT.More, IT.LessEq, IT.MoreEq:
		return genComp(P, proc, instr)
	case IT.Load, IT.Store, IT.Copy:
		return genLoadStore(P, proc, instr)
	case IT.LoadPtr:
		return genLoadPtr(P, proc, instr)
	case IT.StorePtr:
		return genStorePtr(P, proc, instr)
	case IT.Neg:
		return genUnaryMinus(P, proc, instr)
	case IT.Div:
		return genDiv(P, proc, instr)
	case IT.Rem:
		return genRem(P, proc, instr)
	case IT.Not:
		return genNot(P, proc, instr)
	case IT.Convert:
		return genConvert(P, proc, instr)
	case IT.Call:
		return genCall(P, proc, instr)
	default:
		panic("unimplemented: " + instr.String())
	}
}

// uint -> int and int -> uint are "converted" xD
func genConvert(P *mir.Program, proc *mir.Procedure, instr mir.Instr) []*amd64Instr {
	out, newA := resolveOperand(P, proc, instr.A.Operand)
	newDest := convertOptOperandProc(P, proc, instr.Dest)
	if instr.Dest.Type.Size() > instr.A.Type.Size() {
		out = append(out, bin(Movsx, newDest, newA))
		return out
	}
	if instr.A.Class == mirc.Lit {
		out = append(out, bin(Mov, newDest, newA))
		return out
	}
	if instr.A.Class == mirc.Static {
		res := genReg(RAX, instr.Dest.Type)
		out = append(out, []*amd64Instr{
			bin(Mov, "rax", newA),
			bin(Mov, newDest, res),
		}...)
		return out
	}
	if !areOpEqual(instr.A, instr.Dest) {
		newA := getReg(instr.A.Num, instr.Dest.Type)
		newDest := convertOptOperandProc(P, proc, instr.Dest)
		return []*amd64Instr{
			bin(Mov, newDest, newA),
		}
	}
	return []*amd64Instr{}
}

func genCall(P *mir.Program, proc *mir.Procedure, instr mir.Instr) []*amd64Instr {
	newA := convertOptOperandProc(P, proc, instr.A)
	return []*amd64Instr{
		unary(Call, newA),
	}
}

func genLoadStore(P *mir.Program, proc *mir.Procedure, instr mir.Instr) []*amd64Instr {
	out, newA := resolveOperand(P, proc, instr.A.Operand)
	newDest := convertOptOperandProc(P, proc, instr.Dest)
	out = append(out, mov(newDest, newA))
	return out
}

func genLoadPtr(P *mir.Program, proc *mir.Procedure, instr mir.Instr) []*amd64Instr {
	newA := convertOptOperandProc(P, proc, instr.A)
	newDest := convertOptOperandProc(P, proc, instr.Dest)
	a := mov(newDest, genType(instr.Type)+"["+newA+"]") // xD
	return []*amd64Instr{a}
}

func genStorePtr(P *mir.Program, proc *mir.Procedure, instr mir.Instr) []*amd64Instr {
	out, newA := resolveOperand(P, proc, instr.A.Operand)
	newDest := convertOptOperandProc(P, proc, instr.B)
	a := mov(genType(instr.Type)+"["+newDest+"]", newA)
	out = append(out, a)
	return out
}

func genNot(P *mir.Program, proc *mir.Procedure, instr mir.Instr) []*amd64Instr {
	newA := convertOptOperandProc(P, proc, instr.A)
	newDest := convertOptOperandProc(P, proc, instr.Dest)
	return []*amd64Instr{
		bin(Cmp, newA, "0"),
		unary(Sete, newDest),
	}
}

func genUnaryMinus(P *mir.Program, proc *mir.Procedure, instr mir.Instr) []*amd64Instr {
	out, newA := resolveOperand(P, proc, instr.A.Operand)
	newDest := convertOptOperandProc(P, proc, instr.Dest)
	out = append(out, []*amd64Instr{
		mov(newDest, newA),
		unary(Neg, newDest),
	}...)
	return out
}

func genComp(P *mir.Program, proc *mir.Procedure, instr mir.Instr) []*amd64Instr {
	newOp1 := convertOptOperandProc(P, proc, instr.A)
	out, newOp2 := resolveOperand(P, proc, instr.B.Operand)
	newDest := convertOptOperandProc(P, proc, instr.Dest)
	newInstr := genInstrName(instr)
	if instr.A.Class == mirc.Lit || instr.A.Class == mirc.Static {
		rax := genReg(RAX, instr.A.Type)
		out = append(out, []*amd64Instr{
			bin(Mov, rax, newOp1),
			bin(Cmp, rax, newOp2),
			unary(newInstr, newDest),
		}...)
		return out
	}
	out = append(out, []*amd64Instr{
		bin(Cmp, newOp1, newOp2),
		unary(newInstr, newDest),
	}...)
	return out
}

func genDiv(P *mir.Program, proc *mir.Procedure, instr mir.Instr) []*amd64Instr {
	instrName := genInstrName(instr)
	newOp1 := convertOptOperandProc(P, proc, instr.A)
	newOp2 := convertOptOperandProc(P, proc, instr.B)
	newDest := convertOptOperandProc(P, proc, instr.Dest)
	if instr.B.Class == mirc.Lit || instr.B.Class == mirc.Static {
		rbx := genReg(RBX, instr.Type)
		return []*amd64Instr{
			bin(Xor, RDX.QWord, RDX.QWord),
			mov(genReg(RAX, instr.Type), newOp1),
			mov(rbx, newOp2),
			unary(instrName, rbx),
			mov(newDest, genReg(RAX, instr.Type)),
		}
	}
	return []*amd64Instr{
		bin(Xor, RDX.QWord, RDX.QWord),
		mov(genReg(RAX, instr.Type), newOp1),
		unary(instrName, newOp2),
		mov(newDest, genReg(RAX, instr.Type)),
	}
}

func genRem(P *mir.Program, proc *mir.Procedure, instr mir.Instr) []*amd64Instr {
	instrName := genInstrName(instr)

	newOp1 := convertOptOperandProc(P, proc, instr.A)
	newOp2 := convertOptOperandProc(P, proc, instr.B)
	newDest := convertOptOperandProc(P, proc, instr.Dest)
	if mirc.IsImmediate(instr.B.Class) {
		rbx := genReg(RBX, instr.Type)
		return []*amd64Instr{
			bin(Xor, RDX.QWord, RDX.QWord),
			mov(genReg(RAX, instr.Type), newOp1),
			mov(rbx, newOp2),
			unary(instrName, rbx),
			mov(newDest, genReg(RDX, instr.Type)),
		}
	}
	return []*amd64Instr{
		bin(Xor, RDX.QWord, RDX.QWord),
		mov(genReg(RAX, instr.Type), newOp1),
		unary(instrName, newOp2),
		mov(newDest, genReg(RDX, instr.Type)),
	}
}

func genSub(P *mir.Program, proc *mir.Procedure, instr mir.Instr) []*amd64Instr {
	sub := genInstrName(instr)

	if areOpEqual(instr.A, instr.Dest) {
		out, newOp1 := resolveOperand(P, proc, instr.B.Operand)
		newDest := convertOptOperandProc(P, proc, instr.Dest)
		out = append(out, []*amd64Instr{
			bin(sub, newDest, newOp1),
		}...)
		return out
	}

	if areOpEqual(instr.B, instr.Dest) {
		rax := genReg(RAX, instr.Type)
		newOp1 := convertOptOperandProc(P, proc, instr.A)
		out, newOp2 := resolveOperand(P, proc, instr.B.Operand)
		newDest := convertOptOperandProc(P, proc, instr.Dest)
		out = append(out, []*amd64Instr{
			bin(Xor, RAX.QWord, RAX.QWord),
			mov(rax, newOp1),
			bin(sub, rax, newOp2),
			mov(newDest, rax),
		}...)
		return out
	}

	newOp1 := convertOptOperandProc(P, proc, instr.A)
	out, newOp2 := resolveOperand(P, proc, instr.B.Operand)
	newDest := convertOptOperandProc(P, proc, instr.Dest)
	out = append(out, []*amd64Instr{
		mov(newDest, newOp1),
		bin(sub, newDest, newOp2),
	}...)
	return out
}

func genSimpleBin(P *mir.Program, proc *mir.Procedure, instr mir.Instr) []*amd64Instr {
	dest, op, ok := convertToTwoAddr(instr)
	newInstr := genInstrName(instr)
	if ok {
		out, newOp1 := resolveOperand(P, proc, op)
		newDest := convertOperandProc(P, proc, dest)
		out = append(out, []*amd64Instr{
			bin(newInstr, newDest, newOp1),
		}...)
		return out
	}
	newOp1 := convertOptOperandProc(P, proc, instr.A)
	// we only need to resolve it for the second one, the first is already going in a register
	out, newOp2 := resolveOperand(P, proc, instr.B.Operand)
	newDest := convertOptOperandProc(P, proc, instr.Dest)
	out = append(out, []*amd64Instr{
		mov(newDest, newOp1),
		bin(newInstr, newDest, newOp2),
	}...)
	return out
}

func unary(instr string, op string) *amd64Instr {
	return &amd64Instr{
		Instr: instr,
		Op1:   op,
	}
}

func bin(instr string, dest, source string) *amd64Instr {
	return &amd64Instr{
		Instr: instr,
		Op1:   dest,
		Op2:   source,
	}
}

func mov(dest, source string) *amd64Instr {
	return &amd64Instr{
		Instr: Mov,
		Op1:   dest,
		Op2:   source,
	}
}

func convertToTwoAddr(instr mir.Instr) (dest mir.Operand, op mir.Operand, ok bool) {
	if !instr.Dest.Valid {
		return mir.Operand{}, mir.Operand{}, false
	}
	if instr.T == IT.Sub {
		panic("subtraction is not comutative")
	}

	if areOpEqual(instr.A, instr.Dest) {
		return instr.Dest.Operand, instr.B.Operand, true
	}
	if areOpEqual(instr.B, instr.Dest) {
		return instr.Dest.Operand, instr.A.Operand, true
	}

	return mir.Operand{}, mir.Operand{}, false
}

func areOpEqual(a, b mir.OptOperand) bool {
	if !a.Valid || !b.Valid {
		panic("invalid operand")
	}
	return a.Class == b.Class &&
		a.Num == b.Num
}

func convertOperandProc(P *mir.Program, proc *mir.Procedure, op mir.Operand) string {
	return convertOperand(P, op, uint64(proc.NumOfVars), uint64(proc.NumOfSpills), uint64(proc.NumOfMaxCalleeArguments))
}

func convertOptOperandProc(P *mir.Program, proc *mir.Procedure, op mir.OptOperand) string {
	if !op.Valid {
		panic("invalid operand")
	}
	return convertOperand(P, op.Operand, uint64(proc.NumOfVars), uint64(proc.NumOfSpills), uint64(proc.NumOfMaxCalleeArguments))
}

func convertOperand(P *mir.Program, op mir.Operand, NumOfVars, NumOfSpills, NumOfMaxCalleeArguments uint64) string {
	switch op.Class {
	case mirc.Register:
		return getReg(op.Num, op.Type)
	case mirc.CallerInterproc:
		//        v must jump last rbp + return address
		offset := 16 + op.Num*8
		return genType(op.Type) + "[rbp + " + strconv.FormatUint(offset, 10) + "]"
	case mirc.Local:
		//        v begins at 8 because rbp points to the last rbp
		offset := 8 + op.Num*8
		return genType(op.Type) + "[rbp - " + strconv.FormatUint(offset, 10) + "]"
	case mirc.Spill:
		offset := 8 + NumOfVars*8 + op.Num*8
		return genType(op.Type) + "[rbp - " + strconv.FormatUint(offset, 10) + "]"
	case mirc.CalleeInterproc:
		offset := 8 + NumOfVars*8 +
			NumOfSpills*8 +
			// v count                   v index
			(NumOfMaxCalleeArguments-1-op.Num)*8
		return genType(op.Type) + "[rbp - " + strconv.FormatUint(offset, 10) + "]"
	case mirc.Lit:
		return strconv.FormatUint(op.Num, 10)
	case mirc.Static:
		sy := P.Symbols[op.Num]
		if sy.Proc != nil {
			return sy.Proc.Label
		}
		return sy.Mem.Label
	}
	panic("unimplemented: " + op.String())
}

func getReg(num uint64, t *T.Type) string {
	if num > uint64(len(Registers)) || num < 0 {
		panic("oh no")
	}
	r := Registers[num]
	return genReg(r, t)
}

func genReg(r *register, t *T.Type) string {
	if T.IsBasic(t) {
		switch t.Basic {
		case T.Ptr:
			return r.QWord
		case T.I64, T.U64:
			return r.QWord
		case T.I32, T.U32:
			return r.DWord
		case T.I16, T.U16:
			return r.Word
		case T.I8, T.U8:
			return r.Byte
		case T.Bool:
			return r.Byte
		}
	} else {
		if !T.IsProc(t) {
			panic(t.String())
		}
		return r.QWord
	}
	panic(t.String())
}

func genType(t *T.Type) string {
	switch t.Size() {
	case 1:
		return "byte"
	case 2:
		return "word"
	case 4:
		return "dword"
	case 8:
		return "qword"
	}
	panic(t.String())
}

func genInstrName(instr mir.Instr) string {
	if T.IsInt(instr.Type) {
		switch instr.T {
		case IT.Add:
			return Add
		case IT.Sub:
			return Sub
		case IT.Mult:
			return IMul
		case IT.Div, IT.Rem:
			return IDiv
		case IT.And:
			return And
		case IT.Or:
			return Or
		case IT.Eq:
			return Sete
		case IT.Diff:
			return Setne
		case IT.Less:
			return Setl
		case IT.More:
			return Setg
		case IT.MoreEq:
			return Setge
		case IT.LessEq:
			return Setle
		}
	} else {
		switch instr.T {
		case IT.Add:
			return Add
		case IT.Sub:
			return Sub
		case IT.Mult:
			return Mul
		case IT.Div, IT.Rem:
			return Div
		case IT.And:
			return And
		case IT.Or:
			return Or
		case IT.Eq:
			return Sete
		case IT.Diff:
			return Setne
		case IT.Less:
			return Setl
		case IT.More:
			return Setg
		case IT.MoreEq:
			return Setge
		case IT.LessEq:
			return Setle
		}
	}
	fmt.Println(instr)
	panic("unimplemented")
}

/* in amd64, you can only load an imm64 to a register,
so if the value is beyond the 32bit range, you need to
use a register before moving things to memory.

this should really be part of the register allocator i think,
but for now it suffices to resolve it here.
*/
func resolveOperand(P *mir.Program, proc *mir.Procedure, op mir.Operand) ([]*amd64Instr, string) {
	opstr := convertOperandProc(P, proc, op)
	if op.Class == mirc.Lit && op.Num > (1<<31) {
		out := genReg(RBX, op.Type)
		mv := mov(out, opstr)
		return []*amd64Instr{mv}, out
	}
	return nil, opstr
}
