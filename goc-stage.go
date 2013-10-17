package main

import "go/token"
import "go/parser"
import "llvm"
import "fmt"

type GocStage struct {
	Input  string
	Output string
}

func CreateGocStage(Input string, Output string) *GocStage {
	return &GocStage{Input, Output}
}

func (stage *GocStage) Name() string {
	return "goc"
}

func (stage *GocStage) Run() Diag {
	fset := token.NewFileSet()
	ast, err := parser.ParseFile(fset, stage.Input, nil, 0)
	if err != nil {
		diag := GenError(fmt.Sprintf("Error while parsing file %s: %s", stage.Input, err.Error()))
		return &diag
	}
	trans := CreateTranslator()
	mod, diag := trans.translateFile(ast, fset)
	if diag != nil {
		llvm.DisposeModule(mod)
		return diag
	}

	mod.WriteBitcodeToFile(stage.Output)

	llvm.DisposeModule(mod)
	return nil
}
