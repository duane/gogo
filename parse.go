package main

import "math/big"

func parseInt(lit string) *ConstInt {
	Int := big.NewInt(0)

	assert(len(lit) > 0, "Got an empty integer literal!")
	ok := false
	if len(lit) == 1 {
		_, ok = Int.SetString(lit, 10)
	} else if len(lit) == 2 {
		if lit[0] == '0' {
			_, ok = Int.SetString(lit[1:2], 8)
		}
	} else {
		if lit[0] == '0' {
			if lit[1] == 'x' || lit[1] == 'X' {
				_, ok = Int.SetString(lit[2:], 16)
			} else {
				_, ok = Int.SetString(lit[1:], 8)
			}
		} else {
			_, ok = Int.SetString(lit, 10)
		}
	}
	if !ok {
		return nil
	}

	return &ConstInt{Int, true}
}
