package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

func main() {
	// 1. 配置
	infuraKey := "c5daace64d64444790a8d4bdd7c027a6"
	txHashStr := "0xfc805ef4578d2dcf78a34c8264823b85c83b5cec8296bc7737e57ebc6964d576" // 替换成你的交易哈希

	// 2. 连接网络
	client, err := ethclient.Dial("https://sepolia.infura.io/v3/" + infuraKey)
	if err != nil {
		log.Fatal("连接失败:", err)
	}

	// 3. 查询交易
	txHash := common.HexToHash(txHashStr)
	tx, isPending, err := client.TransactionByHash(context.Background(), txHash)
	if err != nil {
		log.Fatal("查询失败:", err)
	}

	// 4. 打印基本信息
	fmt.Printf("是否待处理: %v\n", isPending)
	fmt.Printf("交易金额: %s wei\n", tx.Value().String())
	if tx.To() != nil {
		fmt.Printf("接收地址: %s\n", tx.To().Hex())
	}

	// 5. 查询收据（如果已确认）
	if !isPending {
		receipt, err := client.TransactionReceipt(context.Background(), txHash)
		if err == nil {
			fmt.Printf("\n交易已确认！\n")
			fmt.Printf("区块号: %d\n", receipt.BlockNumber)
			fmt.Printf("状态: ")
			if receipt.Status == 1 {
				fmt.Println("成功")
			} else {
				fmt.Println("失败")
			}
		}
	}
}
