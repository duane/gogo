package main

import "llvm"
import "go/ast"
import "go/token"
import "fmt"

type Translator struct {
	mod     llvm.Module
	Scope   *Scope
	builder llvm.Builder
	Parent  *Translator
	Target  Target
	LLns    *LLVMNamespace
}

type Assignable interface {
	TypedValue
	BuildAssign(*Block, TypedValue) *GoDiag
}

func (v *BoundVar) BuildAssign(block *Block, val TypedValue) *GoDiag {
	assert(!v.Const, "Attempting to assign to a const value.")
	v.Val = val
	return nil
}

func CreateTranslator() *Translator {
	return &Translator{llvm.NullModule(), CreateScope(), llvm.CreateBuilder(), nil, CreateNativeTarget(), nil}
}

func (trans *Translator) translateType(tyExpr ast.Expr) (Type, *GoDiag) {
	switch exprType := tyExpr.(type) {
	case *ast.Ident:
		id, _ := tyExpr.(*ast.Ident)
		ty := trans.Scope.lookupType(id.Name)
		if ty == nil {
			return nil, DiagFromAST(tyExpr, "Unknown type \"%s\".", id.Name)
		}
		return ty, nil
	case *ast.StarExpr:
		star, _ := tyExpr.(*ast.StarExpr)
		atType, diag := trans.translateType(star.X)
		if diag != nil {
			return nil, diag
		}
		return &PointerType{atType}, nil
	default:
		return nil, DiagFromAST(tyExpr, "Unknown internal type expression type: %T.", exprType)
	}
	panic("Unreachable code. Please fix!")
}

func (block *Block) translateCallExpr(call *ast.CallExpr) (TypedValue, *GoDiag) {
	funExpr := call.Fun
	funValue, diag := block.translateExprRHS(funExpr)
	if diag != nil {
		return nil, diag
	}

	funcValue, ok := funValue.(*FuncValue)
	if !ok {
		return nil, DiagFromAST(funExpr, "Given expression not a function!")
	}

	funType, ok := funcValue.Type().(*FuncType)
	if !ok {
		panic("Function value does not have function type!")
	}

	if len(funType.Params) != len(call.Args) {
		return nil, DiagFromAST(call, "Expected %d arguments, found %d!", len(funType.Params), len(call.Args))
	}

	llvmArgs := make([]llvm.Value, len(call.Args))
	for i, argExpr := range call.Args {
		untyped, diag := block.translateExprRHS(argExpr)
		if diag != nil {
			return nil, diag
		}
		typed_val, udiag := untyped.RValue(funType.Params[i])
		if udiag != nil {
			return nil, BindDiagToAST(argExpr, *udiag)
		}
		llvmArgs[i] = typed_val.LLVM()
	}

	// build call expression
	block.Builder.BuildCall(funcValue.LLVM(), llvmArgs, "")

	return funcValue, nil
}

func (block *Block) translateStringLit(lit string) TypedValue {
	// TODO: Proper string escaping.
	unescaped := lit[1 : len(lit)-1]
	byteTy := block.Scope.lookupType("byte")
	strTy := llvm.ArrayType(byteTy.LLVM(), uint(len(unescaped))+1)
	llstr := llvm.ConstString(unescaped, false)
	llvmVal := block.Trans.LLns.requestStaticConstAlloc(&PointerType{byteTy}, strTy, llstr)
	litVal := &ConstString{unescaped, llvmVal}
	return litVal
}

func (block *Block) translateBasicLit(lit *ast.BasicLit) (UntypedValue, *GoDiag) {
	switch lit.Kind {
	case token.STRING:
		return block.translateStringLit(lit.Value), nil
	case token.INT:
		parsed := parseInt(lit.Value)
		if parsed == nil {
			return nil, DiagFromAST(lit, "Unable to parse integer!")
		}
		return parsed, nil
	default:
		return nil, DiagFromAST(lit, fmt.Sprintf("Unable to translate literal: \"%s\".", lit.Value))
	}
	panic("Unreachable!")
	return nil, nil
}

