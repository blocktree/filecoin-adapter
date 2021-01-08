/*
 * Copyright 2018 The openwallet Authors
 * This file is part of the openwallet library.
 *
 * The openwallet library is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * The openwallet library is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Lesser General Public License for more details.
 */

package filecoin

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/blocktree/filecoin-adapter/filecoinTransaction"
	"github.com/blocktree/openwallet/v2/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/specs-actors/actors/builtin"
	"github.com/shopspring/decimal"
	"math/big"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/blocktree/openwallet/v2/openwallet"
	"github.com/prometheus/common/log"
)

type TransactionDecoder struct {
	openwallet.TransactionDecoderBase
	openwallet.AddressDecoderV2
	wm *WalletManager //钱包管理者
}

//NewTransactionDecoder 交易单解析器
func NewTransactionDecoder(wm *WalletManager) *TransactionDecoder {
	decoder := TransactionDecoder{}
	decoder.wm = wm
	return &decoder
}

//CreateRawTransaction 创建交易单
func (decoder *TransactionDecoder) CreateRawTransaction(wrapper openwallet.WalletDAI, rawTx *openwallet.RawTransaction) error {
	return decoder.CreateFilRawTransaction(wrapper, rawTx)
}

//SignRawTransaction 签名交易单
func (decoder *TransactionDecoder) SignRawTransaction(wrapper openwallet.WalletDAI, rawTx *openwallet.RawTransaction) error {
	return decoder.SignFilRawTransaction(wrapper, rawTx)
}

//VerifyRawTransaction 验证交易单，验证交易单并返回加入签名后的交易单
func (decoder *TransactionDecoder) VerifyRawTransaction(wrapper openwallet.WalletDAI, rawTx *openwallet.RawTransaction) error {
	return decoder.VerifyFILRawTransaction(wrapper, rawTx)
}

func (decoder *TransactionDecoder) SubmitRawTransaction(wrapper openwallet.WalletDAI, rawTx *openwallet.RawTransaction) (*openwallet.Transaction, error) {
	if len(rawTx.RawHex) == 0 {
		return nil, fmt.Errorf("transaction hex is empty")
	}

	if !rawTx.IsCompleted {
		return nil, fmt.Errorf("transaction is not completed validation")
	}

	from := rawTx.Signatures[rawTx.Account.AccountID][0].Address.Address
	nonce := rawTx.Signatures[rawTx.Account.AccountID][0].Nonce
	nonceUint, _ := strconv.ParseUint(nonce[2:], 16, 64)

	decoder.wm.Log.Info("nonce : ", nonceUint, " update from : ", from)

	message, err := filecoinTransaction.NewMessageFromJSON( rawTx.RawHex )
	if err!=nil {
		return nil, fmt.Errorf("transaction is not wrong message format")
	}

	sig := rawTx.Signatures[rawTx.Account.AccountID][0].Signature

	now1 := decimal.NewFromInt( time.Now().UnixNano() )

	txid, err := decoder.wm.SendRawTransaction(message, sig, decoder.wm.Config.AccessToken) //.SendRawTransaction(message, sig)

	now2 := decimal.NewFromInt( time.Now().UnixNano() )
	cha := now2.Sub( now1).Div( decimal.NewFromInt( 1e9 ) )

	if err != nil {
		decoder.wm.Log.Errorf("send_to_%v_" + strconv.FormatUint(message.Nonce, 10) + ", use : " + cha.String() + "s, now1 :" + now1.String() + ", now2 : " + now2.String(), rawTx.To )

		decoder.wm.UpdateAddressNonce(wrapper, from, 0)
		decoder.wm.Log.Error("Error Tx to send: ", rawTx.RawHex)
		return nil, err
	}

	//交易成功，地址nonce+1并记录到缓存
	newNonce, _ := math.SafeAdd(nonceUint, uint64(1)) //nonce+1
	decoder.wm.UpdateAddressNonce(wrapper, from, newNonce)

	rawTx.TxID = txid
	rawTx.IsSubmit = true

	decimals := int32(6)

	tx := openwallet.Transaction{
		From:       rawTx.TxFrom,
		To:         rawTx.TxTo,
		Amount:     rawTx.TxAmount,
		Coin:       rawTx.Coin,
		TxID:       rawTx.TxID,
		Decimal:    decimals,
		AccountID:  rawTx.Account.AccountID,
		Fees:       rawTx.Fees,
		SubmitTime: time.Now().Unix(),
	}

	tx.WxID = openwallet.GenTransactionWxID(&tx)

	return &tx, nil
}

