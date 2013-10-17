package main

import "llvm"
import "math/big"
import "fmt"

type UntypedValue interface {

	String() string
	LValue() bool
	// If the value is a constant, untyped variable, this converts the value to
	// the expected type if possible. If it is not possible, it retunrns <nil>.
	// If it is possible, it retuns a new, typed TypedValue.
	// If <expected_type> is nil, it returns a default type. For example, an <int>
	// for a const int.
	RValue(expected_type Type) (TypedValue, *UDiag)
}

type TypedValue interface {
	UntypedValue
	Type() Type
	LLVM() llvm.Value
}

/* 
   This value is a variable. What distinguishes this type of value from other
   values is that it has the potential to be assigned to. Consequently, we must
   also consider when it cannot be assigned to, which leads us to using this
   data type to store the `const`ness of a variable.
 
   The LLVM value returns the last known value for the variable; for example,
   `x:=5;` followed by an evaluation of `x` evaluates to a constant `5`.
 */
type BoundVar struct {
	Ident string
	Const bool
	Val   UntypedValue
}

func (v *BoundVar) Type() Type {
	concrete, ok := v.Val.(TypedValue)
	if !ok {
		panic("Attemped to find type of non-concrete value.")
	}
	return concrete.Type()
}

func (v *BoundVar) String() string {
	return fmt.Sprintf("<%s = %s>", v.Ident, v.Val.String())
}

func (v *BoundVar) LLVM() llvm.Value {
	concrete, ok := v.Val.(TypedValue)
	if !ok {
		panic("Attempted to find type of non-concrete value.")
	}
	return concrete.LLVM()
}

func (v *BoundVar) LValue() bool {
	if v.Const {
		return false
	}
	return true
}

func (v *BoundVar) RValue(expected_type Type) (TypedValue, *UDiag) {
	typed, diag := v.Val.RValue(expected_type)
	if diag != nil {
		return nil, diag
	}
	return typed, nil
}

type ValueMap map[string]*BoundVar

type FuncValue struct {
	Name    string
	Ty      *FuncType
	LLVMVal llvm.Value
}

func (fn *FuncValue) RValue(ty Type) (TypedValue, *UDiag) {
	if ty == nil || ty.Eq(fn.Ty) {
		return fn, nil
	}
	return nil, TypeMismatchDiag(ty, fn.Ty)
}

func (fn *FuncValue) Type() Type {
	return fn.Ty
}

func (fn *FuncValue) String() string {
	return NamedString(fn.Ty, fn.Name)
}

func (fn *FuncValue) LLVM() llvm.Value {
	return fn.LLVMVal
}

func (fn *FuncValue) LValue() bool {
	return false
}

type ConstString struct {
	Str     string
	LLVMVal llvm.Value
}

func (lit *ConstString) RValue(expected_type Type) (TypedValue, *UDiag) {
	str_ty := GetStringType()
	if expected_type == nil || expected_type.Eq(str_ty) {
		return lit, nil
	}
	return nil, TypeMismatchDiag(expected_type, str_ty)
}

func (lit *ConstString) Type() Type {
	return GetStringType()
}

func (lit *ConstString) String() string {
	return "\"" + lit.Str + "\""
}

func (lit *ConstString) LLVM() llvm.Value {
	indices := []llvm.Value{llvm.ConstInt(llvm.IntType(64), 0, false), llvm.ConstInt(llvm.IntType(64), 0, false)}
	return llvm.ConstGEP(lit.LLVMVal, indices)
}

func (lit *ConstString) LValue() bool {
	return false
}

type TypedConstInt struct {
	Inner ConstInt
	Ty Type
}

func (lit *TypedConstInt) Type() Type {
	return lit.Ty
}

func (lit *TypedConstInt) LLVM() llvm.Value {
	return llvm.ConstInt(lit.Ty.LLVM(), uint64(lit.Inner.Int.Int64()), lit.Inner.Signed)
}

func (lit *TypedConstInt) LValue() bool {
	return lit.Inner.LValue()
}

func (lit *TypedConstInt) RValue(expected_type Type) (TypedValue, *UDiag) {
	if expected_type == nil || expected_type.Eq(lit.Ty) {
		return lit, nil
	}
	return nil, TypeMismatchDiag(expected_type, lit.Ty)
}

func (lit *TypedConstInt) String() string {
	return lit.Inner.Int.String()
}

type ConstInt struct {
	Int *big.Int
	Signed bool
}

func (lit *ConstInt) RValue(expected_type Type) (TypedValue, *UDiag) {
	var int_ty *IntType
	if expected_type == nil {  // default to int
		int_ty, _ = global_type_factory.IntType(BLTN_TY_INT).(*IntType)
	} else {
		var ok bool
		int_ty, ok = expected_type.(*IntType)
		if !ok {
			udiag := UDiag("Expected type " + expected_type.String() + " but got integer constant")
			return nil, &udiag
		}
	}

	return &TypedConstInt{*lit, int_ty}, nil
}

func (lit *ConstInt) String() string {
	return lit.Int.String()
}

func (lit *ConstInt) LValue() bool {
	return false
}

type Pointer struct {
	Ty  Type
	Val llvm.Value
}

func CreateNilPointer(ty Type) *Pointer {
	return &Pointer{ty, llvm.ConstPointerNull(ty.LLVM())}
}

func (ptr *Pointer) Type() Type {
	return ptr.Ty
}

func (ptr *Pointer) String() string {
	return "<" + ptr.Ty.String() + " value>"
}

func (ptr *Pointer) LLVM() llvm.Value {
	return ptr.Val
}

func (ptr *Pointer) LValue() bool {
	return false
}

func (ptr *Pointer) RValue(expected_type Type) (TypedValue, *UDiag) {
	if expected_type == nil || expected_type.Eq(ptr.Ty) {
		return ptr, nil
	}
	return nil, TypeMismatchDiag(expected_type, ptr.Ty)
}

