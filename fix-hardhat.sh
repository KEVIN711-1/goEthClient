#!/bin/bash

echo "=== 修复 Hardhat ESM 问题 ==="

# 1. 检查当前目录
echo "当前目录: $(pwd)"
echo ""

# 2. 检查 package.json
if [ -f "package.json" ]; then
  echo "📄 package.json 内容:"
  cat package.json | grep -A5 -B5 '"type"'
  echo ""
  
  # 移除 type 字段或设置为 commonjs
  if grep -q '"type": "module"' package.json; then
    echo "⚠️  发现 'type': 'module'，修改为 'commonjs'..."
    sed -i 's/"type": "module"/"type": "commonjs"/' package.json
  fi
fi

# 3. 确保配置文件是 .js 不是 .cjs
if [ -f "hardhat.config.cjs" ]; then
  echo "📁 重命名 hardhat.config.cjs -> hardhat.config.js"
  mv hardhat.config.cjs hardhat.config.js
fi

# 4. 创建正确的配置文件
echo "⚙️  创建正确的 hardhat.config.js..."
cat > hardhat.config.js << 'CONFIG'
require("@nomicfoundation/hardhat-toolbox");
require("dotenv").config();

module.exports = {
  solidity: "0.8.20",
  paths: {
    sources: "./contracts",
    tests: "./test",
    cache: "./cache",
    artifacts: "./artifacts",
  },
  networks: {
    sepolia: {
      url: process.env.SEPOLIA_RPC_URL || "",
      accounts: process.env.PRIVATE_KEY ? [process.env.PRIVATE_KEY] : [],
      chainId: 11155111,
    },
  },
};
CONFIG

# 5. 验证修复
echo ""
echo "✅ 修复完成！"
echo "运行测试: npx hardhat --version"
npx hardhat --version

echo ""
echo "如果还有问题，尝试:"
echo "1. rm -rf node_modules package-lock.json"
echo "2. npm install"
echo "3. npx hardhat compile"