func (decoder *TransactionDecoder) CreateFilRawTransaction(wrapper openwallet.WalletDAI, rawTx *openwallet.RawTransaction) error {
	var (
		feeInfo         *txFeeInfo
	)

	addresses, err := wrapper.GetAddressList(0, -1, "AccountID", rawTx.Account.AccountID)

	if err != nil {
		return err
	}

	if len(addresses) == 0 {
		return openwallet.Errorf(openwallet.ErrAccountNotAddress, "[%s] have not addresses", rawTx.Account.AccountID)
	}

	var amountStr, to string
	for k, v := range rawTx.To {
		to = k
		amountStr = v
		break
	}

	amountBigInt := common.StringNumToBigIntWithExp(amountStr, decoder.wm.Decimal())

	addressesBalanceList := make([]AddrBalance, 0, len(addresses))

	var feeErr error
	for i, addr := range addresses {
		balance, err := decoder.wm.GetAddrBalance(addr.Address) //decoder.wm.ApiClient.getBalance(addr.Address, decoder.wm.Config.IgnoreReserve, decoder.wm.Config.ReserveAmount)
		if err != nil {
			return err
		}
		nonce_onchain, err := decoder.wm.GetAddrOnChainNonce(addr.Address)
		if err != nil {
			return err
		}
		nonce := decoder.wm.GetAddressNonce(wrapper, addr.Address, nonce_onchain)
		balance.Nonce = nonce

		//计算手续费
		feeInfo, feeErr = decoder.wm.GetTransactionFeeEstimated(addr.Address, to, amountBigInt, nonce)
		if feeErr != nil {
			continue
		}

		//if rawTx.FeeRate != "" {
		//	feeInfo.GasFeeCap = common.StringNumToBigIntWithExp(rawTx.FeeRate, decoder.wm.Decimal())
		//	feeInfo.CalcFee()
		//}

		balance.index = i
		addressesBalanceList = append(addressesBalanceList, *balance)
	}

	sort.Slice(addressesBalanceList, func(i int, j int) bool {
		return addressesBalanceList[i].Balance.Cmp(addressesBalanceList[j].Balance) >= 0
	})

	if feeInfo==nil && feeErr!=nil {
		return feeErr
	}
	fee := feeInfo.Fee.Uint64()
	//if len(rawTx.FeeRate) > 0 {
	//	fee = uint64(decoder.wm.Config.FixedFee) //convertFromAmount(rawTx.FeeRate)
	//} else {
	//	fee = feeInfo.Fee.Uint64()
	//}

	from := ""
	nonce := uint64(0)
	for _, a := range addressesBalanceList {
		from = a.Address
		nonce = a.Nonce
	}

	if from == "" {
		return openwallet.Errorf(openwallet.ErrInsufficientBalanceOfAccount, "the balance: %s is not enough", amountStr)
	}

	nonceMap := map[string]uint64{
		from: nonce,
	}

	rawTx.TxFrom = []string{from}
	rawTx.TxTo = []string{to}
	rawTx.SetExtParam("nonce", nonceMap)
	rawTx.TxAmount = amountStr

	gasFeeCap := common.BigIntToDecimals(feeInfo.GasFeeCap, decoder.wm.Decimal())
	rawTx.FeeRate = gasFeeCap.String()

	totalFeeDecimal := common.BigIntToDecimals(feeInfo.Fee, decoder.wm.Decimal())
	rawTx.Fees = totalFeeDecimal.String()

	decoder.wm.Log.Debugf("nonce: %d", nonce)

	emptyTrans, message, err := decoder.CreateEmptyRawTransactionAndMessage(from, to, amountStr, nonce, decoder.wm.Decimal(), feeInfo) //.CreateEmptyRawTransactionAndMessage(fromPub, hex.EncodeToString(toPub), amount, nonce, fee, mostHeightBlock)
	if err != nil {
		return err
	}
	rawTx.RawHex = emptyTrans

	if rawTx.Signatures == nil {
		rawTx.Signatures = make(map[string][]*openwallet.KeySignature)
	}

	keySigs := make([]*openwallet.KeySignature, 0)

	addr, err := wrapper.GetAddress(from)
	if err != nil {
		return err
	}
	signature := openwallet.KeySignature{
		EccType: decoder.wm.Config.CurveType,
		Nonce:   "0x" + strconv.FormatUint(nonce, 16),
		Address: addr,
		Message: message,
		RSV : true,
	}

	keySigs = append(keySigs, &signature)

	rawTx.Signatures[rawTx.Account.AccountID] = keySigs

	rawTx.FeeRate = big.NewInt(0).SetUint64(fee).String()

	rawTx.IsBuilt = true

	return nil
}

