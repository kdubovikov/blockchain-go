package main

import (
	"fmt"
	"strconv"

	"github.com/kdubovikov/blockchain-go/blockchain"
)

func main() {
	chain := blockchain.InitBlockChain()
	chain.AddBlock("First block")
	chain.AddBlock("Second block")
	chain.AddBlock("Third block")

	for _, block := range chain.Blocks {
		fmt.Printf("Previous Hash:\t%x\n", block.PrevHash)
		fmt.Printf("Data:\t%s\n", block.Data)
		fmt.Printf("Hash:\t%x\n", block.Hash)

		pow := blockchain.NewProof(block)
		fmt.Printf("PoW: %s\n", strconv.FormatBool(pow.Validate()))
		fmt.Println()
	}
}
