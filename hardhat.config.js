// hardhat.config.js
require("@nomicfoundation/hardhat-toolbox");
require("dotenv").config();

module.exports = {
  solidity: "0.8.20",  // 和你合约的版本一致
  
  networks: {
    // 本地开发网络
    hardhat: {
      chainId: 31337,
    },
    
    // Sepolia 测试网
    sepolia: {
      url: process.env.SEPOLIA_RPC_URL,
      accounts: [process.env.PRIVATE_KEY],
      chainId: 11155111,
    },
  },
  
  // 如果你有 Etherscan API Key，可以添加验证
  etherscan: {
    apiKey: process.env.ETHERSCAN_API_KEY,
  },
  
  // 打印 gas 消耗
  gasReporter: {
    enabled: true,
    currency: "USD",
  }
};