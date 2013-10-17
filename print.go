package main

func (trans *Translator) initPrint() {
	trans.addExternFunction("print_int", nil, trans.Scope.lookupType("int64"))
	trans.addExternFunction("print_uint", nil, trans.Scope.lookupType("uint64"))
}
