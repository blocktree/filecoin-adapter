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
	"encoding/json"
	"errors"
	"fmt"
	"github.com/blocktree/openwallet/v2/common"
	"github.com/blocktree/openwallet/v2/openwallet"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/shopspring/decimal"
	"github.com/tidwall/gjson"
	"math/big"
	"reflect"
	"strconv"
)

const CidLength = 62

const OK_ExitCode = 0

type OwBlock struct {
	Hash          string        `json:"block"`
	PrevBlockHash string        `json:"previousBlock"`
	Timestamp     uint64        `json:"timestamp"`
	Height        uint64        `json:"height"`
	Transactions  []*Transaction `json:"transactions"`

	TipSet       *TipSet
}

type TipSet struct {
	Blks []FilBlockHeader 	`json:"Blocks"`
	BlkCids []string
	Height uint64 			`json:"Height"`
}

type ActiveSync struct {
	Stage uint64 `json:"Stage"`
	Height uint64 `json:"Height"`
	Message string `json:Message`
}

type ActiveSyncList struct {
	ActiveSyncs []ActiveSync `json:ActiveSyncs`
}

//BlockHeader 区块链头
func (b *OwBlock) BlockHeader() *openwallet.BlockHeader {

	obj := openwallet.BlockHeader{}
	//解析json
	obj.Hash = b.Hash
	//obj.Confirmations = b.Confirmations
	obj.Previousblockhash = b.PrevBlockHash
	obj.Height = b.Height
	//obj.Symbol = Symbol

	return &obj
}

func NewBlockHeader(blockHeaderJson *gjson.Result) (FilBlockHeader, error) {
	result := FilBlockHeader{}

	err := json.Unmarshal([]byte(blockHeaderJson.Raw), &result)
	if err != nil {
		return result, err
	}

	result.ParentHashs = make([]string, 0)

	for _, parent := range gjson.Get(blockHeaderJson.Raw, "Parents").Array() {
		hash := gjson.Get(parent.Raw, "/").String()
		result.ParentHashs = append( result.ParentHashs, hash)
	}

	return result, nil
}

func NewBlockTransaction(transactionJson *gjson.Result) (*Transaction, error) {
	result := &Transaction{}

	err := json.Unmarshal([]byte(transactionJson.Raw), &result)
	if err != nil {
		return nil, err
	}

	result.Status = "0"
	return result, nil
}

type Transaction struct {
	Hash             string `storm:"id"`
	BlockNumber      string `storm:"index"`
	BlockHash        string `storm:"index"`
	From             string `json:"From"`
	To               string `json:"To"`
	Gas              string
	GasPrice         string `json:"GasPrice"`
	Value            string `json:"Value"`
	TransactionIndex uint64
	TimeStamp        uint64
	Nonce            uint64 `json:"Nonce"`
	BlockHeight      uint64 //transaction scanning 的时候对其进行赋值
	Method           int64  //方法
	Status           string //链上状态，0：失败，1：成功
}

type FilBlock struct {
	FilBlockHeader
	Transactions []*Transaction
}

type FilBlockHeader struct {
	Miner           string  `json:"Miner"`
	ParentWeight    string  `json:"ParentWeight"`
	Height     		uint64  `json:"Height"`
	Timestamp       uint64  `json:"Timestamp"`
	BlockHeaderCid  string
	ParentHashs     []string
}

type txFeeInfo struct {
	GasLimit *big.Int
	GasPrice *big.Int
	GasPremium *big.Int
	GasFeeCap *big.Int
	Fee      *big.Int
}

func (txFee *txFeeInfo) CalcFee() error {
	fee := new(big.Int)
	fee.Mul(txFee.GasFeeCap, txFee.GasLimit)
	txFee.Fee = fee
	return nil
}

type CallMsg struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Data  string `json:"data"`
	Value string `json:"value"`
}

type CallResult map[string]interface{}

func (r CallResult) MarshalJSON() ([]byte, error) {
	newR := make(map[string]interface{})
	for key, value := range r {
		val := reflect.ValueOf(value) //读取变量的值，可能是指针或值
		if isByteArray(val.Type()) {
			newR[key] = toHex(value)
		} else {
			newR[key] = value
		}
	}
	return json.Marshal(newR)
}

// return : []{"/":"ba..."}...
func GetTipSetKey(hash string) []interface{}{
	blockCids := make([]interface{}, 0)
	for i:=0; i<(len(hash)/CidLength); i++ {
		blockCidHash := hash[i*CidLength:(i+1)*CidLength]

		blockCid := map[string]interface{}{
			"/" : blockCidHash,
		}

		blockCids = append(blockCids, blockCid)
	}
	return blockCids
}

func toHex(key interface{}) string {
	return fmt.Sprintf("0x%x", key)
}

func isByteArray(typ reflect.Type) bool {
	return (typ.Kind() == reflect.Slice || typ.Kind() == reflect.Array) && isByte(typ.Elem())
}

func isByte(typ reflect.Type) bool {
	return typ.Kind() == reflect.Uint8
}

func GetRealAmountStr(bigIntStr string, amountDecimal int32) (string, error){
	amountBigInt, ok := big.NewInt(0).SetString( bigIntStr, 10)
	if !ok {
		return "", errors.New("大转小错误")
	}
	amount_dec := common.BigIntToDecimals( amountBigInt, amountDecimal )
	amount := amount_dec.Abs().String()
	return amount, nil
}

func GetRealAmountDec(bigIntStr string, amountDecimal int32) (decimal.Decimal, error){
	amountBigInt, ok := big.NewInt(0).SetString( bigIntStr, 10)
	if !ok {
		return decimal.Zero, errors.New("decimal大转小错误")
	}
	amount_dec := common.BigIntToDecimals( amountBigInt, amountDecimal )
	return amount_dec, nil
}

//把真实金额的字符串，转化为最小单位的字符串
func GetBigIntAmountStr(amountStr string, amountDecimal int32) string{
	d, _ := decimal.NewFromString(amountStr)
	ten := math.BigPow(10, int64(amountDecimal) )
	w, _ := decimal.NewFromString( ten.String() )
	d = d.Mul(w)
	return d.String()
}

// amount 字符串转为最小单位的表示
func ConvertFromAmount(amountStr string, amountDecimal int32) *big.Int {
	//bigAmountStr := GetBigIntAmountStr(amountStr, amountDecimal)
	r := common.StringNumToBigIntWithExp(amountStr, amountDecimal)
	return r
}

// amount 字符串转为最小单位的表示
func OldConvertFromAmount(amountStr string, amountDecimal int32) uint64 {
	r, _ := strconv.ParseInt(GetBigIntAmountStr(amountStr, amountDecimal), 10, 64)
	return uint64( r )
}

type AddrBalance struct {
	Address string
	RealBalance *decimal.Decimal
	Balance *big.Int
	Nonce   uint64
	index   int
}
