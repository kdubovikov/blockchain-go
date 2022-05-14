package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"runtime/debug"
	"strconv"

	"github.com/kdubovikov/blockchain-go/blockchain"
	"github.com/kdubovikov/blockchain-go/network"
	"github.com/kdubovikov/blockchain-go/wallet"
)

type CommandLine struct{}

func (cli *CommandLine) printUsage() {
	fmt.Println("Usage: ")
	fmt.Println(" getbalance - address ADDRESS - get the balance for the given address")
	fmt.Println(" createblockchain - address ADDRESS - creates a blockchain")
	fmt.Println(" printchain - print the blocks in the chain")
	fmt.Println(" send -from FROM -to TO -amount AMOUNT - send the specified amount from one address to another ")
	fmt.Println(" createwallet - creates new wallet")
	fmt.Println(" listaddresses - lists addresses of all wallets")
	fmt.Println(" reindexutxo - Rebuilds the UTXO set")
	fmt.Println(" startnode -miner ADDRESS - Start a node with ID specified in NODE_ID env. var. -miner enables mining")
}

func (cli *CommandLine) validateArgs() {
	if len(os.Args) < 2 {
		cli.printUsage()
		runtime.Goexit()
	}
}

func (cli *CommandLine) StartNode(nodeID, minerAddress string) {
	fmt.Printf("Starting Node %s\n", nodeID)

	if len(minerAddress) > 0 {
		if wallet.ValidateAddress(minerAddress) {
			fmt.Println("Mining is on. Address to receive rewards: ", minerAddress)
		} else {
			log.Panic("Wrong miner address!")
		}
	}
	network.StartServer(nodeID, minerAddress)
}

func (cli *CommandLine) createBlockChain(address, nodeId string) {
	if !wallet.ValidateAddress(address) {
		log.Panic("Address is not Valid")
	}
	chain := blockchain.InitBlockChain(address, nodeId)
	chain.Database.Close()

	UTXOSet := blockchain.UTXOSet{Blockchain: chain}
	UTXOSet.Reindex()

	fmt.Println("Finished")
}

func (cli *CommandLine) getBalance(address, nodeId string) {
	if !wallet.ValidateAddress(address) {
		log.Panic("Address is not Valid")
	}
	chain := blockchain.ContinueBlockChain(nodeId)
	UTXOSet := blockchain.UTXOSet{Blockchain: chain}
	defer chain.Database.Close()

	balance := 0

	pubKeyHash := wallet.Base58Decode([]byte(address))
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-4]
	UTXOs := UTXOSet.FindUTXO(pubKeyHash)

	for _, out := range UTXOs {
		balance += out.Value
	}

	fmt.Printf("Balance of %s: %d\n", address, balance)
}

func (cli *CommandLine) send(from, to string, amount int, nodeId string, mineNow bool) {
	if !wallet.ValidateAddress(from) {
		log.Panic("Address is not Valid")
	}
	if !wallet.ValidateAddress(to) {
		log.Panic("Address is not Valid")
	}
	chain := blockchain.ContinueBlockChain(nodeId)
	UTXOSet := blockchain.UTXOSet{chain}
	defer chain.Database.Close()

	wallets, err := wallet.CreateWallets(nodeId)
	if err != nil {
		log.Panic(err)
	}
	wallet := wallets.GetWallet(from)

	tx := blockchain.NewTransaction(&wallet, to, amount, &UTXOSet)
	if mineNow {
		cbTx := blockchain.CoinbaseTx(from, "")
		txs := []*blockchain.Transaction{cbTx, tx}
		block := chain.MineBlock(txs)
		UTXOSet.Update(block)
	} else {
		network.SendTx(network.KnownNodes[0], tx)
		fmt.Println("send tx")
	}

	fmt.Println("Success!")
}

func (cli *CommandLine) printChain(nodeId string) {
	chain := blockchain.ContinueBlockChain(nodeId)
	iter := chain.Iterator()

	for {
		block := iter.Next()

		fmt.Printf("Previous Hash:\t%x\n", block.PrevHash)
		fmt.Printf("Hash:\t%x\n", block.Hash)

		pow := blockchain.NewProof(block)
		fmt.Printf("PoW: %s\n", strconv.FormatBool(pow.Validate()))
		fmt.Println()

		for _, tx := range block.Transactions {
			fmt.Println(tx)
		}

		if len(block.PrevHash) == 0 {
			break
		}
	}
}

func (cli *CommandLine) listAddresses(nodeId string) {
	wallets, _ := wallet.CreateWallets(nodeId)
	addrsses := wallets.GetAllAddresses()

	for _, address := range addrsses {
		fmt.Println(address)
	}
}

