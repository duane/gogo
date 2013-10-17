package main

var global_type_factory TypeFactory

type TypeFactory struct {
	Target Target
}

func (factory TypeFactory) IntType(ty uint) Type {
	return &IntType{ty, factory.Target}
}