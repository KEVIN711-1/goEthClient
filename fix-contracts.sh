#!/bin/bash

echo "=== 修复合约目录问题 ==="

# 1. 清理 contracts 目录中的非合约文件
echo "🧹 清理 contracts 目录..."
cd /home/goEth

# 删除 node_modules 和其他非 .sol 文件
find contracts/ -type f ! -name "*.sol" -delete 2>/dev/null || true
rm -rf contracts/node_modules contracts/package.json contracts/package-lock.json

# 2. 验证 contracts 目录内容
echo ""
echo "📁 contracts 目录内容:"
ls -la contracts/ || echo "contracts 目录不存在"

# 3. 确保有合约文件
if [ ! -f "contracts/Counter.sol" ]; then
  echo "📝 创建 Counter.sol..."
  cat > contracts/Counter.sol << 'SOL'
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract Counter {
    uint256 private count;
    
    event CountIncreased(address indexed from, uint256 newCount);
    
    constructor() {
        count = 0;
    }
    
    function increment() public {
        count += 1;
        emit CountIncreased(msg.sender, count);
    }
    
    function getCount() public view returns (uint256) {
        return count;
    }
    
    function reset() public {
        count = 0;
        emit CountIncreased(msg.sender, 0);
    }
}
SOL
fi

# 4. 清理 Hardhat 缓存
echo ""
echo "🗑️  清理缓存..."
rm -rf cache artifacts

# 5. 忽略 Node.js 版本警告并编译
echo ""
echo "🔧 编译合约（忽略 Node.js 版本警告）..."
export IGNORE_NODE_VERSION_CHECK=true

if npx hardhat compile; then
  echo "✅ 编译成功！"
  echo ""
  echo "🎉 问题已解决！"
  echo "下一步：创建部署脚本并部署到 Sepolia"
else
  echo "❌ 编译失败，查看错误信息"
  echo ""
  echo "尝试手动清理："
  echo "1. rm -rf node_modules package-lock.json"
  echo "2. npm install"
  echo "3. rm -rf contracts/node_modules"
  echo "4. IGNORE_NODE_VERSION_CHECK=true npx hardhat compile"
fi
