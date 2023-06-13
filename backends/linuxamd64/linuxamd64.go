package linuxamd64

import (
	"pir"
	"pir/backends/linuxamd64/fasm"
	"pir/backends/linuxamd64/resalloc"
	"pir/checker"
)

func GenerateFasm(p *pir.Program) string {
	checker.Check(p)
	mirProgram := resalloc.Allocate(p, len(fasm.Registers))
	fasmProgram := fasm.Generate(mirProgram)
	return fasmProgram.Contents
}
