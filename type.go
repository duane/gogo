package main

import "llvm"
import "fmt"
import "math/big"

type Type interface {
	String() string
	LLVM() llvm.Type
	Eq(Type) bool
	Base() Type
	BaseIDString() string
	Zero(*LLVMNamespace) TypedValue
	Named() bool // Named types are interfaces, aliases.
}

type TypeMap map[string]Type

func NamedString(ty Type, name string) string {
	return fmt.Sprintf("%s %s", name, ty.String())
}

const (
	BLTN_TY_UINT8 = iota
	BLTN_TY_UINT16
	BLTN_TY_UINT32
	BLTN_TY_UINT64

	BLTN_TY_INT8
	BLTN_TY_INT16
	BLTN_TY_INT32
	BLTN_TY_INT64

  BLTN_TY_INT
  BLTN_TY_UINT

	BLTN_TY_LIT
)

type IntType struct {
	Type uint
	target Target
}

func (num *IntType) BitWidth() uint {
	switch num.Type {
	case BLTN_TY_UINT8:
		return 8
	case BLTN_TY_INT8:
		return 8
	case BLTN_TY_UINT16:
		return 16
	case BLTN_TY_INT16:
		return 16
	case BLTN_TY_UINT32:
		return 32
	case BLTN_TY_INT32:
		return 32
	case BLTN_TY_UINT64:
		return 64
	case BLTN_TY_INT64:
		return 64
	case BLTN_TY_INT:
		return num.target.WordSize()
	case BLTN_TY_UINT:
		return num.target.WordSize()
	default:
		panic(fmt.Sprintf("Invalid internal state (unknown integer type %u).", num.Type))
		break
	}
	panic("Unreachable code. Please fix.")
}

func (num *IntType) String() string {
	switch num.Type {
	case BLTN_TY_UINT8:
		return "uint8"
	case BLTN_TY_UINT16:
		return "uint16"
	case BLTN_TY_UINT32:
		return "uint32"
	case BLTN_TY_UINT64:
		return "uint64"
	case BLTN_TY_INT8:
		return "int8"
	case BLTN_TY_INT16:
		return "int16"
	case BLTN_TY_INT32:
		return "int32"
	case BLTN_TY_INT64:
		return "int64"
	case BLTN_TY_INT:
		return "int"
	case BLTN_TY_UINT:
		return "uint"
	default:
		panic(fmt.Sprintf("Invalid internal state (unknown integer type %u).", num.Type))
		break
	}
	panic("Unreachable code. Please fix.")
}

func (num *IntType) LLVM() llvm.Type {
	return llvm.IntType(num.BitWidth())
}

func (num *IntType) Eq(ty Type) bool {
	other, ok := ty.(*IntType)
	if ok {
		return num.Type == other.Type
	}
	return false
}

func (num *IntType) Base() Type {
	return num
}

func (num *IntType) Zero(ns *LLVMNamespace) TypedValue {
	return &TypedConstInt{ConstInt{big.NewInt(0), true}, num}
}

func (num *IntType) BaseIDString() string {
	return num.String()
}

func (num *IntType) Named() bool {
	return false
}

type AliasType struct {
	Ident string
	Alias Type
}

func (alias *AliasType) String() string {
	return alias.Ident
}

func (alias *AliasType) LLVM() llvm.Type {
	return alias.Alias.LLVM()
}

func (alias *AliasType) Eq(ty Type) bool {
	other, ok := ty.(*AliasType)
	if ok {
		return alias.Ident == other.Ident && alias.Alias.Eq(other.Alias)
	}
	return false
}

func (alias *AliasType) Base() Type {
	return alias.Alias.Base()
}

func (alias *AliasType) Zero(ns *LLVMNamespace) TypedValue {
	return alias.Alias.Zero(ns)
}

func (alias *AliasType) BaseIDString() string {
	return alias.Alias.Base().BaseIDString()
}

