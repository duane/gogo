package main

import "go/token"
import "go/ast"
import "fmt"
import "os"

// an unbound (blame-less) diagnostic.
type UDiag string

type GoDiag struct {
	start, end token.Pos
	fset       *token.FileSet
	msg        string
}

const (
	BLAME_TEXT_SINGLE uint = iota
	BLAME_TEXT_MULTI
	BLAME_BINARY
	BLAME_CMD
	BLAME_NONE
)

// TODO: write convenience constructors for each type.
type Blame struct {
	Type                     uint
	File                     string // for TEXT_SINGLE, TEXT_MULTI, and BINARY
	Line, Col, Extent, Caret uint   // for text files, single line
	LineStart, LineEnd       uint   // for text files, multiline
	Offset                   uint   // for binary files
	Cmd, Invocation, Output  string // for commands
}

func DiagFromAST(ast ast.Node, format string, args ...interface{}) *GoDiag {
	return &GoDiag{ast.Pos(), ast.End(), nil, fmt.Sprintf(format, args...)}
}

func BindDiagToAST(ast ast.Node, unbound UDiag) *GoDiag {
	return &GoDiag{ast.Pos(), ast.End(), nil, string(unbound)}
}

// TODO: this is pretty much the same as UDiag; fix!
type GenError string

func (err *GenError) Msg() string {
	return string(*err)
}

func (err *GenError) Blame() Blame {
	return Blame{BLAME_NONE, "", 0, 0, 0, 0, 0, 0, 0, "", "", ""}
}

type Diag interface {
	Blame() Blame
	Msg() string
}

func (diag *GoDiag) Blame() Blame {
	assert(diag.fset != nil, "Please set godiag's fileset before printing!")

	// first, see whether we have a multi-line diagnostic or not.
	fileRef := diag.fset.File(diag.start)
	assert(fileRef == diag.fset.File(diag.end), "Diagnostic appears to span multiple files!")
	startPos := fileRef.Position(diag.start)
	endPos := fileRef.Position(diag.end)
	assert(startPos.Line <= endPos.Line, "Diagnostic appears to run backwards.")
	if startPos.Line == endPos.Line {
		// Single line
		assert(startPos.Column <= endPos.Column, "Diagnostic appears to run backwards.")
		return Blame{BLAME_TEXT_SINGLE, startPos.Filename, uint(startPos.Line), uint(startPos.Column), uint(endPos.Column - startPos.Column), uint(startPos.Column), 0, 0, 0, "", "", ""}
	} else {
		// Multi-line
		return Blame{BLAME_TEXT_MULTI, startPos.Filename, 0, 0, 0, 0, uint(startPos.Line), uint(endPos.Line), 0, "", "", ""}
	}
	panic("Unreachable!")
	return Blame{}
}

func (diag *GoDiag) Msg() string {
	return diag.msg
}

func (blame Blame) simpleRef() string {
	switch blame.Type {
	case BLAME_TEXT_SINGLE:
		if blame.Extent == 0 {
			return fmt.Sprintf("%s:%d:%d", blame.File, blame.Line, blame.Col)
		}
		return fmt.Sprintf("%s:%d:%d-%d", blame.File, blame.Line, blame.Col, blame.Col+blame.Extent)
	case BLAME_TEXT_MULTI:
		return fmt.Sprintf("%s:%d-d", blame.File, blame.LineStart, blame.LineEnd)
	case BLAME_BINARY:
		return fmt.Sprintf("%s[offset %X bytes]", blame.File, blame.Offset)
	case BLAME_CMD:
		return fmt.Sprintf("command '%s'", blame.Cmd)
	}
	panic("Bad internal state! (Unknown blame type!)")
	return ""
}

func (blame Blame) printLine() bool {
	assert(blame.Type == BLAME_TEXT_SINGLE, "Attempting to get a single line for a non-single-line diagnostic!")
	file, err := os.Open(blame.File)
	if err != nil {
		return false
	}
	// now seek forward to line n
	n := blame.Line
	byt := []byte{0}
	for i := 0; i < int(n-1); i++ {
		for byt[0] != '\n' {
			read, err := file.Read(byt)
			if err != nil || read == 0 {
				file.Close()
				return false
			}
		}
		byt[0] = 0
	}
	line := []byte{}
	for {
		read, err := file.Read(byt)
		if err != nil || read == 0 {
			break
		}
		if byt[0] == '\n' {
			break
		}
		line = append(line, byt[0])
	}
	file.Close()
	lineStr := string(line)
	fmt.Printf("\t%s\n", lineStr)
	return true
}

func (blame Blame) printTextSingle() {
	if !blame.printLine() { // something went wrong with finding the line in the file.
		return
	}

	assert(blame.Col <= blame.Caret && blame.Caret <= (blame.Col+blame.Extent), "Caret not in column range!")

	caretOffset := blame.Caret - blame.Col

	detailBuf := make([]byte, blame.Col+blame.Extent)
	var i uint
	for i = 0; i < blame.Col-1; i++ {
		detailBuf[i] = ' '
	}
	for i = 0; i < blame.Extent; i++ {
		if i == caretOffset {
			detailBuf[blame.Col+i] = '^'
		} else {
			detailBuf[blame.Col+i] = '~'
		}
	}

	fmt.Printf("\t%s\n", string(detailBuf))
}

func (blame Blame) printTextMulti() {
	fmt.Printf("\tTODO: Multi-line diagnostic\n")
}

func (blame Blame) printBinary() {
}

func (blame Blame) printCmd() {
	fmt.Printf("\tCommand invocation: %s\n", blame.Invocation)
	fmt.Printf("\tCommand Output:\n%s", blame.Output)
}

func PrintDiagnostic(diag Diag) {
	blame := diag.Blame()
	if blame.Type == BLAME_NONE {
		fmt.Printf("Error: %s\n", diag.Msg())
		return
	}
	fmt.Printf("Error: %s: %s\n", blame.simpleRef(), diag.Msg())
	switch blame.Type {
	case BLAME_TEXT_SINGLE:
		blame.printTextSingle()
		break
	case BLAME_TEXT_MULTI:
		blame.printTextMulti()
		break
	case BLAME_BINARY:
		blame.printBinary()
		break
	case BLAME_CMD:
		blame.printCmd()
		break
	case BLAME_NONE:
	default:
		panic("Bad internal state! Unknown blame type given.")
		break
	}
}
