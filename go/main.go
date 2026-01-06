package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// 注意：需要先运行 abigen 生成 counter.go 文件
// abigen --abi=build/Counter.abi --bin=build/Counter.bin --pkg=main --out=counter.go

func main() {
	fmt.Println("=== 部署和交互 Counter 合约 ===")

	// 1. 连接到 Sepolia 测试网络
	// 使用 Infura 的免费端点
	infuraURL := "https://sepolia.infura.io/v3/c5daace64d64444790a8d4bdd7c027a6"
	client, err := ethclient.Dial(infuraURL)
	if err != nil {
		log.Fatal("连接失败: ", err)
	}
	defer client.Close()

	fmt.Println("✅ 成功连接到 Sepolia 测试网络")

	// 2. 设置私钥（从环境变量或直接设置）
	privateKeyHex := os.Getenv("PRIVATE_KEY")
	if privateKeyHex == "" {
		// 提示用户设置环境变量
		fmt.Println("请设置 PRIVATE_KEY 环境变量")
		fmt.Println("例如：export PRIVATE_KEY=你的私钥（不带0x前缀）")
		return
	}

	// 去除可能的 0x 前缀
	privateKeyHex = strings.TrimPrefix(privateKeyHex, "0x")

	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		log.Fatal("私钥格式错误: ", err)
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("无法获取公钥")
	}

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	fmt.Printf("✅ 使用地址: %s\n", fromAddress.Hex())

	// 3. 获取账户余额
	balance, err := client.BalanceAt(context.Background(), fromAddress, nil)
	if err != nil {
		log.Fatal("获取余额失败: ", err)
	}

	ethBalance := new(big.Float).Quo(new(big.Float).SetInt(balance), big.NewFloat(1e18))
	fmt.Printf("💰 账户余额: %f ETH\n", ethBalance)

	// 检查是否有足够的 ETH 支付 gas
	if balance.Cmp(big.NewInt(1000000000000000)) < 0 { // 0.001 ETH
		fmt.Println("⚠️  余额不足，请从水龙头获取测试币:")
		fmt.Println("   https://sepoliafaucet.com/")
		fmt.Println("   https://faucet.quicknode.com/ethereum/sepolia")
		return
	}

	// 4. 创建交易认证器
	chainID, err := client.NetworkID(context.Background())
	if err != nil {
		log.Fatal("获取链ID失败: ", err)
	}

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		log.Fatal("创建交易认证器失败: ", err)
	}

	// 设置合理的 gas 参数
	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		log.Fatal("获取 gas 价格失败: ", err)
	}

	// 增加 gas 价格确保交易快速确认（可选）
	gasPrice = new(big.Int).Mul(gasPrice, big.NewInt(12))
	gasPrice = new(big.Int).Div(gasPrice, big.NewInt(10))

	auth.GasPrice = gasPrice
	auth.GasLimit = uint64(300000) // 足够的 gas limit

	// 5. 从文件读取 ABI 和字节码
	abiData, err := ioutil.ReadFile("build/Counter.abi")
	if err != nil {
		log.Fatal("读取 ABI 文件失败: ", err)
	}

	binData, err := ioutil.ReadFile("build/Counter.bin")
	if err != nil {
		log.Fatal("读取字节码文件失败: ", err)
	}

	// 6. 部署合约
	fmt.Println("\n🚀 正在部署合约...")

	// 将字节码转换为 hex
	bytecode := common.FromHex(strings.TrimSpace(string(binData)))

	// 部署合约
	address, tx, _, err := bind.DeployContract(auth, abiData, bytecode, client)
	if err != nil {
		log.Fatal("部署合约失败: ", err)
	}

	fmt.Printf("📝 部署交易哈希: %s\n", tx.Hash().Hex())
	fmt.Printf("🏗️  合约地址: %s\n", address.Hex())

	// 等待交易确认
	fmt.Println("⏳ 等待交易确认...")
	receipt, err := bind.WaitMined(context.Background(), client, tx)
	if err != nil {
		log.Fatal("等待交易确认失败: ", err)
	}

	if receipt.Status == 1 {
		fmt.Println("✅ 合约部署成功!")

		// 保存合约地址到文件
		err = ioutil.WriteFile("contract_address.txt", []byte(address.Hex()), 0644)
		if err != nil {
			fmt.Printf("⚠️  无法保存合约地址到文件: %v\n", err)
		} else {
			fmt.Println("📄 合约地址已保存到 contract_address.txt")
		}
	} else {
		fmt.Println("❌ 合约部署失败")
		return
	}

	// 7. 与合约交互

	// 创建合约实例
	fmt.Println("\n🔗 创建合约实例...")
	contract, err := NewCounter(address, client)
	if err != nil {
		log.Fatal("创建合约实例失败: ", err)
	}

	// 获取初始计数器值
	fmt.Println("📊 读取当前计数器值...")
	count, err := contract.GetCount(&bind.CallOpts{
		From: fromAddress,
	})
	if err != nil {
		log.Fatal("读取计数器失败: ", err)
	}
	fmt.Printf("📈 当前计数: %d\n", count)

	// 增加计数器
	fmt.Println("\n➕ 增加计数器...")

	// 更新 nonce 和 gas 价格
	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		log.Fatal("获取 nonce 失败: ", err)
	}
	auth.Nonce = big.NewInt(int64(nonce))

	// 再次获取最新的 gas 价格
	gasPrice, err = client.SuggestGasPrice(context.Background())
	if err != nil {
		log.Fatal("获取 gas 价格失败: ", err)
	}
	auth.GasPrice = gasPrice

	// 调用 increment 方法
	tx, err = contract.Increment(auth)
	if err != nil {
		log.Fatal("调用 increment 失败: ", err)
	}

	fmt.Printf("📝 交易哈希: %s\n", tx.Hash().Hex())

	// 等待交易确认
	fmt.Println("⏳ 等待交易确认...")
	receipt, err = bind.WaitMined(context.Background(), client, tx)
	if err != nil {
		log.Fatal("等待交易确认失败: ", err)
	}

	if receipt.Status == 1 {
		fmt.Println("✅ increment 调用成功!")

		// 再次读取计数器值
		newCount, err := contract.GetCount(&bind.CallOpts{
			From: fromAddress,
		})
		if err != nil {
			log.Fatal("读取计数器失败: ", err)
		}
		fmt.Printf("📈 新的计数: %d\n", newCount)

		// 检查事件日志
		logs, err := client.FilterLogs(context.Background(), bind.FilterOpts{
			Start:   receipt.BlockNumber.Uint64(),
			End:     &receipt.BlockNumber,
			Context: context.Background(),
		})
		if err != nil {
			fmt.Printf("⚠️  无法获取日志: %v\n", err)
		} else {
			for _, vLog := range logs {
				fmt.Printf("📋 事件日志: %v\n", vLog)
			}
		}
	} else {
		fmt.Println("❌ increment 调用失败")
	}

	fmt.Println("\n🎉 所有操作完成!")
}
