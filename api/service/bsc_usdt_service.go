package service

import (
	"context"
	"errors"
	"math/big"
	"strings"
	"time"

	appTypes "chatplus/core/types"
	logger2 "chatplus/logger"

	"github.com/shopspring/decimal"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

var bscLogger = logger2.GetLogger()

// BscUsdtService 负责在 BSC 链上对 USDT 进行 70/25/5 的自动分账
type BscUsdtService struct {
	cfg    *appTypes.BscConfig
	client *ethclient.Client
}

// NewBscUsdtService 创建 BSC USDT 分账服务
// 如果未启用或关键配置缺失，将返回一个“禁用状态”的服务实例，不会影响主流程
func NewBscUsdtService(appConfig *appTypes.AppConfig) (*BscUsdtService, error) {
	cfg := &appConfig.Bsc

	// 未启用时直接返回一个空 client 的实例，业务层会通过 Enabled() 判断是否真正执行
	if !cfg.Enabled {
		bscLogger.Info("BSC USDT 分账未启用（config.Bsc.enabled = false）")
		return &BscUsdtService{cfg: cfg, client: nil}, nil
	}

	if cfg.RpcURL == "" || cfg.UsdtContract == "" {
		bscLogger.Error("BSC 配置缺失：RpcURL 或 UsdtContract 为空，将禁用链上分账")
		return &BscUsdtService{cfg: cfg, client: nil}, nil
	}

	client, err := ethclient.Dial(cfg.RpcURL)
	if err != nil {
		return nil, err
	}

	return &BscUsdtService{
		cfg:    cfg,
		client: client,
	}, nil
}

// Enabled 当前服务是否真正可用
func (s *BscUsdtService) Enabled() bool {
	return s != nil &&
		s.cfg != nil &&
		s.cfg.Enabled &&
		s.client != nil &&
		s.cfg.FromPrivateKey != "" &&
		s.cfg.UsdtContract != "" &&
		s.cfg.RpcURL != ""
}

// DistributeUSDT 将 totalAmount（单位：USDT）按 70% / 25% / 5% 分别打到三个目标地址
// 该方法假设 USDT 使用 18 位小数（BSC 主网 USDT 为 18 位）
func (s *BscUsdtService) DistributeUSDT(ctx context.Context, totalAmount float64, orderNo string) error {
	if !s.Enabled() {
		// 未启用时静默返回，不影响主业务
		return nil
	}

	if totalAmount <= 0 {
		return errors.New("totalAmount must be greater than 0")
	}

	// 使用 decimal 进行精确计算，避免 float 精度问题
	totalDec := decimal.NewFromFloat(totalAmount).Shift(18) // 放大 1e18，转换为最小单位

	amount70 := totalDec.Mul(decimal.NewFromFloat(0.70)).Truncate(0)
	amount25 := totalDec.Mul(decimal.NewFromFloat(0.25)).Truncate(0)
	// 剩余全部给 5% 地址，避免因为精度损失导致和不为 total
	amount5 := totalDec.Sub(amount70).Sub(amount25)

	toPower := s.cfg.PowerPoolAddress
	toBuyback := s.cfg.BuybackAddress
	toPlatform := s.cfg.PlatformAddress

	if toPower == "" && toBuyback == "" && toPlatform == "" {
		return errors.New("all target addresses are empty")
	}

	privKeyHex := strings.TrimPrefix(s.cfg.FromPrivateKey, "0x")
	privateKey, err := crypto.HexToECDSA(privKeyHex)
	if err != nil {
		return err
	}

	fromAddress := crypto.PubkeyToAddress(privateKey.PublicKey)

	chainID := big.NewInt(56) // 默认 BSC 主网
	if s.cfg.ChainID != 0 {
		chainID = big.NewInt(s.cfg.ChainID)
	}

	// ERC20 transfer 方法 ABI
	erc20ABI, err := abi.JSON(strings.NewReader(`[{"constant":false,"inputs":[{"name":"_to","type":"address"},{"name":"_value","type":"uint256"}],"name":"transfer","outputs":[{"name":"","type":"bool"}],"type":"function"}]`))
	if err != nil {
		return err
	}

	usdtAddr := common.HexToAddress(s.cfg.UsdtContract)

	nonce, err := s.client.PendingNonceAt(ctx, fromAddress)
	if err != nil {
		return err
	}

	gasPrice, err := s.client.SuggestGasPrice(ctx)
	if err != nil {
		return err
	}

	type transferTarget struct {
		address string
		amount  decimal.Decimal
		desc    string
	}

	targets := []transferTarget{
		{address: toPower, amount: amount70, desc: "power_pool_70"},
		{address: toBuyback, amount: amount25, desc: "buyback_25"},
		{address: toPlatform, amount: amount5, desc: "platform_5"},
	}

	for _, t := range targets {
		if t.address == "" {
			continue
		}
		if t.amount.Cmp(decimal.Zero) <= 0 {
			continue
		}

		amountInt := new(big.Int)
		if _, ok := amountInt.SetString(t.amount.String(), 10); !ok {
			bscLogger.Error("无法解析分账金额为 big.Int：", t.amount.String())
			continue
		}

		toAddr := common.HexToAddress(t.address)

		data, err := erc20ABI.Pack("transfer", toAddr, amountInt)
		if err != nil {
			bscLogger.Error("构造 transfer 调用数据失败：", err)
			continue
		}

		// 估算 gasLimit
		msg := ethereum.CallMsg{
			From: fromAddress,
			To:   &usdtAddr,
			Data: data,
		}
		gasLimit, err := s.client.EstimateGas(ctx, msg)
		if err != nil {
			// 如果估算失败，使用一个保守的默认值
			gasLimit = 120000
			bscLogger.Error("估算 gas 失败，使用默认 gasLimit：", err)
		}

		tx := ethtypes.NewTransaction(nonce, usdtAddr, big.NewInt(0), gasLimit, gasPrice, data)

		signedTx, err := ethtypes.SignTx(tx, ethtypes.NewEIP155Signer(chainID), privateKey)
		if err != nil {
			bscLogger.Error("签名交易失败：", err)
			continue
		}

		sendCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		err = s.client.SendTransaction(sendCtx, signedTx)
		cancel()
		if err != nil {
			bscLogger.Error("发送交易失败：", err)
			continue
		}

		bscLogger.Info("BSC USDT 分账交易已发送：",
			"order_no=", orderNo,
			"desc=", t.desc,
			"to=", t.address,
			"amount_wei=", amountInt.String(),
			"tx_hash=", signedTx.Hash().Hex(),
		)

		nonce++
	}

	return nil
}