func (decoder *TransactionDecoder) SignFilRawTransaction(wrapper openwallet.WalletDAI, rawTx *openwallet.RawTransaction) error {
	key, err := wrapper.HDKey()
	if err != nil {
		return nil
	}

	keySignatures := rawTx.Signatures[rawTx.Account.AccountID]

	if keySignatures != nil {
		for _, keySignature := range keySignatures {

			childKey, err := key.DerivedKeyWithPath(keySignature.Address.HDPath, keySignature.EccType)
			keyBytes, err := childKey.GetPrivateKeyBytes()
			if err != nil {
				return err
			}

			//签名交易
			///////交易单哈希签名
			signature, err := filecoinTransaction.SignTransaction(keySignature.Message, keyBytes)
			if err != nil {
				return fmt.Errorf("transaction hash sign failed, unexpected error: %v", err)
			}
			keySignature.Signature = hex.EncodeToString(signature)
		}
	}

	rawTx.Signatures[rawTx.Account.AccountID] = keySignatures

	return nil
}

func (decoder *TransactionDecoder) VerifyFILRawTransaction(wrapper openwallet.WalletDAI, rawTx *openwallet.RawTransaction) error {

	var (
		emptyTrans = rawTx.RawHex
		signature  = ""
	)

	for accountID, keySignatures := range rawTx.Signatures {
		log.Debug("accountID Signatures:", accountID)
		for _, keySignature := range keySignatures {

			signature = keySignature.Signature

			log.Debug("Signature:", keySignature.Signature)
			log.Debug("PublicKey:", keySignature.Address.PublicKey)
		}
	}

	_, pass := filecoinTransaction.VerifyAndCombineTransaction(emptyTrans, signature)

	if pass {
		log.Debug("transaction verify passed")
		rawTx.IsCompleted = true
		//rawTx.RawHex = signedTrans
	} else {
		log.Debug("transaction verify failed")
		rawTx.IsCompleted = false
	}

	return nil
}

func (decoder *TransactionDecoder) GetRawTransactionFeeRate() (feeRate string, unit string, err error) {
	pricedecimal := common.BigIntToDecimals(decoder.wm.Config.FixGasPrice, decoder.wm.Decimal())
	return pricedecimal.String(), "Gas", nil
}

//CreateSummaryRawTransaction 创建汇总交易，返回原始交易单数组
func (decoder *TransactionDecoder) CreateSummaryRawTransaction(wrapper openwallet.WalletDAI, sumRawTx *openwallet.SummaryRawTransaction) ([]*openwallet.RawTransaction, error) {
	if sumRawTx.Coin.IsContract {
		return nil, nil
	} else {
		return decoder.CreateSimpleSummaryRawTransaction(wrapper, sumRawTx)
	}
}

