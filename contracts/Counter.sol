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
