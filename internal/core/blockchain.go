package core

type Blockchain interface {
	VerifyTransaction(tx *Transaction) error
	AddTransaction(tx *Transaction)

	VerifyBlock(b *Block) error
	AddBlock(b *Block)

	GenerateBlock(b *Block) *Block
	GenerateBlockTo(addr Hash160, txs []*Transaction) *Block
}