func (decoder *TransactionDecoder) CreateSimpleSummaryRawTransaction(wrapper openwallet.WalletDAI, sumRawTx *openwallet.SummaryRawTransaction) ([]*openwallet.RawTransaction, error) {

	var (
		rawTxArray      = make([]*openwallet.RawTransaction, 0)
		accountID       = sumRawTx.Account.AccountID
		minTransfer     = ConvertFromAmount(sumRawTx.MinTransfer, decoder.wm.Decimal())
		retainedBalance = ConvertFromAmount(sumRawTx.RetainedBalance, decoder.wm.Decimal())
	)

	if minTransfer.Cmp(retainedBalance) < 0 {
		return nil, fmt.Errorf("mini transfer amount must be greater than address retained balance")
	}

	//获取wallet
	addresses, err := wrapper.GetAddressList(sumRawTx.AddressStartIndex, sumRawTx.AddressLimit,
		"AccountID", sumRawTx.Account.AccountID)
	if err != nil {
		return nil, err
	}

	if len(addresses) == 0 {
		return nil, fmt.Errorf("[%s] have not addresses", accountID)
	}

	searchAddrs := make([]string, 0)
	for _, address := range addresses {
		searchAddrs = append(searchAddrs, address.Address)
	}

	addrBalanceArray, err := decoder.wm.Blockscanner.GetBalanceByAddress(searchAddrs...)
	if err != nil {
		return nil, err
	}

	//地址余额从大到小排序
	sort.Slice(addrBalanceArray, func(i int, j int) bool {
		a_amount, _ := decimal.NewFromString(addrBalanceArray[i].Balance)
		b_amount, _ := decimal.NewFromString(addrBalanceArray[j].Balance)
		if a_amount.LessThan(b_amount) {
			return true
		} else {
			return false
		}
	})

	for _, addrBalance := range addrBalanceArray {

		//检查余额是否超过最低转账
		addrBalance_BI := ConvertFromAmount(addrBalance.Balance, decoder.wm.Decimal())

		if addrBalance_BI.Cmp(minTransfer) < 0 {
			continue
		}

		//计算汇总数量 = 余额 - 保留余额
		sumAmount_BI := new(big.Int)
		sumAmount_BI.Sub(addrBalance_BI, retainedBalance)

		nonce_db, _ := wrapper.GetAddressExtParam(addrBalance.Address, addrBalance.Symbol + "-nonce")
		if nonce_db != nil {
			nonceStr := common.NewString(nonce_db)
			nonceStrArr := strings.Split(string(nonceStr), "_")
			if len(nonceStrArr)>1 {
				saveTime := common.NewString(nonceStrArr[1]).UInt64()	//上次汇总此地址的时间，秒为单位
				now := uint64(time.Now().Unix())
				diff, _ := math.SafeSub(now, saveTime)

				if diff < decoder.wm.Config.LessSumDiff {	// 少于配置的时间，就不要汇总此地址了
					decoder.wm.Log.Std.Error("%v address nonce diff = %d ", addrBalance.Address, diff)
					continue
				}
			}
		}

		addrOnChainNonce, err := decoder.wm.GetAddrOnChainNonce( addrBalance.Address )
		if err != nil {
			decoder.wm.Log.Std.Error("Failed to get nonce when create summay transaction! %v ", addrBalance.Address)
			continue
		}
		nonce := decoder.wm.GetAddressNonce(wrapper, addrBalance.Address, addrOnChainNonce)

		//计算手续费
		fee, createErr := decoder.wm.GetTransactionFeeEstimated(addrBalance.Address, sumRawTx.SummaryAddress, sumAmount_BI, nonce)
		if createErr != nil {
			decoder.wm.Log.Std.Error("GetTransactionFeeEstimated from[%v] -> to[%v] failed, err=%v", addrBalance.Address, sumRawTx.SummaryAddress, createErr)
			return nil, createErr
		}

		//if sumRawTx.FeeRate != "" {
		//	fee.GasPrice = common.StringNumToBigIntWithExp(sumRawTx.FeeRate, decoder.wm.Decimal()) //ConvertToBigInt(rawTx.FeeRate, 16)
		//	if createErr != nil {
		//		decoder.wm.Log.Std.Error("fee rate passed through error, err=%v", createErr)
		//		return nil, createErr
		//	}
		//	fee.CalcFee()
		//}

		//减去手续费
		sumAmount_BI.Sub(sumAmount_BI, fee.Fee)
		if sumAmount_BI.Cmp(big.NewInt(0)) <= 0 {
			continue
		}

		sumAmount := common.BigIntToDecimals(sumAmount_BI, decoder.wm.Decimal())
		fees := common.BigIntToDecimals(fee.Fee, decoder.wm.Decimal())

		decoder.wm.Log.Info(
			"summary_log_1, address : ", addrBalance.Address,
			" balance : ", addrBalance.Balance,
			" fees : ", fees,
			" sumAmount : ", sumAmount,
			" nonce : ", nonce)

		//创建一笔交易单
		rawTx := &openwallet.RawTransaction{
			Coin:     sumRawTx.Coin,
			Account:  sumRawTx.Account,
			ExtParam: sumRawTx.ExtParam,
			To: map[string]string{
				sumRawTx.SummaryAddress: sumAmount.StringFixed(decoder.wm.Decimal()),
			},
			Required: 1,
			FeeRate:  sumRawTx.FeeRate,
		}

		createErr = decoder.createRawTransaction(
			wrapper,
			rawTx,
			addrBalance,
			fee,
			nonce)
		if createErr != nil {
			decoder.wm.Log.Std.Error("createRawTransaction, err=%v", addrBalance.Address, sumRawTx.SummaryAddress, createErr)
			return nil, createErr
		}

		//创建成功，添加到队列
		rawTxArray = append(rawTxArray, rawTx)
	}
	return rawTxArray, nil
}

