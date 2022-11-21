package p2p

import (
	"gocoin/core"
	"gocoin/marshal"
)

const (
	S_HEADER  = 20
	S_COMMAND = 12

	CMD_GETADDR   = "getaddr"
	CMD_ADDR      = "addr"
	CMD_GETBLOCKS = "getblocks"
	CMD_INV       = "inv"
	CMD_GETDATA   = "getdata"
	CMD_BLOCK     = "block"
	CMD_TX        = "tx"

	INV_TX    = 1
	INV_BLOCK = 2
)

var HEADER_MAGIC = [4]byte{0xf9, 0xbe, 0xb4, 0xd9}

type Header struct {
	Magic    [4]byte
	Command  string
	SPayload uint32
}

func (h *Header) ToBytes() []byte {
	buf := make([]byte, 20)
	copy(buf[0:4], h.Magic[:])
	copy(buf[4:16], h.Command[:])
	copy(buf[16:20], marshal.Uint32ToBytes(h.SPayload))

	return buf
}

func (h *Header) SetBytes(buf []byte) {
	copy(h.Magic[:], buf[0:4])

	ptr := 4
	for i := 0; i < S_COMMAND; i++ { // find null bytes
		if buf[ptr] == 0 {
			break
		}
		ptr++
	}
	h.Command = string(buf[4:ptr])

	h.SPayload = marshal.Uint32FromBytes(buf[16:20])
}

func ReceiveHeader(data []byte) Header {
	var h Header
	h.SetBytes(data)

	return h
}

func SendGetAddr() []byte {
	h := Header{
		Magic:    HEADER_MAGIC,
		Command:  CMD_GETADDR,
		SPayload: 0,
	}

	return h.ToBytes()
}

type MultiAddr struct {
	SAddr uint32
	Addr  string
}

type MsgAddr struct {
	NAddr uint32      // number of addresses
	Addrs []MultiAddr // multi-address
}

func (m *MsgAddr) ToBytes() []byte {
	buf := make([]byte, 4)
	copy(buf[0:4], marshal.Uint32ToBytes(m.NAddr))

	for _, addr := range m.Addrs {
		buf = append(buf, marshal.Uint32ToBytes(addr.SAddr)...)
		buf = append(buf, []byte(addr.Addr)...)
	}

	return buf
}

func (m *MsgAddr) SetBytes(buf []byte) {
	m.NAddr = marshal.Uint32FromBytes(buf[0:4])
	buf = buf[4:]

	for i := uint32(0); i < m.NAddr; i++ {
		saddr := marshal.Uint32FromBytes(buf[0:4])
		addr := string(buf[4 : 4+saddr])

		m.Addrs = append(m.Addrs, MultiAddr{SAddr: saddr, Addr: addr})
		buf = buf[4+saddr:]
	}
}

func SendAddr(addrs []string) []byte {
	buf := make([]byte, 0)

	multiAddrs := make([]MultiAddr, len(addrs))
	for i, addr := range addrs {
		multiAddrs[i] = MultiAddr{SAddr: uint32(len(addr)), Addr: addr}
	}
	payload := MsgAddr{
		NAddr: uint32(len(addrs)),
		Addrs: multiAddrs,
	}

	buf = append(buf, payload.ToBytes()...)
	h := Header{
		Magic:    HEADER_MAGIC,
		Command:  CMD_ADDR,
		SPayload: uint32(len(buf)),
	}

	return append(h.ToBytes(), buf...)
}

func ReceiveAddr(data []byte) []string {
	var addrs []string

	payload := MsgAddr{}
	payload.SetBytes(data[:])

	for _, addr := range payload.Addrs {
		addrs = append(addrs, addr.Addr)
	}

	return addrs
}

type MsgGetBlocks struct {
	NBlocks     uint32
	BlockHashes []core.Hash256
	EndHash     core.Hash256
}

func (m *MsgGetBlocks) ToBytes() []byte {
	buf := make([]byte, 0)

	buf = append(buf, marshal.Uint32ToBytes(m.NBlocks)...)
	for _, hash := range m.BlockHashes {
		buf = append(buf, hash[:]...)
	}
	buf = append(buf, m.EndHash[:]...)

	return buf
}