// Returns LValue associated with expr, or returns error.
func (block *Block) translateExprLHS(expr ast.Expr) (Assignable, *GoDiag) {
	switch expr.(type) {
	case *ast.Ident:
		ident, _ := expr.(*ast.Ident)
		lVal := block.Scope.lookupVar(ident.Name)
		if !lVal.LValue() {
			return nil, DiagFromAST(expr, "Unable to assign to variable \"%s\".", ident)
		}
		return lVal, nil
	default:
		return nil, DiagFromAST(expr, "Expected an lvalue expression.")
	}
	panic("Unreachable, please fix.")
	return nil, nil
}

func (block *Block) translateExprRHS(expr ast.Expr) (UntypedValue, *GoDiag) {
	switch exprTy := expr.(type) {
	case *ast.CallExpr:
		// function call
		callExpr, _ := expr.(*ast.CallExpr)
		rVal, diag := block.translateCallExpr(callExpr)
		if diag != nil {
			return nil, diag
		}
		return rVal, nil
	case *ast.Ident:
		// identifier lookup
		ident, _ := expr.(*ast.Ident)
		identVal := block.Scope.lookupVar(ident.Name)
		if identVal == nil {
			return nil, DiagFromAST(expr, "Unknown identifier \"%s\".", ident)
		}
		return identVal, nil
	case *ast.BasicLit:
		basic, _ := expr.(*ast.BasicLit)
		return block.translateBasicLit(basic)
	default:
		return nil, DiagFromAST(expr, "Cannot translate expr of type: %T\n", exprTy)
		break
	}
	return nil, nil
}

func (block *Block) translateExprRHSTyped(expr ast.Expr, expected_type Type) (TypedValue, *GoDiag) {
	untyped, diag := block.translateExprRHS(expr)
	if diag != nil {
		return nil, diag
	}
	typed, udiag := untyped.RValue(expected_type)
	if udiag != nil {
		return nil, BindDiagToAST(expr, *udiag)
	}
	return typed, nil
}

func (block *Block) translateReturn(ret *ast.ReturnStmt) *GoDiag {
	// first check that we only return one expression.
	if ret.Results == nil {
		// ResultTy must be nil too; otherwise, function must provide a value.
		if block.ResultTy != nil {
			return DiagFromAST(ret, "Function is expected to return a value!")
		}
		block.Builder.BuildRetVoid()
		return nil
	}
	if len(ret.Results) > 1 {
		return DiagFromAST(ret, "Only single-value return is implemented at this time.")
	}
	result, diag := block.translateExprRHSTyped(ret.Results[0], block.ResultTy)
	if diag != nil {
		return diag
	}

	// ok, now return the result.
	block.Builder.BuildRet(result.LLVM())
	return nil
}

func (block *Block) translateVarDecl(gen *ast.GenDecl) *GoDiag {
	cnst := false
	if gen.Tok == token.CONST {
		cnst = true
	}

	for _, spec := range gen.Specs {
		valueSpec, ok := spec.(*ast.ValueSpec)
		assert(ok, "Expected *ast.ValueSpec, but got different type!")
		assert(len(valueSpec.Names) > 0, "Found a spec with zero names to assign to.")

		var ty Type
		var diag *GoDiag
		if valueSpec.Type != nil {
			ty, diag = block.Trans.translateType(valueSpec.Type)
			if diag != nil {
				return diag
			}
		}

		if cnst {
			return DiagFromAST(valueSpec, "Const declarations are not yet implemented.")
		} else {
			// translate a true variable declaration.
			if ty == nil {
				return DiagFromAST(valueSpec, "Unable to handle non-typed variable declarations at this time.")
			}

			// Make sure that the variables are entirely initialized to zero values or entirely initialized to expressions.
			if len(valueSpec.Values) != 0 && len(valueSpec.Values) != len(valueSpec.Names) {
				return DiagFromAST(valueSpec, "Partial initialization of variables in a variable declaration is not allowed.")
			}

			// for each variable...
			for idx, name := range valueSpec.Names {
				// first check to make sure that the name is not already used as a variable.
				if block.Scope.lookupVar(name.Name) != nil {
					return DiagFromAST(name, "A variable already exists with this identifier.")
				}

				if len(valueSpec.Values) == 0 {
					// if there's no initializer, initialize to zero value.
					rValue = ty.Zero(block.Trans.LLns)
				} else {
					rExpr := valueSpec.Values[idx]
					rValue, diag = block.translateExprRHS(rExpr)
					if diag != nil {
						return diag
					}
					if !rValue.Type().Eq(ty) {
						return DiagFromAST(rExpr, "Expected initializer of type \"%s\", but found type \"%s\".", ty.String(), rValue.Type().String())
					}
					return DiagFromAST(rExpr, "Expected initializer of type \"%s\", but found type \"%s\".", ty.String(), rValue.Type().String())
				}

				// now set the value.
				block.Scope.addValue(name.Name, rValue)
			}

			// we've now translated everything; return success
			return nil
		}
	}
	panic("Unreachable.")
}

