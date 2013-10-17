package main

import "fmt"
import "llvm"

type LLVMNamespace struct {
	TypeCounter map[string]uint
	Zero        map[string]TypedValue
	Mod         llvm.Module
}

func CreateNamespace(mod llvm.Module) *LLVMNamespace {
	return &LLVMNamespace{make(map[string]uint, 0), make(map[string]TypedValue, 0), mod}
}

func (ns *LLVMNamespace) createAndSetGlobal(id string, ty llvm.Type, ll llvm.Value) llvm.Value {
	llvmVal := ns.Mod.AddGlobal(ty, id)
	llvm.SetGlobalConstant(llvmVal, true)
	llvm.SetInitializer(llvmVal, ll)
	return llvmVal
}

func (ns *LLVMNamespace) requestStaticConstAlloc(ty Type, llTy llvm.Type, val llvm.Value) llvm.Value {
	rootTy := ty.BaseIDString()
	curr, ok := ns.TypeCounter[rootTy]
	if !ok {
		curr = 0
	}
	allocated := fmt.Sprintf("%s.%d", rootTy, curr)
	ns.TypeCounter[rootTy] = curr + 1

	// and create it
	return ns.createAndSetGlobal(allocated, llTy, val)
}

/*
 * This function creates empty type instances as necessary. After the first
 * instance is created, the same value is used for all successive requests.
 *
 * The value in the LLVM module is initialized as a constant, module-wide value.
 */

func (ns *LLVMNamespace) allocZero(zeroVal TypedValue) llvm.Value {
	// a simple function to memoize empty, CONST values.
	rootTy := zeroVal.Type().BaseIDString()
	emptyID := rootTy + ".zero"
	return ns.createAndSetGlobal(emptyID, zeroVal.Type().LLVM(), zeroVal.LLVM())
}
