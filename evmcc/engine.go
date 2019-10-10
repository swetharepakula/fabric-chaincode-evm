package main

import "time"

type Blockchain struct {
}

func (*Blockchain) LastBlockHeight() uint64 {
	panic("Block Height shouldn't be called")
}

func (*Blockchain) LastBlockTime() time.Time {
	panic("Block Time shouldn't be called")
}

func (*Blockchain) BlockHash(height uint64) ([]byte, error) {
	panic("Block Hash shouldn't be called")
}