func (block *Block) translateGenDecl(gen *ast.GenDecl) *GoDiag {
	switch gen.Tok {
	
	case token.VAR, token.CONST:
		return block.translateVarDecl(gen)
	default:
		return DiagFromAST(gen, "General declaration type \"%s\" not implemented yet.", gen.Tok)
	}
	panic("Unreachable!")
	return nil
}

// TODO: in cases such as `a, b = b, a`, the assignment may be incorrect.

func (block *Block) translateAssign(assign *ast.AssignStmt) *GoDiag {
	// for now, we can't do short assignments, just long assignments.
	if len(assign.Lhs) != len(assign.Rhs) {
		return DiagFromAST(assign, "Every variable must have an equivalent rValue")
	}
	for idx, lExpr := range assign.Lhs {
		lValue, diag := block.translateExprLHS(lExpr)
		if diag != nil {
			return diag
		}
		rExpr := assign.Rhs[idx]
		rValue, diag := block.translateExprRHS(rExpr, nil)
		if diag != nil {
			return diag
		}

		diag = lValue.BuildAssign(block, rValue)
		if diag != nil {
			return diag
		}
	}
	return nil
}

func (block *Block) translateDecl(declStmt *ast.DeclStmt) *GoDiag {
	decl := declStmt.Decl
	switch declTy := decl.(type) {
	case *ast.GenDecl:
		gen, _ := decl.(*ast.GenDecl)
		return block.translateGenDecl(gen)
	default:
		return DiagFromAST(declStmt, "Unknown block declaration type: %T.", declTy)
	}
	panic("Unreachable! Fix.")
}

func (block *Block) translateStatement(statement ast.Stmt) *GoDiag {
	switch statementType := statement.(type) {
	case *ast.ExprStmt:
		expr, _ := statement.(*ast.ExprStmt)
		_, diag := block.translateExprRHS(expr.X)
		return diag
	case *ast.ReturnStmt:
		ret, _ := statement.(*ast.ReturnStmt)
		return block.translateReturn(ret)
	case *ast.DeclStmt:
		decl, _ := statement.(*ast.DeclStmt)
		return block.translateDecl(decl)
	case *ast.AssignStmt:
		assign, _ := statement.(*ast.AssignStmt)
		return block.translateAssign(assign)
	default:
		return DiagFromAST(statement, "Unknown statement type: %T", statementType)
		break
	}
	return nil
}

func (trans *Translator) CreateBlockForFunction(llvmFunc llvm.Value, fnTy FuncType) *Block {
	scope := trans.Scope.createChild()
	block := llvm.AppendBasicBlock(llvmFunc, "entry")
	builder := llvm.CreateBuilder()
	builder.PositionBuilderAtEnd(block)
	resultTy := fnTy.Result
	return &Block{scope, block, builder, resultTy, trans}
}