func (m *MsgGetBlocks) SetBytes(buf []byte) {
	m.NBlocks = marshal.Uint32FromBytes(buf[0:4])
	buf = buf[4:]

	for i := uint32(0); i < m.NBlocks; i++ {
		var hash core.Hash256
		copy(hash[:], buf[0:32])
		m.BlockHashes = append(m.BlockHashes, hash)
		buf = buf[32:]
	}

	copy(m.EndHash[:], buf[0:32])
}

func SendGetBlocks(payload MsgGetBlocks) []byte {
	buf := make([]byte, 0)

	buf = append(buf, payload.ToBytes()...)
	h := Header{
		Magic:    HEADER_MAGIC,
		Command:  CMD_GETBLOCKS,
		SPayload: uint32(len(buf)),
	}

	return append(h.ToBytes(), buf...)
}

func ReceiveGetBlocks(data []byte) MsgGetBlocks {
	payload := MsgGetBlocks{}
	payload.SetBytes(data[:])

	return payload
}

type Inventory struct {
	TypeId uint32
	Hash   core.Hash256
}

type MsgInv struct {
	NInv    uint32
	InvList []Inventory
}

func (m *MsgInv) ToBytes() []byte {
	buf := make([]byte, 0)

	buf = append(buf, marshal.Uint32ToBytes(m.NInv)...)
	for _, inv := range m.InvList {
		buf = append(buf, marshal.Uint32ToBytes(inv.TypeId)...)
		buf = append(buf, inv.Hash[:]...)
	}

	return buf
}

func (m *MsgInv) SetBytes(buf []byte) {
	m.NInv = marshal.Uint32FromBytes(buf[0:4])
	buf = buf[4:]

	for i := uint32(0); i < m.NInv; i++ {
		var inv Inventory
		inv.TypeId = marshal.Uint32FromBytes(buf[0:4])
		copy(inv.Hash[:], buf[4:36])

		m.InvList = append(m.InvList, inv)
		buf = buf[36:]
	}
}

func SendInv(invList []Inventory) []byte {
	buf := make([]byte, 0)

	payload := MsgInv{
		NInv:    uint32(len(invList)),
		InvList: invList,
	}

	buf = append(buf, payload.ToBytes()...)
	h := Header{
		Magic:    HEADER_MAGIC,
		Command:  CMD_INV,
		SPayload: uint32(len(buf)),
	}

	return append(h.ToBytes(), buf...)
}

func ReceiveInv(data []byte) []Inventory {
	var invList []Inventory

	payload := MsgInv{}
	payload.SetBytes(data[:])

	for _, inv := range payload.InvList {
		invList = append(invList, inv)
	}

	return invList
}

type MsgGetData MsgInv

func SendGetData(invList []Inventory) []byte {
	buf := make([]byte, 0)

	payload := MsgInv{
		NInv:    uint32(len(invList)),
		InvList: invList,
	}

	buf = append(buf, payload.ToBytes()...)
	h := Header{
		Magic:    HEADER_MAGIC,
		Command:  CMD_GETDATA,
		SPayload: uint32(len(buf)),
	}

	return append(h.ToBytes(), buf...)
}

func ReceiveGetData(data []byte) []Inventory {
	return ReceiveInv(data)
}

func SendBlock(block *core.Block) []byte {
	buf := make([]byte, 0)
	buf = append(buf, marshal.Block(block)...)

	h := Header{
		Magic:    HEADER_MAGIC,
		Command:  CMD_BLOCK,
		SPayload: uint32(len(buf)),
	}

	return append(h.ToBytes(), buf...)
}

func ReceiveBlock(data []byte) *core.Block {
	return marshal.UBlock(data)
}

func SendTx(tx *core.Transaction) []byte {
	buf := make([]byte, 0)
	buf = append(buf, marshal.Transaction(tx)...)

	h := Header{
		Magic:    HEADER_MAGIC,
		Command:  CMD_TX,
		SPayload: uint32(len(buf)),
	}

	return append(h.ToBytes(), buf...)
}

func ReceiveTx(data []byte) *core.Transaction {
	return marshal.UTransaction(data)
}
