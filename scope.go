package main

import "fmt"

type Scope struct {
	Types  *TypeMap
	Values *ValueMap
	Parent *Scope
}

func CreateScope() *Scope {
	types := make(TypeMap, 0)
	values := make(ValueMap, 0)
	scope := &Scope{&types, &values, nil}
	return scope
}

/*
 * returns type at `ident`, or nil if `ident` is not bound to a type.
 */

func (scope *Scope) lookupType(ident string) Type {
	targetScope := scope
	var ty Type = nil
	for targetScope != nil {
		tyInner, ok := (*targetScope.Types)[ident]
		ty = tyInner // TODO: fix this hack.
		if ok {
			break
		}
		targetScope = targetScope.Parent
	}
	return ty
}

/*
 * addType adds type `ty` to scope iff `ident` is not bound to a type already. Returns `true` on success.
 *
 */
func (scope *Scope) addType(ident string, ty Type) bool {
	existing := scope.lookupType(ident)
	if existing != nil {
		return false
	}

	// add type to immediate scope
	(*scope.Types)[ident] = ty
	return true
}

/*
 * addTypeAlias first looks up type at identifier `rTypeID`.
 * Then, it adds an alias at `lTypeID` which references the rvalue type.
 * In short, types[lTypeID] = Alias{types[rTypeID]}
 *
 * returns `nil` on success.
 *
 * Q: Why does this function return a `*UDiag` instead of a `bool`?
 * A: Because there are multiple reasons for failure here.
 *
 */
func (scope *Scope) addTypeAlias(lTypeID string, rTypeID string) *UDiag {
	rType := scope.lookupType(rTypeID)
	if rType == nil {
		udiag := UDiag(fmt.Sprintf("Type \"%s\" not found.", rTypeID))
		return &udiag
	}
	lType := &AliasType{lTypeID, rType}
	if !scope.addType(lTypeID, lType) {
		udiag := UDiag(fmt.Sprintf("Type \"%s\" already exists.", lTypeID))
		return &udiag
	}
	return nil
}

/*
 * Looks up the value bound to `ident` in the Value map.
 */
func (scope *Scope) lookupVar(ident string) *BoundVar {
	targetScope := scope
	for targetScope != nil {
		v, ok := (*targetScope.Values)[ident]
		if ok {
			assert(v != nil, "Found a nil binding!")
			return v
		}
		targetScope = targetScope.Parent
	}
	return nil
}

func (scope *Scope) addValue(ident string, val UntypedValue) bool {
	existing := scope.lookupVar(ident)
	if existing == nil {
		(*scope.Values)[ident] = &BoundVar{ident, false, val}
		return true
	}
	return false
}

func (scope *Scope) dump() {
	fmt.Printf("Types:\n")
	for name, ty := range *scope.Types {
		fmt.Printf("\t%s: %s\n", name, ty.String())
	}

	fmt.Printf("Values:\n")
	for name, val := range *scope.Values {
		fmt.Printf("\t%s: %s\n", name, val.String())
	}
}

func (scope *Scope) createChild() *Scope {
	types := make(TypeMap)
	values := make(ValueMap)
	return &Scope{&types, &values, scope}
}