func (trans *Translator) translateFuncDecl(decl *ast.FuncDecl) *GoDiag {
	fnTypeDecl := decl.Type
	if len(fnTypeDecl.Results.List) > 1 {
		return DiagFromAST(decl, "Returning more than one value is not yet permitted.")
	}

	paramTypes := make([]Type, 0)
	paramList := fnTypeDecl.Params
	for _, field := range paramList.List {
		ty, diag := trans.translateType(field.Type)
		if diag != nil {
			return diag
		}
		for i := 0; i < len(field.Names); i++ { // for every variable in the field
			paramTypes = append(paramTypes, ty)
		}
	}

	var resultTy Type
	if len(fnTypeDecl.Results.List) == 0 {
		resultTy = nil
	} else {
		ty, diag := trans.translateType(fnTypeDecl.Results.List[0].Type)
		if diag != nil {
			return diag
		}
		resultTy = ty
	}

	fnTy := FuncType{paramTypes, resultTy}

	llvmFnTy := fnTy.LLVM()

	llvmFn := trans.mod.AddFunction(decl.Name.Name, llvmFnTy)

	block := trans.CreateBlockForFunction(llvmFn, fnTy)

	for _, statement := range decl.Body.List {
		diag := block.translateStatement(statement)
		if diag != nil {
			return diag
		}
	}

	return nil
}

func (trans *Translator) translateDecl(decl ast.Decl) *GoDiag {
	switch declType := decl.(type) {
	case *ast.FuncDecl:
		fDecl, _ := decl.(*ast.FuncDecl)
		if fDecl.Recv != nil {
			return DiagFromAST(fDecl, "Methods not supported yet.")
		}
		return trans.translateFuncDecl(fDecl)
		break
	default:
		return DiagFromAST(decl, "Unsupported Decl type: \"%T\".", declType)
	}
	return nil
}

func (trans *Translator) addExternFunction(name string, result Type, params ...Type) TypedValue {
	ty := &FuncType{params, result}
	llvmVal := trans.mod.AddFunction(name, ty.LLVM())
	llvm.SetLinkage(llvmVal, llvm.ExternalLinkage)
	val := &FuncValue{name, ty, llvmVal}
	trans.Scope.addValue(name, val)
	return val
}

func (trans *Translator) CreateGoScope() {
	scope := CreateScope()
	// initialize base go language type system
	scope.addType("uint8", &IntType{BLTN_TY_UINT8, trans.Target})
	scope.addType("int8", &IntType{BLTN_TY_INT8, trans.Target})
	scope.addType("uint16", &IntType{BLTN_TY_UINT16, trans.Target})
	scope.addType("int16", &IntType{BLTN_TY_INT16, trans.Target})
	scope.addType("uint32", &IntType{BLTN_TY_UINT32, trans.Target})
	scope.addType("int32", &IntType{BLTN_TY_INT32, trans.Target})
	scope.addType("uint64", &IntType{BLTN_TY_UINT64, trans.Target})
	scope.addType("int64", &IntType{BLTN_TY_INT64, trans.Target})
	scope.addType("int", &IntType{BLTN_TY_INT, trans.Target})
	scope.addType("uint", &IntType{BLTN_TY_UINT, trans.Target})

	// type synonyms
	scope.addTypeAlias("byte", "uint8")

	trans.Scope = scope

	// and add in temporary libc linkage.
	trans.addExternFunction("puts", nil, &PointerType{trans.Scope.lookupType("uint8")})
	trans.initPrint()
}

func (trans *Translator) translateFile(file *ast.File, fset *token.FileSet) (llvm.Module, Diag) {
	assert(file != nil, "File is nil.")
	trans.mod = llvm.ModuleCreateWithName(file.Name.Name)
	trans.LLns = CreateNamespace(trans.mod)
	trans.mod.SetTarget(trans.Target.Triple)
	trans.mod.SetDataLayout(trans.Target.DataLayout)
	trans.CreateGoScope()
	if file.Decls == nil {
		return trans.mod, nil
	}

	for i := 0; i < len(file.Decls); i++ {
		decl := file.Decls[i]
		diag := trans.translateDecl(decl)
		if diag != nil {
			diag.fset = fset
			return trans.mod, diag
		}
	}
	return trans.mod, nil
}
