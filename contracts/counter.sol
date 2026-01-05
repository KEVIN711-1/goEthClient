// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract Counter {
    uint256 private count;
    
    event CountIncreased(address indexed from, uint256 newCount);
    
    constructor() {
        count = 0;
    }
    
    // 增加计数器
    function increment() public {
        count += 1;
        emit CountIncreased(msg.sender, count);
    }
    
    // 获取当前计数
    function getCount() public view returns (uint256) {
        return count;
    }
    
    // 重置计数器
    function reset() public {
        count = 0;
        emit CountIncreased(msg.sender, 0);
    }
}