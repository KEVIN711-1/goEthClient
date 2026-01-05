package main

import (
	"context"
	"fmt"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

func main() {
	// === 配置部分 ===
	infuraKey := "c5daace64d64444790a8d4bdd7c027a6"

	// 要查询的交易哈希
	targetTxHash := "0xfc805ef4578d2dcf78a34c8264823b85c83b5cec8296bc7737e57ebc6964d576"

	// 区块号（如果你知道的话）
	// 如果不知道区块号，我们可以先查询交易获取区块号
	blockNumber := big.NewInt(9983874) // 用你的区块号替换这里

	// === 1. 连接网络 ===
	client, err := ethclient.Dial("https://sepolia.infura.io/v3/" + infuraKey)
	if err != nil {
		log.Fatal("连接失败:", err)
	}
	defer client.Close()

	fmt.Printf("🔍 查找交易: %s\n", targetTxHash)

	// === 2. 先获取交易详情，确认区块号 ===
	txHash := common.HexToHash(targetTxHash)

	// 方法A: 如果不知道区块号，先查询交易获取区块号
	if blockNumber.Int64() == 0 {
		fmt.Println("正在查询交易获取区块号...")
		receipt, err := client.TransactionReceipt(context.Background(), txHash)
		if err != nil {
			log.Fatal("查询交易失败，可能交易不存在或未确认:", err)
		}
		blockNumber = receipt.BlockNumber
		fmt.Printf("✅ 找到交易，所在区块: %d\n", blockNumber)
	}

	fmt.Printf("在区块 #%d 中查找交易...\n", blockNumber)

	// === 3. 获取区块信息 ===
	block, err := client.BlockByNumber(context.Background(), blockNumber)
	if err != nil {
		log.Fatal("获取区块失败:", err)
	}

	fmt.Printf("✅ 找到区块，包含 %d 笔交易\n", len(block.Transactions()))

	// === 4. 在区块中查找交易位置 ===
	found := false
	position := 0

	for i, tx := range block.Transactions() {
		if tx.Hash() == txHash {
			found = true
			position = i + 1 // 位置从1开始计数

			// === 5. 显示交易详细信息 ===
			fmt.Println("=======================================================")
			fmt.Println("🎯 找到目标交易！")
			fmt.Println("=======================================================")

			fmt.Printf("📍 在区块中的位置: 第 %d 笔交易\n", position)
			fmt.Printf("📄 交易哈希: %s\n", tx.Hash().Hex())
			fmt.Printf("💰 转账金额: %s wei\n", tx.Value().String())
			fmt.Printf("   ≈ %.6f ETH\n", weiToEth(tx.Value()))
			fmt.Printf("⛽ Gas Limit: %d\n", tx.Gas())
			fmt.Printf("⛽ Gas Price: %s wei\n", tx.GasPrice().String())
			fmt.Printf("   ≈ %.2f Gwei\n", weiToGwei(tx.GasPrice()))
			fmt.Printf("🔢 Nonce: %d\n", tx.Nonce())

			if tx.To() != nil {
				fmt.Printf("📥 接收地址: %s\n", tx.To().Hex())
			}

			// === 6. 获取交易收据 ===
			receipt, err := client.TransactionReceipt(context.Background(), txHash)
			if err == nil {
				fmt.Println("\n📋 交易收据信息:")
				fmt.Printf("   区块号: %d\n", receipt.BlockNumber)
				fmt.Printf("   区块哈希: %s\n", receipt.BlockHash.Hex())
				fmt.Printf("   状态: ")
				if receipt.Status == 1 {
					fmt.Println("✅ 成功")
				} else {
					fmt.Println("❌ 失败")
				}
				fmt.Printf("   Gas Used: %d\n", receipt.GasUsed)
				fmt.Printf("   累计Gas Used: %d\n", receipt.CumulativeGasUsed)

				// 计算交易费用
				txFee := new(big.Int).Mul(tx.GasPrice(), big.NewInt(int64(receipt.GasUsed)))
				fmt.Printf("   交易费用: %s wei\n", txFee.String())
				fmt.Printf("     ≈ %.6f ETH\n", weiToEth(txFee))
			}

			break
		}
	}

	// === 7. 输出结果 ===
	if !found {
		fmt.Printf("\n❌ 在区块 #%d 中未找到交易 %s\n", blockNumber, targetTxHash)
		fmt.Println("可能原因:")
		fmt.Println("1. 交易哈希错误")
		fmt.Println("2. 区块号错误")
		fmt.Println("3. 交易在另一个区块中")

		// 尝试在整个区块链中查找
		fmt.Println("\n尝试在整个区块链中查找...")
		receipt, err := client.TransactionReceipt(context.Background(), txHash)
		if err != nil {
			fmt.Println("交易不存在或未确认")
		} else {
			fmt.Printf("✅ 交易存在于区块 #%d\n", receipt.BlockNumber)
			fmt.Println("请用正确的区块号重新运行程序")
		}
	} else {
		fmt.Println("=======================================================")
		fmt.Printf("✅ 查询完成！交易 %s\n", targetTxHash)
		fmt.Printf("   在区块 #%d 中的位置: 第 %d 笔交易\n", blockNumber, position)
		fmt.Println("=======================================================")
	}
}

// wei 转 ETH
func weiToEth(wei *big.Int) float64 {
	if wei == nil {
		return 0
	}
	weiFloat := new(big.Float).SetInt(wei)
	ethFloat := new(big.Float).Quo(weiFloat, big.NewFloat(1e18))
	result, _ := ethFloat.Float64()
	return result
}

// wei 转 Gwei
func weiToGwei(wei *big.Int) float64 {
	if wei == nil {
		return 0
	}
	weiFloat := new(big.Float).SetInt(wei)
	gweiFloat := new(big.Float).Quo(weiFloat, big.NewFloat(1e9))
	result, _ := gweiFloat.Float64()
	return result
}
