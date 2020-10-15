package filecoinTransaction

import (
	"encoding/hex"
	"fmt"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/specs-actors/actors/builtin"
	"math/big"
	"testing"
)

func Test_transaction(t *testing.T) {
	priv, _ := hex.DecodeString("xxxx-64bit")
	fmt.Println("priv : ", hex.EncodeToString(priv), ", len : ", len(priv))

	//pub, _ := owcrypt.GenPubkey(priv, owcrypt.ECC_CURVE_SECP256K1)
	//fmt.Println("pub : ", hex.EncodeToString(pub), ", len : ", len(pub))
	//
	//dec := filecoin_addrdec.NewAddressDecoderV2(true)
	//from, _ := dec.AddressEncode(pub, false)
	//fmt.Println("from : ", from)

	//传入参数
	from := "t1qqk56urrbt2rut75i72vh6s2xkcwme4c52r4xwi"
	to := "t1wh2fhzvb5rcfoleedkupov442qp4hw34kzm52ki"
	realAmountStr := "5.1235"
	nonce := uint64(2)

	//配置参数
	decimals := int32(18)
	gasLimit := big.NewInt(1000000 )
	gasPrice := big.NewInt( 1 )

	//func开始
	fromAddr, _ := address.NewFromString(from )
	toAddr, _ := address.NewFromString(to)

	valueStr, _ := GetBigIntAmountStr(realAmountStr, decimals)
	value, _ := BigFromString( valueStr )

	method := builtin.MethodSend

	msg := Message{
		To:       toAddr,
		From:     fromAddr,
		Value:    value,
		GasFeeCap: NewInt( uint64(gasPrice.Int64()) ),
		GasPremium: NewInt( uint64(gasPrice.Int64()) ),
		Nonce:    nonce,
		GasLimit: gasLimit.Int64(),
		Method:   method,
	}

	// 创建空交易单和待签消息
	emptyTrans, message, err := msg.CreateEmptyTransactionAndMessage()
	if err != nil {
		t.Error("create failed : ", err)
		return
	}
	fmt.Println("空交易单 ： ", emptyTrans)
	fmt.Println("待签消息 ： ",message)

	signature, err := SignTransaction(message, priv)
	if err != nil {
		t.Error("sign failed")
		return
	}
	fmt.Println("签名结果 ： ", hex.EncodeToString(signature))

	//验签与交易单合并
	signedTrans, pass := VerifyAndCombineTransaction(emptyTrans, hex.EncodeToString(signature))
	if pass {
		fmt.Println("验签成功")
		fmt.Println("签名交易单 ： ", signedTrans)
	} else {
		t.Error("验签失败")
	}
}
