#!/bin/bash

echo "🚀 启动 Hardhat 开发环境..."

# 切换到项目目录
cd /home/goEth

echo "1. 检查 Hardhat 配置文件..."
if [ ! -f "hardhat.config.js" ]; then
    echo "   创建默认 hardhat.config.js..."
    cat > hardhat.config.js << 'EOF'
/** @type import('hardhat/config').HardhatUserConfig */
module.exports = {
  solidity: "0.8.19",
  networks: {
    hardhat: {
      chainId: 31337,
      mining: {
        auto: true,
        interval: 2000
      }
    }
  }
};
EOF
fi

echo "2. 启动 Hardhat 节点..."
# 在后台启动 Hardhat 节点
npx hardhat node > hardhat.log 2>&1 &
HARDHAT_PID=$!

echo "   等待节点启动..."
sleep 5

echo "3. 检查节点状态..."
if curl -s http://localhost:8545 > /dev/null; then
    echo "   ✅ Hardhat 节点运行在 http://localhost:8545"
else
    echo "   ❌ Hardhat 节点启动失败，查看 hardhat.log"
    cat hardhat.log
    exit 1
fi

echo "4. 运行 Go 测试程序..."
go run main.go

echo "5. 清理..."
kill $HARDHAT_PID 2>/dev/null

echo "🎉 环境测试完成！"