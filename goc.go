package main

import "os"
func main() {
	pipe := CreatePipeline()
	pipe.AddStage(CreateGocStage("test.go", "test.bc"))
	pipe.AddStage(CreateLLCStage("test.bc", "test.o"))
	pipe.AddStage(CreateLinkStage([]string{"rt/c/rt.o", "test.o"}, "test"))
	//pipe.AddStage(CreateCleanStage("test.bc", "test.o"))
	diag := pipe.Execute(true)
	if diag != nil {
		PrintDiagnostic(diag)
		os.Exit(1)
	}
}