func (decoder *TransactionDecoder) createRawTransaction(wrapper openwallet.WalletDAI, rawTx *openwallet.RawTransaction, addrBalance *openwallet.Balance, feeInfo *txFeeInfo, nonce uint64) error {

	var amountStr, to string
	for k, v := range rawTx.To {
		to = k
		amountStr = v
		break
	}

	from := addrBalance.Address
	fromAddr, err := wrapper.GetAddress(from)
	if err != nil {
		return err
	}

	rawTx.TxFrom = []string{from}
	rawTx.TxTo = []string{to}
	rawTx.TxAmount = amountStr

	gasFeeCap := common.BigIntToDecimals(feeInfo.GasFeeCap, decoder.wm.Decimal())
	rawTx.FeeRate = gasFeeCap.String()

	totalFeeDecimal := common.BigIntToDecimals(feeInfo.Fee, decoder.wm.Decimal())
	rawTx.Fees = totalFeeDecimal.String()

	nonceJSON := map[string]interface{}{}
	if len(rawTx.ExtParam) > 0 {
		err = json.Unmarshal([]byte(rawTx.ExtParam), &nonceJSON)
		if err != nil {
			return err
		}
	}
	nonceJSON[from] = nonce

	rawTx.SetExtParam("nonce", nonceJSON)

	emptyTrans, hash, err := decoder.CreateEmptyRawTransactionAndMessage(from, to, amountStr, nonce, decoder.wm.Decimal(), feeInfo) //.CreateEmptyRawTransactionAndMessage(fromAddr.PublicKey, hex.EncodeToString(toPub), amount, nonce, fee, mostHeightBlock)

	if err != nil {
		return err
	}
	rawTx.RawHex = emptyTrans

	if rawTx.Signatures == nil {
		rawTx.Signatures = make(map[string][]*openwallet.KeySignature)
	}

	keySigs := make([]*openwallet.KeySignature, 0)

	signature := openwallet.KeySignature{
		EccType: decoder.wm.Config.CurveType,
		Nonce:   "0x" + strconv.FormatUint(nonce, 16),
		Address: fromAddr,
		Message: hash,
		RSV: true,
	}

	keySigs = append(keySigs, &signature)

	rawTx.Signatures[rawTx.Account.AccountID] = keySigs

	rawTx.FeeRate = feeInfo.Fee.String()

	rawTx.IsBuilt = true

	return nil
}

//CreateSummaryRawTransactionWithError 创建汇总交易，返回能原始交易单数组（包含带错误的原始交易单）
func (decoder *TransactionDecoder) CreateSummaryRawTransactionWithError(wrapper openwallet.WalletDAI, sumRawTx *openwallet.SummaryRawTransaction) ([]*openwallet.RawTransactionWithError, error) {
	raTxWithErr := make([]*openwallet.RawTransactionWithError, 0)
	rawTxs, err := decoder.CreateSummaryRawTransaction(wrapper, sumRawTx)
	if err != nil {
		return nil, err
	}
	for _, tx := range rawTxs {
		raTxWithErr = append(raTxWithErr, &openwallet.RawTransactionWithError{
			RawTx: tx,
			Error: nil,
		})
	}
	return raTxWithErr, nil
}

func (decoder *TransactionDecoder) CreateEmptyRawTransactionAndMessage(from, to, realAmountStr string, nonce uint64, decimals int32, feeInfo *txFeeInfo) (string, string, error) {
	fromAddr, _ := address.NewFromString(from )
	toAddr, _ := address.NewFromString(to)

	valueStr := GetBigIntAmountStr(realAmountStr, decimals)
	value, _ := filecoinTransaction.BigFromString( valueStr )

	method := builtin.MethodSend

	msg := filecoinTransaction.Message{
		To:       toAddr,
		From:     fromAddr,
		Value:    value,
		//GasPrice: filecoinTransaction.NewInt( uint64(gasPrice.Int64()) ),
		GasFeeCap: abi.NewTokenAmount( feeInfo.GasFeeCap.Int64() ),
		GasPremium: abi.NewTokenAmount( feeInfo.GasPremium.Int64() ),
		Nonce:    nonce,
		GasLimit: feeInfo.GasLimit.Int64(),
		Method:   method,
	}

	// 创建空交易单和待签消息
	return msg.CreateEmptyTransactionAndMessage()
}
