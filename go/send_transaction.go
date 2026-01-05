package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

func main() {
	// === 配置部分 ===
	// ⚠️ 替换成你的信息！
	infuraKey := "c5daace64d64444790a8d4bdd7c027a6"
	privateKeyStr := "2a18c62db90c2e2120d3a4a87ac4ea6574fcc476009792e29a9fb8e7b9d99e2f" // 64个字符，没有0x

	// 接收地址（示例，可以改）
	toAddress := common.HexToAddress("0xAa1e61Bb5b5f43eF299DB79380790e2e0d4c07fb")

	// === 1. 连接网络 ===
	client, err := ethclient.Dial("https://sepolia.infura.io/v3/" + infuraKey)
	if err != nil {
		log.Fatal("连接失败:", err)
	}

	// === 2. 准备账户 ===
	privateKey, err := crypto.HexToECDSA(privateKeyStr)
	if err != nil {
		log.Fatal("私钥错误:", err)
	}

	publicKey := privateKey.Public().(*ecdsa.PublicKey)
	fromAddress := crypto.PubkeyToAddress(*publicKey)

	fmt.Println("发送地址:", fromAddress.Hex())

	// === 3. 获取必要信息 ===
	// 余额
	balance, _ := client.BalanceAt(context.Background(), fromAddress, nil)
	fmt.Printf("余额: %s wei\n", balance.String())

	// nonce
	nonce, _ := client.PendingNonceAt(context.Background(), fromAddress)

	// gas价格
	gasPrice, _ := client.SuggestGasPrice(context.Background())

	// 链ID
	chainID, _ := client.NetworkID(context.Background())

	// === 4. 创建交易 ===
	// 转账 0.0001 ETH
	value := big.NewInt(100000000000000) // 0.0001 ETH

	tx := types.NewTransaction(
		nonce,
		toAddress,
		value,
		21000,    // gas limit
		gasPrice, // gas price
		nil,      // data
	)

	// === 5. 签名 ===
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		log.Fatal("签名失败:", err)
	}

	// === 6. 发送 ===
	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		log.Fatal("发送失败:", err)
	}

	fmt.Println("\n✅ 交易成功发送!")
	fmt.Println("交易哈希:", signedTx.Hash().Hex())
	fmt.Println("查看链接: https://sepolia.etherscan.io/tx/" + signedTx.Hash().Hex())
}