func (cli *CommandLine) addWallet(nodeId string) {
	wallets, _ := wallet.CreateWallets(nodeId)
	address := wallets.AddWallet()
	wallets.SaveFile(nodeId)

	fmt.Printf("New address is %s\n", address)
}

func (cli *CommandLine) reindexUTXO(nodeId string) {
	chain := blockchain.ContinueBlockChain(nodeId)
	defer chain.Database.Close()
	UTXOSet := blockchain.UTXOSet{chain}
	UTXOSet.Reindex()

	count := UTXOSet.CountTransactions()
	fmt.Printf("Done! There are %d transactions in the UTXO set.\n", count)
}

func (cli *CommandLine) run() {
	cli.validateArgs()

	nodeId := os.Getenv("NODE_ID")
	if nodeId == "" {
		fmt.Printf("NODE_ID env is not set!")
		runtime.Goexit()
	}

	getBalanceCmd := flag.NewFlagSet("getbalance", flag.ExitOnError)
	createblockchainCmd := flag.NewFlagSet("createblockchain", flag.ExitOnError)
	sendCmd := flag.NewFlagSet("send", flag.ExitOnError)
	printChainCmd := flag.NewFlagSet("print", flag.ExitOnError)
	createWalletCmd := flag.NewFlagSet("createwallet", flag.ExitOnError)
	listAddrssesCmd := flag.NewFlagSet("listaddresses", flag.ExitOnError)
	reindexUTXOCmd := flag.NewFlagSet("reindexutxo", flag.ExitOnError)
	startNodeCmd := flag.NewFlagSet("startnode", flag.ExitOnError)

	getBalanceAddress := getBalanceCmd.String("address", "", "address to to get the balace for")
	createBlockchainAddress := createblockchainCmd.String("address", "", "The address to send genesis block reward to")
	sendFrom := sendCmd.String("from", "", "Source wallet address")
	sendTo := sendCmd.String("to", "", "Destination wallet address")
	sendAmount := sendCmd.Int("amount", 0, "Amount to send")
	startNodeMiner := startNodeCmd.String("miner", "", "Enable mining mode and send reward to ADDRESS")
	sendMine := sendCmd.Bool("mine", false, "Mine immediately on the same node")

	switch os.Args[1] {
	case "getbalance":
		err := getBalanceCmd.Parse(os.Args[2:])
		blockchain.Handle(err)

	case "createblockchain":
		err := createblockchainCmd.Parse(os.Args[2:])
		blockchain.Handle(err)

	case "reindexutxo":
		err := reindexUTXOCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "startnode":
		err := startNodeCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "printchain":
		err := printChainCmd.Parse(os.Args[2:])
		blockchain.Handle(err)

	case "send":
		err := sendCmd.Parse(os.Args[2:])
		blockchain.Handle(err)

	case "createwallet":
		err := createWalletCmd.Parse(os.Args[2:])
		blockchain.Handle(err)

	case "listaddresses":
		err := listAddrssesCmd.Parse(os.Args[2:])
		blockchain.Handle(err)

	default:
		cli.printUsage()
		runtime.Goexit()
	}

	if getBalanceCmd.Parsed() {
		if *getBalanceAddress == "" {
			getBalanceCmd.Usage()
			runtime.Goexit()
		}
		cli.getBalance(*getBalanceAddress, nodeId)
	}

	if createblockchainCmd.Parsed() {
		if *createBlockchainAddress == "" {
			createblockchainCmd.Usage()
			runtime.Goexit()
		}
		cli.createBlockChain(*createBlockchainAddress, nodeId)
	}

	if printChainCmd.Parsed() {
		cli.printChain(nodeId)
	}

	if sendCmd.Parsed() {
		if *sendFrom == "" || *sendTo == "" || *sendAmount <= 0 {
			sendCmd.Usage()
			runtime.Goexit()
		}

		cli.send(*sendFrom, *sendTo, *sendAmount, nodeId, *sendMine)
	}

	if createWalletCmd.Parsed() {
		cli.addWallet(nodeId)
	}

	if listAddrssesCmd.Parsed() {
		cli.listAddresses(nodeId)
	}
	if reindexUTXOCmd.Parsed() {
		cli.reindexUTXO(nodeId)
	}
	if startNodeCmd.Parsed() {
		nodeID := os.Getenv("NODE_ID")
		if nodeID == "" {
			startNodeCmd.Usage()
			runtime.Goexit()
		}
		cli.StartNode(nodeID, *startNodeMiner)
	}
}

func main() {
	defer os.Exit(0)
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("stacktrace from panic: \n" + string(debug.Stack()))
		}
	}()

	cli := CommandLine{}
	cli.run()
}
