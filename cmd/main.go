// this file only exists so Go actually checks all files in a go vet :)
// and actually, you have to explicitly go vet this file
//     go vet cmd/main.go
package main

import (
	"github.com/padeir0/pir"
	"github.com/padeir0/pir/backends/linuxamd64"
	"github.com/padeir0/pir/backends/linuxamd64/fasm"
	"github.com/padeir0/pir/backends/linuxamd64/mir"
	mirchecker "github.com/padeir0/pir/backends/linuxamd64/mir/checker"
	mirc "github.com/padeir0/pir/backends/linuxamd64/mir/class"
	mk "github.com/padeir0/pir/backends/linuxamd64/mir/instrkind"
	pirchecker "github.com/padeir0/pir/checker"
	pirc "github.com/padeir0/pir/class"
	ik "github.com/padeir0/pir/instrkind"
	T "github.com/padeir0/pir/types"

	"fmt"
)

func main() {
	fmt.Println(T.I16)
	fmt.Println(ik.Add)
	fmt.Println(mk.Add)
	mirchecker.Check(&mir.Program{})
	pirchecker.Check(&pir.Program{})
	fasm.Generate(&mir.Program{})
	linuxamd64.GenerateFasm(&pir.Program{})
	fmt.Println(mirc.Lit)
	fmt.Println(pirc.Lit)
}
