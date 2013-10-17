package main

import "llvm"

type Block struct {
	Scope    *Scope
	Block    llvm.BasicBlock
	Builder  llvm.Builder
	ResultTy Type
	Trans    *Translator
}
