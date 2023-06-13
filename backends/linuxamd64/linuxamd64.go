package linuxamd64

import (
	"github.com/padeir0/pir"
	"github.com/padeir0/pir/backends/linuxamd64/fasm"
	"github.com/padeir0/pir/backends/linuxamd64/resalloc"
	"github.com/padeir0/pir/checker"
)

func GenerateFasm(p *pir.Program) string {
	checker.Check(p)
	mirProgram := resalloc.Allocate(p, len(fasm.Registers))
	fasmProgram := fasm.Generate(mirProgram)
	return fasmProgram.Contents
}