func (alias *AliasType) Named() bool {
	return true
}

type PointerType struct {
	At Type
}

func (ptr *PointerType) String() string {
	return "*" + ptr.At.String()
}

func (ptr *PointerType) LLVM() llvm.Type {
	return llvm.PointerType(ptr.At.LLVM(), 0)
}

func (ptr *PointerType) Eq(ty Type) bool {
	other, ok := ty.(*PointerType)
	if ok {
		return ptr.At.Eq(other.At)
	}
	return false
}

func (ptr *PointerType) Base() Type {
	return &PointerType{ptr.At.Base()}
}

func (ptr *PointerType) BaseIDString() string {
	return "p." + ptr.At.BaseIDString()
}

func (ptr *PointerType) Zero(ns *LLVMNamespace) TypedValue {
	return CreateNilPointer(ptr)
}

func (ptr *PointerType) Named() bool {
	return ptr.At.Named()
}

type FuncType struct {
	Params []Type
	Result Type
}

func (fn *FuncType) String() string {
	paramStr := ""
	rest := false
	for _, ty := range fn.Params {
		if rest {
			paramStr = paramStr + ", "
		}
		paramStr += ty.String()
		rest = true
	}

	resultStr := ""
	if fn.Result != nil {
		resultStr = " " + fn.Result.String()
	}

	return fmt.Sprintf("func (%s)%s", paramStr, resultStr)
}

func (fn *FuncType) LLVM() llvm.Type {
	paramTypes := make([]llvm.Type, 0)
	for _, param := range fn.Params {
		paramTypes = append(paramTypes, param.LLVM())
	}
	var returnType llvm.Type
	if fn.Result == nil {
		returnType = llvm.VoidType()
	} else {
		returnType = fn.Result.LLVM()
	}

	llvmTy := llvm.FunctionType(returnType, paramTypes, false)
	return llvmTy
}

func (fn *FuncType) Eq(ty Type) bool {
	other, ok := ty.(*FuncType)
	if ok {
		if len(fn.Params) != len(other.Params) {
			return false
		}
		for i, ty := range fn.Params {
			if !ty.Eq(other.Params[i]) {
				return false
			}
		}
		if !fn.Result.Eq(other.Result) {
			return false
		}
		return true
	}
	return false
}

func (fn *FuncType) Base() Type {
	baseParams := make([]Type, len(fn.Params))
	for i, v := range fn.Params {
		baseParams[i] = v.Base()
	}
	return &FuncType{baseParams, fn.Result.Base()}
}

func (fn *FuncType) BaseIDString() string {
	// This is very, very painful.
	str := "f."
	str = str + "r."
	if fn.Result != nil {
		str = str + fn.Result.BaseIDString() + "."
	}
	str = str + "p"
	for _, paramTy := range fn.Params {
		str = str + "." + paramTy.BaseIDString()
	}
	return str
}

func (fn *FuncType) Zero(ns *LLVMNamespace) TypedValue {
	return CreateNilPointer(fn)
}

func CreateFunctionType(result Type, params ...Type) *FuncType {
	return &FuncType{params, result}
}

func (fn *FuncType) Named() bool {
	return false
}

func GetStringType() Type {
	char_ty := global_type_factory.IntType(BLTN_TY_UINT8)
	return &PointerType{char_ty}
}

func assignable(typeA Type, typeB Type) bool {
	// this is stupid.
	switch tyA := typeA.(type) {
	default:
		switch tyB := typeB.(type) {
		default:
			if typeA.Eq(typeB) {
				return true
			}

			if typeA.Base().Eq(typeB.Base()) && (!(tyA.Named() && tyB.Named())) {
				return true
			}
		}
	}
	panic("unreachable!")
}

func TypeMismatchDiag(expected Type, actual Type) *UDiag {
	udiag := UDiag("Expected type " + expected.String() + " but got type " + actual.String()) 
	return &udiag
}
