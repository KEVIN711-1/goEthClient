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
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

func main() {
	fmt.Println("=== 部署和交互 Counter 合约 ===")

	// ============================== 第一部分：连接区块链网络 ==============================
	// 网络名称: Sepolia Test Network
	// RPC URL: https://sepolia.infura.io/v3/c5daace64d64444790a8d4bdd7c027a6
	// 链ID: 11155111
	// 符号: ETH
	// 区块浏览器: https://sepolia.etherscan.io
	infuraURL := "https://sepolia.infura.io/v3/c5daace64d64444790a8d4bdd7c027a6"

	// ethclient.Dial 创建一个与以太坊节点的连接
	// 类似数据库连接，后续所有操作都需要通过这个 client
	client, err := ethclient.Dial(infuraURL)
	if err != nil {
		log.Fatal("连接失败: ", err)
	}
	// defer 确保程序退出时关闭连接
	// 延迟关闭：确保函数退出前执行 client.Close()
	defer client.Close()

	fmt.Println("✅ 成功连接到 Sepolia 测试网络")

	// ============================== 第二部分：设置账户和私钥 ==============================
	// 2. 设置私钥 - 这是你的"密码"，可以控制账户
	// 私钥是一个64个字符的十六进制字符串（32字节）
	privateKeyHex := os.Getenv("PRIVATE_KEY")
	if privateKeyHex == "" {
		fmt.Println("请设置 PRIVATE_KEY 环境变量")
		fmt.Println("例如：export PRIVATE_KEY=你的私钥（不带0x前缀）")
		return
	}
	// 去掉私钥可能有的 "0x" 前缀
	privateKeyHex = strings.TrimPrefix(privateKeyHex, "0x")

	// 将十六进制字符串转换为 ECDSA 私钥对象，将字符串形式的私钥转换为可用的加密对象
	// ECDSA 是非对称加密算法，用于生成密钥对
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		log.Fatal("私钥格式错误: ", err)
	}

	// 私钥 (Private Key)
	//  ↓ ECDSA 椭圆曲线加密算法
	// 公钥 (Public Key)
	// 	↓ Keccak256 哈希函数
	// 公钥哈希 (Public Key Hash)
	// 	↓ 取最后20个字节
	// 以太坊地址 (Address)

	// 私钥包含生成公钥的所有信息
	// 这是一个单向过程：私钥 → 公钥（不可逆）
	// Public() 方法返回一个通用的 interface{}
	publicKey := privateKey.Public()

	// 类型断言：将 interface{} 转换为具体的 *ecdsa.PublicKey 类型
	// ok 为 true 表示转换成功
	// 这确保了公钥确实是 ECDSA 类型
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("无法获取公钥")
	}

	// 从公钥推导出以太坊地址
	// 地址 = Keccak256(公钥) 的最后20个字节
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	fmt.Printf("✅ 使用地址: %s\n", fromAddress.Hex())

	// ============================== 第三部分：检查账户余额 ==============================
	// 3. 获取账户余额（单位：wei）
	// 1 ETH = 10^18 wei
	// context.Background() 提供请求的上下文
	// nil 表示查询最新区块的余额
	balance, err := client.BalanceAt(context.Background(), fromAddress, nil)
	if err != nil {
		log.Fatal("获取余额失败: ", err)
	}

	// 将 wei 转换为 ETH 显示
	// big.Float 用于处理大浮点数
	ethBalance := new(big.Float).Quo(new(big.Float).SetInt(balance), big.NewFloat(1e18))
	fmt.Printf("💰 账户余额: %f ETH\n", ethBalance)

	// 检查余额是否足够支付 gas 费用
	// 0.001 ETH = 1,000,000,000,000,000 wei
	if balance.Cmp(big.NewInt(1000000000000000)) < 0 { // 0.001 ETH
		fmt.Println("⚠️  余额不足，请从水龙头获取测试币:")
		fmt.Println("   https://sepoliafaucet.com/")
		fmt.Println("   https://faucet.quicknode.com/ethereum/sepolia")
		return
	}
	// ============================== 第四部分：准备交易认证器 ==============================
	// 4. 创建交易认证器（Transactor）
	// 用于签署和发送交易

	// 获取链 ID（Sepolia 是 11155111）
	chainID, err := client.NetworkID(context.Background())
	if err != nil {
		log.Fatal("获取链ID失败: ", err)
	}

	// 创建带链ID的交易认证器
	// 链ID用于防止重放攻击（防止在一条链上签名的交易在另一条链上重放）
	// 现在 auth 对象包含了：
	// - 你的私钥（用于签名）
	// - 链ID（防止重放攻击）
	// - 其他交易参数（稍后设置）
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		log.Fatal("创建交易认证器失败: ", err)
	}

	// 获取账户的 nonce（交易序号）
	// nonce 确保交易按顺序执行，防止双花攻击
	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		log.Fatal("获取 nonce 失败: ", err)
	}
	auth.Nonce = big.NewInt(int64(nonce))

	// 获取建议的 gas 价格
	// gas 价格决定交易被打包的速度，价格越高越快
	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		log.Fatal("获取 gas 价格失败: ", err)
	}

	// 增加 20% gas 价格确保快速确认
	// 在测试网，可以设置稍高一些确保交易快速处理
	gasPrice = new(big.Int).Mul(gasPrice, big.NewInt(12)) // 第一行：gasPrice × 12
	gasPrice = new(big.Int).Div(gasPrice, big.NewInt(10)) // 第二行：÷ 10

	auth.GasPrice = gasPrice
	auth.GasLimit = uint64(300000)
	auth.Value = big.NewInt(0)

	fmt.Printf("⛽ Gas 价格: %d wei\n", gasPrice)
	fmt.Printf("⛽ Gas Limit: %d\n", auth.GasLimit)

	// ============================== 第五部分：读取合约文件 ==============================
	// 5. 从文件读取合约的 ABI 和字节码
	// ABI (Application Binary Interface): 合约的接口定义，告诉Go如何调用合约函数
	// 字节码: 编译后的合约代码，会被部署到区块链上
	abiData, err := ioutil.ReadFile("build/Counter.abi")
	if err != nil {
		log.Fatal("读取 ABI 文件失败: ", err)
	}

	// 读取字节码文件（十六进制字符串）
	binData, err := ioutil.ReadFile("build/Counter.bin")
	if err != nil {
		log.Fatal("读取字节码文件失败: ", err)
	}

	// ============================== 第六部分：部署合约到区块链 ==============================
	// 6. 解析 ABI
	// 将 JSON 格式的 ABI 解析为 Go 可以操作的结构
	parsedABI, err := abi.JSON(strings.NewReader(string(abiData)))
	if err != nil {
		log.Fatal("解析 ABI 失败: ", err)
	}

	// 7. 部署合约
	fmt.Println("\n🚀 正在部署合约...")

	bytecode := common.FromHex(strings.TrimSpace(string(binData)))

	// 部署合约到区块链
	// 参数解释：
	// - auth: 交易认证器（包含私钥、nonce、gas等）
	// - parsedABI: 合约接口定义
	// - bytecode: 合约的字节码
	// - client: 区块链连接
	// 返回值：
	// - address: 部署后合约的地址
	// - tx: 部署交易对象
	// - contract: 合约实例（这里用 _ 忽略，后面会重新创建）
	// - err: 错误信息

	// bytecode 是 EVM（以太坊虚拟机）能直接执行的机器码
	// ABI 告诉 Go：
	// 1. 函数名是什么？
	// 2. 需要什么参数？
	// 3. 返回什么值？
	// 4. 如何编码/解码数据？
	address, tx, _, err := bind.DeployContract(auth, parsedABI, bytecode, client)
	// 问题：这个 contract 实例是基于未确认的交易创建的
	// 合约可能还没在链上生效！
	if err != nil {
		log.Fatal("部署合约失败: ", err)
	}

	fmt.Printf("📝 部署交易哈希: %s\n", tx.Hash().Hex())
	fmt.Printf("🏗️  合约地址: %s\n", address.Hex())

	// 等待交易确认
	fmt.Println("⏳ 等待交易确认...")
	startTime := time.Now()

	// bind.WaitMined 会等待交易被打包进区块
	receipt, err := bind.WaitMined(context.Background(), client, tx)
	if err != nil {
		log.Fatal("等待交易确认失败: ", err)
	}

	elapsedTime := time.Since(startTime)

	// 检查交易状态：receipt.Status == 1 表示成功，0 表示失败
	if receipt.Status == 1 {
		fmt.Printf("✅ 合约部署成功! (耗时: %v)\n", elapsedTime)

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
	// 8. 与合约交互
	fmt.Println("\n🔗 创建合约实例...")
	time.Sleep(2 * time.Second)

	// ============================== 第七部分：与已部署的合约交互 ==============================
	// 创建合约绑定实例
	// 这个对象知道如何调用合约的函数
	// 参数：合约地址、ABI、以及三个client（分别用于不同操作）
	boundContract := bind.NewBoundContract(address, parsedABI, client, client, client)
	// 1. 使用生成的绑定代码（类型安全，推荐）
	// contract, err := NewCounter(address, client)
	// ↑ 这是 abigen 根据你的 Counter.sol 生成的特定类型

	// 获取初始计数器值
	fmt.Println("📊 读取当前计数器值...")

	// 以太坊合约调用方法对比表
	// 方法	用途	是否上链	是否消耗Gas	是否修改状态	返回值	使用场景
	// Call()	调用只读函数（view/pure）	❌ 本地执行	❌ 免费	❌ 不修改	error	查询余额、读取状态、计算数据
	// Transact()	调用会修改状态的函数	✅ 上链执行	✅ 消耗Gas	✅ 修改状态	*Transaction, error	转账、更新状态、执行合约逻辑
	// RawTransact()	发送已编码的调用数据	✅ 上链执行	✅ 消耗Gas	✅ 可能修改	*Transaction, error	离线签名、批量交易、手动编码
	// Transfer()	向合约发送ETH（无数据）	✅ 上链执行	✅ 消耗Gas	✅ 可能修改	*Transaction, error	向合约充值、支付费用
	// FilterLogs()	查询历史事件日志	❌ 本地查询	❌ 免费	❌ 不修改	[]types.Log, error	分析历史事件、数据统计
	// WatchLogs()	实时监听事件	⚡ 实时监听	❌ 免费	❌ 不修改	chan types.Log, Subscription, error	实时通知、监控合约活动

	// 修复：使用 interface{} 类型接收返回值
	var results []interface{} // 用于接收返回值的切片
	err = boundContract.Call(&bind.CallOpts{
		From: fromAddress, // 调用者地址
	}, &results, "getCount") // 函数

	if err != nil {
		log.Fatal("读取计数器失败: ", err)
	}

	// 解析结果
	// results[0] 对应第一个返回值
	if len(results) > 0 {
		if count, ok := results[0].(*big.Int); ok {
			fmt.Printf("📈 当前计数: %d\n", count)
		} else {
			fmt.Printf("⚠️  无法解析计数器值: %v\n", results[0])
		}
	}

	// 9. 增加计数器
	fmt.Println("\n➕ 增加计数器...")

	// 更新 nonce
	nonce, err = client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		log.Fatal("获取 nonce 失败: ", err)
	}
	auth.Nonce = big.NewInt(int64(nonce))

	// 更新 gas 价格
	gasPrice, err = client.SuggestGasPrice(context.Background())
	if err != nil {
		log.Fatal("获取 gas 价格失败: ", err)
	}
	auth.GasPrice = gasPrice

	// 调用 increment 方法（会修改区块链状态）
	// Transact 方法用于调用需要修改状态的函数
	tx, err = boundContract.Transact(auth, "increment")
	if err != nil {
		log.Fatal("调用 increment 失败: ", err)
	}

	fmt.Printf("📝 交易哈希: %s\n", tx.Hash().Hex())

	// 等待交易确认
	fmt.Println("⏳ 等待交易确认...")
	startTime = time.Now()

	receipt, err = bind.WaitMined(context.Background(), client, tx)
	if err != nil {
		log.Fatal("等待交易确认失败: ", err)
	}

	elapsedTime = time.Since(startTime)

	if receipt.Status == 1 {
		fmt.Printf("✅ increment 调用成功! (耗时: %v)\n", elapsedTime)

		// 再次读取计数器值
		time.Sleep(2 * time.Second) // 给链一些时间

		var newResults []interface{}
		err = boundContract.Call(&bind.CallOpts{
			From: fromAddress,
		}, &newResults, "getCount")

		if err != nil {
			log.Fatal("读取计数器失败: ", err)
		}

		if len(newResults) > 0 {
			if newCount, ok := newResults[0].(*big.Int); ok {
				fmt.Printf("📈 新的计数: %d\n", newCount)
			} else {
				fmt.Printf("⚠️  无法解析新计数器值: %v\n", newResults[0])
			}
		}

		// 检查事件日志
		fmt.Println("\n📋 检查事件日志...")

		query := ethereum.FilterQuery{
			FromBlock: receipt.BlockNumber,
			ToBlock:   receipt.BlockNumber,
			Addresses: []common.Address{address},
		}

		logs, err := client.FilterLogs(context.Background(), query)
		if err != nil {
			fmt.Printf("⚠️  无法获取日志: %v\n", err)
		} else if len(logs) > 0 {
			for i, vLog := range logs {
				fmt.Printf("事件 %d - 交易哈希: %s\n", i+1, vLog.TxHash.Hex())

				event, err := parsedABI.EventByID(vLog.Topics[0])
				if err != nil {
					fmt.Printf("  无法解析事件: %v\n", err)
				} else {
					fmt.Printf("  事件名称: %s\n", event.Name)

					// 尝试解析事件数据
					if event.Name == "CountIncremented" {
						var eventData struct {
							NewCount *big.Int
						}
						err = parsedABI.UnpackIntoInterface(&eventData, event.Name, vLog.Data)
						if err != nil {
							fmt.Printf("  无法解析事件数据: %v\n", err)
						} else {
							fmt.Printf("  新的计数值: %d\n", eventData.NewCount)
						}
					}
				}
			}
		} else {
			fmt.Println("  没有找到事件日志")
		}
	} else {
		fmt.Println("❌ increment 调用失败")
	}

	fmt.Println("\n🎉 所有操作完成!")
	fmt.Printf("📄 合约地址: %s\n", address.Hex())
	fmt.Printf("📄 查看合约: https://sepolia.etherscan.io/address/%s\n", address.Hex())
}
