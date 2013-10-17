package main

import "llvm"

type Target struct {
	DataLayout string
	Triple     string
	Data       llvm.TargetData
}

func CreateNativeTarget() Target {
	targets := map[string]map[string]Target{
		"Darwin": {
			"i686": Target{
				"e-p:32:32:32-i1:8:8-i8:8:8-i16:16:16-i32:32:32-i64:32:64-f32:32:32-f64:32:64-v64:64:64-v128:128:128-a0:0:64-f80:128:128-n8:16:32-S128",
				"i686-apple-darwin",
				llvm.TargetData{nil},
			},
			"x86_64": Target{
				"e-p:64:64:64-i1:8:8-i8:8:8-i16:16:16-i32:32:32-i64:64:64-f32:32:32-f64:64:64-v64:64:64-v128:128:128-a0:0:64-s0:64:64-f80:128:128-n8:16:32:64-S128",
				"x86_64-apple-darwin",
				llvm.TargetData{nil},
			},
		},
	}

	os, _, arch := Uname()
	tar := targets[os][arch]
	tar.Data = llvm.CreateTargetData(tar.DataLayout)
	return tar
}

func (tar Target) DisposeTarget() {
	tar.Data.Dispose()
}

func (tar Target) WordSize() uint {
	return tar.Data.PointerSize()
}
