package main

import (
	"github.com/btcsuite/btcd/wire"
	"testing"
)

func NewHash() *wire.ShaHash {
	hash, _ := wire.NewShaHash([]byte{ // Make go vet happy.
		0xdc, 0xe9, 0x69, 0x10, 0x94, 0xda, 0x23, 0xc7,
		0xe7, 0x67, 0x13, 0xd0, 0x75, 0xd4, 0xa1, 0x0b,
		0x79, 0x40, 0x08, 0xa6, 0x36, 0xac, 0xc2, 0x4b,
		0x26, 0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	})

	return hash
}

func NewInvTxMsg() *wire.MsgInv {
	hash := NewHash()
	invVect := wire.NewInvVect(wire.InvTypeTx, hash)

	msg := wire.NewMsgInv()
	msg.AddInvVect(invVect)

	return msg
}

func NewInvBlkMsg() *wire.MsgInv {
	hash := NewHash()
	invVect := wire.NewInvVect(wire.InvTypeBlock, hash)

	msg := wire.NewMsgInv()
	msg.AddInvVect(invVect)

	return msg
}

func TestDecode(*testing.T) {
}
