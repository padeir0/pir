package linuxamd64

import (
	"github.com/padeir0/pir"
	"github.com/padeir0/pir/backends/linuxamd64/fasm"
	"github.com/padeir0/pir/backends/linuxamd64/resalloc"
	. "github.com/padeir0/pir/errors"

	mirchecker "github.com/padeir0/pir/backends/linuxamd64/mir/checker"
	pirchecker "github.com/padeir0/pir/checker"
)

func GenerateFasm(p *pir.Program) (string, *Error) {
	err := pirchecker.Check(p)
	if err != nil {
		return "", err
	}
	mirProgram := resalloc.Allocate(p, len(fasm.Registers))
	err = mirchecker.Check(mirProgram)
	if err != nil {
		return "", err
	}
	fasmProgram := fasm.Generate(mirProgram)
	return fasmProgram.Contents, nil
}
