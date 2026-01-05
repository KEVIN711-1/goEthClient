// SPDX-License-Identifier: SEE LICENSE IN LICENSE
pragma solidity ^0.8;

import "@openzeppelin/contracts-upgradeable/proxy/utils/Initializable.sol";
import "@openzeppelin/contracts-upgradeable/proxy/utils/UUPSUpgradeable.sol";
import "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "@openzeppelin/contracts/token/ERC721/IERC721.sol";
import "@openzeppelin/contracts/token/ERC721/IERC721Receiver.sol";
// import "@chainlink/contracts/src/v0.8/interfaces/AggregatorV3Interface.sol";

import {AggregatorV3Interface} from "@chainlink/contracts/src/v0.8/shared/interfaces/AggregatorV3Interface.sol";
// AggregatorV3Interface 这是什么库
// import "hardhat/console.sol";

//contract NftAuction is Initializable, UUPSUpgradeable {
contract NftAuction is Initializable, IERC721Receiver {

    // 结构体
    struct Auction {
        address seller;             // 卖家
        uint256 duration;           // 拍卖持续时间
        uint256 startPrice;         // 起始价格
        uint256 startTime;          // 开始时间
        bool ended;                 // 是否结束
        address highestBidder;      // 最高出价者
        uint256 highestBid;         // 最高价格
        address nftContract;        // NFT合约地址
        uint256 tokenId;            // NFT ID
        address tokenAddress;       // 代币类型，通过地址区分 
        // 参与竞价的资产类型 0x 地址表示eth，其他地址表示erc20 为什么这样定？约定熟成的吗？
        // 0x0000000000000000000000000000000000000000 表示eth
    }

    // 状态变量
    mapping(uint256 => Auction) public auctions;
    // 下一个拍卖ID
    uint256 public nextAuctionId;
    // 管理员地址
    address public admin;

    // AggregatorV3Interface internal priceETHFeed;
    // AggregatorV3Interface 这是什么类型？
    mapping(address => AggregatorV3Interface) public priceFeeds; //价格数据

    function initialize() public initializer {
        admin = msg.sender;
    }

    function setPriceFeed(
        address tokenAddress,
        address _priceFeed 
    ) public {
        //为什么是地址类型，不应该是uint256吗 ？
        // 存的是"问价格的地方"，不是"价格本身"
        // 每次需要价格时，去这个地方问最新的

        // ETH ->  USD 询价地址0x694AA1769357215DE4FAC081bf1f309aDC325306
        // USDC -> USD 询价地址0xA2F78ab2355fe2f984D808B5CeE7FD0A93D5270E 
        priceFeeds[tokenAddress] = AggregatorV3Interface(_priceFeed);
    }

    event PriceDebug(
        uint256 indexed auctionId,
        address tokenAddress,
        uint256 tokenPrice,
        uint256 bidAmount,
        uint256 bidValueUSD,
        uint256 startPriceUSD
    );

    // 此函数的作用是什么?
    // 因为支持多币种拍卖，
    // 各个币种之间汇率不一样不能直接比较数字大小，
    // 需要获取该币种汇率转换后的USD金额比较
    // ETH -> USD => 3011 8508 7500 => 3011.85087500
    // USDC -> USD => 9997 5000 => 0.99975000
    
    function getChainlinkDataFeedLatestAnswer(
        address tokenAddress
    ) public view returns (int) {
        AggregatorV3Interface priceFeed = priceFeeds[tokenAddress];
        //     你：喂，现在ETH什么价？
        //     预言机：好的，我看看...
        //   - 这是第252次报价（roundId: 252）
        //   - 价格是：2000美元（answer: 200000000000）
        //   - 报价开始时间：xxxx
        //   - 最新更新时间：刚刚
        //   - 回答轮次：252
        // prettier-ignore
        (
            /* uint80 roundId */,
            int256 answer,
            /*uint256 startedAt*/,
            /*uint256 updatedAt*/,
            /*uint80 answeredInRound*/
        ) = priceFeed.latestRoundData();

        // 获取该币种最新的美元汇率
        return answer;
    }
    
    // 创建拍卖
    function createAuction(
        uint256 _duration, //时间
        uint256 _startPrice, // 起拍价 默认美元
        address _nftAddress, // 拍品地址
        uint256 _tokenId //拍品token id
    ) public {
        // 只有管理员可以创建拍卖
        require(msg.sender == admin, "Only admin can create auctions");
        // 检查参数
        require(_duration >= 10, "Duration must be greater than 10s");
        require(_startPrice > 0, "Start price must be greater than 0");

        // 转移NFT到合约
        //IERC721(_nftAddress).approve(address(this), _tokenId);
        IERC721(_nftAddress).safeTransferFrom(msg.sender, address(this), _tokenId);
        //创建者将拍平 转移到当前的 拍卖市场保管

        //拍卖市场根据拍品创建一个拍买任务
        auctions[nextAuctionId] = Auction({
            seller: msg.sender,
            duration: _duration,
            startPrice: _startPrice,
            ended: false,
            highestBidder: address(0),
            highestBid: 0,
            startTime: block.timestamp,
            nftContract: _nftAddress,
            tokenId: _tokenId,
            tokenAddress: address(0)
        });

        nextAuctionId++;
    }

    // 买家参与买单
    // TODO: ERC20 也能参加
    function placeBid(
        uint256 _auctionID,     // 参与竞拍的任务id
        address _tokenAddress   // 想要拍买的NFT产品？
    ) external payable {
        uint256 amount = msg.value; // 出价
        uint256 tokenPrice;
        // 统一的价值尺度 美金
        // ETH 是 ？ 美金
        // 1个 USDC 是 ？ 美金

        Auction storage auction = auctions[_auctionID];
        // 判断当前拍卖是否结束
        require(
            !auction.ended &&
                auction.startTime + auction.duration > block.timestamp,
                 // （startTime+durationTime）拍买时间是否大于当前函数调用的时间点
            "Auction has ended"
        );
        // 判断出价是否大于当前最高出价


        uint payValueUsd;
        if (_tokenAddress != address(0)) {
            // 处理 ERC20
            // 检查是否是 ERC20 资产
            // 将用户的代币乘以 与美元的汇率
            // 获取汇率转换后的价格payValue， 在后续竞拍中统一单位比较

            // tokenPrice 有8位小数点 3011 8508 7500
            tokenPrice = uint256(getChainlinkDataFeedLatestAnswer(_tokenAddress))/10**8;
        } else {
            // 处理 ETH
            // 以太坊中约定俗成用零地址0x000...000代表原生ETH
            tokenPrice = uint256(getChainlinkDataFeedLatestAnswer(address(0)))/10**8;
        }
        // WEI -> ETH /10**8
        // ETH -> USD = eth * (tokenPrice/10*8) 
        payValueUsd = (amount * tokenPrice) / 10**18;

            // 4. 记录汇率信息
        emit PriceDebug(
            _auctionID,
            _tokenAddress,
            tokenPrice,
            amount,
            payValueUsd,
            auction.startPrice
        );
        // 此处发现错误，起拍价应该默认统一美元才行，此处又进行了一次汇率换算
        // uint startPriceValue = auction.startPrice *
        //     uint(getChainlinkDataFeedLatestAnswer(auction.tokenAddress));

        // uint highestBidValue = auction.highestBid *
        //     uint(getChainlinkDataFeedLatestAnswer(auction.tokenAddress));
        require(
            payValueUsd >= auction.startPrice && payValueUsd > auction.highestBid,
            "Bid must be higher than the current highest bid"
        );

        // 转移 ERC20 到合约
        if (_tokenAddress != address(0)) {
            // 出价合理，则将代币转移到当前拍买合约中保管
            IERC20(_tokenAddress).transferFrom(msg.sender, address(this), amount);
        }

        // 退还前最高价
        if (auction.highestBid > 0) {
            if (auction.tokenAddress == address(0)) {
                // 以太坊中约定俗成用零地址0x000...000代表原生ETH
                // auction.tokenAddress = _tokenAddress;
                // 给最高出价者退回ETH
                // payable(auction.highestBidder).transfer(auction.highestBid);

                // 将美金转换位wei 退款
                // 1美元 = 10^18 / 3011.85087500
                // 10**18/(tokenPrice/10**8）
                // highestWei = highestBid*10**18/(tokenPrice/10**8);
                // 应该先乘后除，避免精度丢失
                uint256 highestWei = (auction.highestBid * 10**18) / tokenPrice;
                (bool success, ) = payable(auction.highestBidder).call{value: highestWei}("");
                require(success, "transfer failed");

            } else {
                // 退回之前的ERC20
                // 根据代币类型，向highestBidder 退回highestBid 出价
                IERC20(auction.tokenAddress).transfer(
                    auction.highestBidder,
                    auction.highestBid
                );
                // (bool success, ) = IERC20(auction.tokenAddress).call{value: auction.highestBid}("this is the transfer");
                // require(success, "transfer failed");
            }
        }
        
        // 更新当前拍买任务进度
        // 更新最高出价者、出价金额以及货币类型
        auction.tokenAddress = _tokenAddress;
        auction.highestBid = payValueUsd;
        auction.highestBidder = msg.sender;
    }

    // 结束拍卖
    function endAuction(uint256 _auctionID) external { // external 的意思？
        Auction storage auction = auctions[_auctionID];

        // 开发调试工具：只在Hardhat测试环境中有效
        // 与事件的区别：
        // console.log：开发时调试，不上主网
        // event：链上日志，永久存储，前端可监听
        // console.log(
        //     "endAuction",
        //     auction.startTime,
        //     auction.duration,
        //     block.timestamp
        // );
        // 判断当前拍卖是否结束，拍买时间到了才可以结束拍买，管理由都不能提前结束吗
        require(
            !auction.ended &&
                (auction.startTime + auction.duration) <= block.timestamp,
            "Auction has not ended"
        );
        // 转移NFT到最高出价者
        IERC721(auction.nftContract).safeTransferFrom(
            address(this),
            auction.highestBidder,
            auction.tokenId
        );
        // 转移剩余的资金到卖家
        // payable(address(this)).transfer(address(this).balance);
        auction.ended = true;
    }

    // function _authorizeUpgrade(address) internal view override {
    //     // 只有管理员可以升级合约
    //     require(msg.sender == admin, "Only admin can upgrade");
    // }

    // 删除参数名
    // ✅ 优点：最简洁，明确表示"我不需要这些参数"
    // ✅ 符合标准：函数签名完全匹配接口要求
    // ✅ 节省Gas：不执行任何额外操作
    function onERC721Received(
        address,
        address,
        uint256 ,
        bytes calldata
    ) external pure override returns (bytes4) {
        return this.onERC721Received.selector;
    }
}