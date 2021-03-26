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
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/blocktree/filecoin-adapter/filecoinTransaction"
	"github.com/blocktree/filecoin-adapter/filecoin_addrdec"
	"github.com/blocktree/go-owcrypt"
	"github.com/blocktree/openwallet/v2/common"
	"github.com/blocktree/openwallet/v2/log"
	"github.com/blocktree/openwallet/v2/openwallet"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/specs-actors/actors/builtin"
	"github.com/tidwall/gjson"
	"math/big"
	"strconv"
	"strings"
	"time"

	//"github.com/blocktree/quorum-adapter/quorum_addrdec"
	"github.com/blocktree/filecoin-adapter/filecoin_rpc"
)

type WalletManager struct {
	openwallet.AssetsAdapterBase

	WalletClient            *filecoin_rpc.Client              // 节点客户端
	Config                  *WalletConfig                   //钱包管理配置
	Blockscanner            *FILBlockScanner         //区块扫描器
	Decoder                 openwallet.AddressDecoderV2     //地址编码器
	TxDecoder               openwallet.TransactionDecoder   //交易单编码器
	ContractDecoder         openwallet.SmartContractDecoder //智能合约解释器
	Log                     *log.OWLogger                   //日志工具
	CustomAddressEncodeFunc func(address string) string     //自定义地址转换算法
	CustomAddressDecodeFunc func(address string) string     //自定义地址转换算法
}

func NewWalletManager() *WalletManager {
	wm := WalletManager{}
	wm.Config = NewConfig()
	wm.Blockscanner = NewFILBlockScanner(&wm)
	wm.Decoder = filecoin_addrdec.NewAddressDecoderV2( wm.Config.isTestNet )
	wm.TxDecoder = NewTransactionDecoder(&wm)
	wm.Log = log.NewOWLogger(wm.Symbol())
	wm.CustomAddressEncodeFunc = CustomAddressEncode
	wm.CustomAddressDecodeFunc = CustomAddressDecode

	return &wm
}

// GetTransactionInBlock
// {"jsonrpc":"2.0","result":{"BlsMessages":[{"Version":0,"To":"t021661","From":"t3vpdi3tg2oppc4bicav723n3e7bzlfaxhvcxjbl72qpe7hra4aggh6oekit3saswmrq5trjap2kdyubpfwxcq","Nonce":68773,"Value":"0","GasPrice":"1","GasLimit":1000000,"Method":6,"Params":"hgMZlyzYKlgmAAFVwh8g78Wdao2sMOq1YR/WjzB33viwbeHwYyfenrp2Qq9pGU4aAAGhw4AaAJpCbw=="},{"Version":0,"To":"t021661","From":"t3vpdi3tg2oppc4bicav723n3e7bzlfaxhvcxjbl72qpe7hra4aggh6oekit3saswmrq5trjap2kdyubpfwxcq","Nonce":68774,"Value":"0","GasPrice":"1","GasLimit":1000000,"Method":6,"Params":"hgMZlzLYKlgmAAFVwh8gzIjaV/osvdivwN4QjAQEd8Y9LwlB/ASCwzm3IVOW3jYaAAGhyIAaAJpCbw=="},{"Version":0,"To":"t0122507","From":"t3ufwpcpo5f4goc2xn3q3rnj32dgrzcud5q7uj3sctxdx2jcr2as7rslztiky7lv6b5vhup75s3cjwhmzwv2va","Nonce":7,"Value":"1000","GasPrice":"1","GasLimit":100000000000,"Method":5,"Params":"hAGBAIGCCFjAorsoEG0M4kuBcV/fOIsqUK42nFBdUXC3biMtKrLjXjo/3FDKZweFSAGswEyYcusEtt1OZM3tO+4DM0OKgmp/qsvgoSVEnbzRxOwdcuAgOiTKVCVMIANb9oVjTh+yMFhGFW1Sy2+u2chgCTMkDQXJcAJ1hQbxyPh6l6wFZPQdZkfJxlk/MYb+3DThTlRFdIvfoYF4T7afAg1ti/APHpi0ii8+QwZnVl0/abBUnw0BRUqxfB657tN9H6eoAJhvvvqdQQA="},{"Version":0,"To":"t0121681","From":"t3q7sm6stkn3ynqkiinxnprijynpwwpbigdq2umzk7w54eii4a6jlzb7yz23u5l2lg4xm43w5zopn74n5aybra","Nonce":33076,"Value":"0","GasPrice":"1","GasLimit":1000000,"Method":6,"Params":"hgMaAAJUfdgqWCYAAVXCHyCO+B0AptwJsd26rx1OMo2AVXqTsYSnW2mmfNPgX8MAOxoAAZ1KgBoAFPMe"},{"Version":0,"To":"t0121681","From":"t3q7sm6stkn3ynqkiinxnprijynpwwpbigdq2umzk7w54eii4a6jlzb7yz23u5l2lg4xm43w5zopn74n5aybra","Nonce":33077,"Value":"0","GasPrice":"1","GasLimit":1000000,"Method":6,"Params":"hgMaAAKIcNgqWCYAAVXCHyCFtFiQ216K21jSpt+ZHGgPoziR3ddeCq0p7iy6GNF7KBoAAZTYgBoAFPMe"},{"Version":0,"To":"t0121681","From":"t3q7sm6stkn3ynqkiinxnprijynpwwpbigdq2umzk7w54eii4a6jlzb7yz23u5l2lg4xm43w5zopn74n5aybra","Nonce":33078,"Value":"0","GasPrice":"1","GasLimit":1000000,"Method":6,"Params":"hgMaAAPUEdgqWCYAAVXCHyA/M7w8EUwOz181SnDm3B3LcevtfWywHGcOnY3UVvVaHxoAAZ05gBoAFPMe"},{"Version":0,"To":"t0121681","From":"t3q7sm6stkn3ynqkiinxnprijynpwwpbigdq2umzk7w54eii4a6jlzb7yz23u5l2lg4xm43w5zopn74n5aybra","Nonce":33079,"Value":"0","GasPrice":"1","GasLimit":1000000,"Method":6,"Params":"hgMaAAWoAdgqWCYAAVXCHyCZ0tQsh8BZvvCW9/8CJbH7sJRVZZ+9rNiQvRrOgkv+OBoAAZzhgBoAFPMe"},{"Version":0,"To":"t0121681","From":"t3q7sm6stkn3ynqkiinxnprijynpwwpbigdq2umzk7w54eii4a6jlzb7yz23u5l2lg4xm43w5zopn74n5aybra","Nonce":33080,"Value":"0","GasPrice":"1","GasLimit":1000000,"Method":6,"Params":"hgMaAAV4B9gqWCYAAVXCHyAZ3aOFbRNBm8AJImJIqYog5qZ18BHXnny9/2VthSFWDhoAAZ0jgBoAFPMe"},{"Version":0,"To":"t0121681","From":"t3q7sm6stkn3ynqkiinxnprijynpwwpbigdq2umzk7w54eii4a6jlzb7yz23u5l2lg4xm43w5zopn74n5aybra","Nonce":33081,"Value":"0","GasPrice":"1","GasLimit":1000000,"Method":6,"Params":"hgMaAAYsB9gqWCYAAVXCHyADT9DB2XFFdRoHcpbcMHeSvSWTTY9Y9TqG+VKqMmBSSRoAAZ1GgBoAFPMe"},{"Version":0,"To":"t0121681","From":"t3q7sm6stkn3ynqkiinxnprijynpwwpbigdq2umzk7w54eii4a6jlzb7yz23u5l2lg4xm43w5zopn74n5aybra","Nonce":33082,"Value":"0","GasPrice":"1","GasLimit":1000000,"Method":6,"Params":"hgMaAAVMB9gqWCYAAVXCHyCw6KUqZOqO8X8sHKKB2QlOjtHuuFIuTsfi2TEq/h4CGBoAAZlXgBoAFPMe"},{"Version":0,"To":"t0121681","From":"t3q7sm6stkn3ynqkiinxnprijynpwwpbigdq2umzk7w54eii4a6jlzb7yz23u5l2lg4xm43w5zopn74n5aybra","Nonce":33083,"Value":"0","GasPrice":"1","GasLimit":1000000,"Method":6,"Params":"hgMaAAXgAtgqWCYAAVXCHyA/5vTQ7aiZ/LWlh0UZxUhDi6RLTS6EJz2bSbD1wxXVRhoAAZz9gBoAFPMe"},{"Version":0,"To":"t0121681","From":"t3q7sm6stkn3ynqkiinxnprijynpwwpbigdq2umzk7w54eii4a6jlzb7yz23u5l2lg4xm43w5zopn74n5aybra","Nonce":33084,"Value":"0","GasPrice":"1","GasLimit":1000000,"Method":6,"Params":"hgMaAAQwHNgqWCYAAVXCHyDRyA65srgBUmzA6Mv9t+OG6B5d1gXMlo9pJ1nRWWZuBxoAAZ4tgBoAFPMe"},{"Version":0,"To":"t0121681","From":"t3q7sm6stkn3ynqkiinxnprijynpwwpbigdq2umzk7w54eii4a6jlzb7yz23u5l2lg4xm43w5zopn74n5aybra","Nonce":33085,"Value":"0","GasPrice":"1","GasLimit":1000000,"Method":6,"Params":"hgMaAAb4BNgqWCYAAVXCHyBkmk85/8Fhm8KoXNBOu5q/5OlIX38umspxhOiqP0fnYBoAAZ0ygBoAFPMe"},{"Version":0,"To":"t04067","From":"t3rbq4ijj5owypedib5txu7fbmeq3x27znjgd3ctpchmuu27gilaqnmqe5vw3v6emtv7tzj3trjbhpux4j4m4a","Nonce":13414,"Value":"113624963406258","GasPrice":"1","GasLimit":1000000,"Method":7,"Params":"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"},{"Version":0,"To":"t0120843","From":"t3w3ycyp42virtnhxrh3l7zt3nvvydyyofyfwvytiqwye2alxgul4oecnenahaer3ydki33dwzndh72dt2hrfa","Nonce":1722,"Value":"0","GasPrice":"1","GasLimit":1000000,"Method":6,"Params":"hgMZBY/YKlgmAAFVwh8gBPNNDQ60nk5YyiGwwdHnaD2XKU1fYHi//Fhq5nG3aSIaAAGhEYAaAJpIaQ=="},{"Version":0,"To":"t0118768","From":"t3qyssqgkgqxa6ims36v3zsfutjhmj5zeljfob7nfsyhkq7ursmkjsdt3bkapbyrhetuobjfzknls2ianmeona","Nonce":122753,"Value":"113624963406258","GasPrice":"1","GasLimit":1000000,"Method":7,"Params":"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"},{"Version":0,"To":"t0121289","From":"t3v7rz7tkyhtscerb3hwyloyh5424yxj4tc2kjc7wg7ied35bxodrlnfbypomg7zetkomvplmpxlcjvz6jqabq","Nonce":9726,"Value":"113624963406258","GasPrice":"1","GasLimit":1000000,"Method":7,"Params":"ghkTTlkHgIkJXBMY2y+2a53wk5YMVBvokfe8ZXauUzLPlq1lB3dJTJcVn0U1eLfGXhUCjg75SIDMMS0AA83BXn5h16LfsJ1l7DNEqC5nym7ZRFuXX5PXuX0qr7iLUVftscZc20g9xBd/XLpjIrzmKgMT6ayV+KJN5sr0V3HYwRuSaKOW0ny6Le9Z/EDG0u8V5hM0RWiqF4i37uqVuo/PEgn5vDY1QUQC7C6tJTkGWL1bFIYcOm/6+BqQ+cabysHspgtGsUoBg6dC0ckHAehUUPMcrXu6vBAWsP1UVs66s/AL6aFvcOwsZHMFn12vC6Lv3+lSQs5cgLSvC4yQNwofznvOF0ul6XZdWJAqb43nvT8rIyDaDJxK29dIPioRjIA13wxsOhsddgNdp+35bZ4RsW6eUEbPCsFuSQ2jy2Gf7kqy+nWmUEgEmMKJLpw/mxXVWbU7nyxT5K632K8ENj61X8e2vUocdeEmRxQLauGMmL1Pv3kkFFf2nzCpBoddDpOHY2daTZRUJLj0KdZBdYjatrliTft4jMwmtp5mKE1LHCxsRLpE00olYk8LzWQqjMS+8UmH/IcHWa4Ct86EI+ZESLpr00yY9H74pgunTGSCyMn9dPQhf5KwdaoT2WdoNhT9VXSr1JWC3QVX5xDsntbFf7VWFxCKyh438YbrrGckzDnC0BIW5uxSI91y2c8zjKKWAiUyiwXWiIDgEyDwfhV0AVzQGHHXRJC+jYe9GiWyqCzjO028rrowvxJ+T5s+kk1vZlmTArz5HKiOXAuWpiFtrk1ee13lAfdz8wyPd1nePWoqh2zx96EPLpLMaciaCAaW+uqjRJoXM6r6FmkvOi5bKJebpO2ClbKBm8hUao+VOcaKRKPNjX8CVZA1HjbtkJ+WZuz03oTKRQcyfllAs4n0APW3bA4y8yY4ySQyVcLJQFbs5Cl2EvabItTnhVzctv3OBo3vfFLTApKQrNrGOycT2zWCdz7bUTog7SzF1K9QgzzEUgK3VFGMbsp6I/3bR/9xE60RPhbcvojOt8HyXYhncqVjPOoLwhsSvkvfD00cYZKQ5FXGtPjFSuX5hf6WGWEVE/xLBPZqu4/RNwVK0a01LNWXFcI1XJxHzZi1mg18cfgZDBYjUEH0okJzf/7d8nbLTB+WSut6JQRSfoPANR1lMUuU8tzqiLPmUNflmUjEd2CwimKtiq9J3dpd7+NM/1s7dUjngI9Oz42G1G4H2/PrYeFJZzHvrF88C4ii4/kjBMpY1JbGA10ypRbU/EY01v1n1Dj1gnsW9qQoKBr1tCYoWcr3Ry9ax0TjTh9WOzGZM9BcTi8dyh0voRdoTpNyWwCsJZB/2iXmu4YmEFKXmQpJNcFkEId2C+xpYHLIMrLhfo2aaj5yTSUXfiprch/uBHGMVfP+eqVcWRQwodsQiO7BZVQRfGqZjbzfjqGykbj5F+QUwb5WAY/vzsNp19hrhfukxjmW1VIMxYuouNGAJ35M+3DR0P+O0tRsvGExTC5bic1OkBCq+Pzi/co/6n/H6a8i8PY/QtfS1ZmUs95YOQNWhfr7QPPlTKpSQxH6CvKn2lYRm3W56SSn6wd8GmD4J0m5tBZmLTZnB5IYkShiSWKAqZsRE8gkSIJHPQTZpRclEYLCktF7E/T1xmcW0JLUXMskGsbRiAuexQORWce1grbeT5Wv2v32R4q2sh9XJvmXpHl3sF9mwSzl0a9h6gMMO8gyBcG2XvoRObba/0Ln85T+60b3l//5gNrAm0fdWoQPcgHNif6wxRjy5xiQtpQK7z4cX2awi4ZU8bBE+UXA5Ppuk71i1zphyLV12mDBzk4tpSOqHvqQ1yFjTdOpWj7w8KxlNwIjJ7aUcqJJvnzgrqXR1oJ/PQi3VrJczc63AKTXeJ22E99XvGR9XRakeBRj8ADq+FJFCrdTCgBUzF2td4KJdg0wHir9gDKgCkH7kNRQj4CoAFrAVNsxD+CwP92WMSNoKGIC4NCkyK8Re6rGx3gOMWNzXvPc7q0iT/lTHbm2QI1yGVGq7nvGrDbczydkXtRCVxdeBw67A4Y+SYDjEtokKhU/u5IRklctkNxOakdBSATKbRiDrYQXIgvEvoDT37icuYQE+Pgbb4hnk04VFUL4s/xDkXdj8rPy6ABr7CpK9LOjJq6MuaFuRnMWQf+ssBoF7rBFM5+FTgoY0TRcEqOU7DPCKoLkRQ2ssatz8ceseuVMbM9WUw/+OU2u39cXvlvjPJM2Oy2cL5FiUEPI5ikH2H/BiqQLb1m5H15tmxPNeapBww8Z7qrntLTVaihiW3m6XKypRk0hLZcQnGCl67OPBYj0nxENtZvg1eOs4u5d+f20F/An/6zx1nFORYFKC6J31MMamSPk9pfqH039pHM/cRjChRiNIHUb9OFbjQEum2EgeeBjmk8S6lQOHYmjs5ikpdzQRPF1VAF7v/PjEAoz9NGQhuAb4gOjSpA1cERa/gd3cJQFU77d8qp/5CFV2HOz0WhBFePZkYQ17mFl6cPwxprHHjL4UrSzBrNAOCMdMpLvFR/I8jZ/pprsM1V3t1hZN8Mduv1vRQ=="},{"Version":0,"To":"t0121289","From":"t3v7rz7tkyhtscerb3hwyloyh5424yxj4tc2kjc7wg7ied35bxodrlnfbypomg7zetkomvplmpxlcjvz6jqabq","Nonce":9727,"Value":"113624963406258","GasPrice":"1","GasLimit":1000000,"Method":7,"Params":"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"}],"SecpkMessages":[{"Message":{"Version":0,"To":"t1wh2fhzvb5rcfoleedkupov442qp4hw34kzm52ki","From":"t1hw4amnow4gsgk2ottjdpdverfwhaznyrslsmoni","Nonce":102027,"Value":"50000000000000000000","GasPrice":"0","GasLimit":10000,"Method":0,"Params":""},"Signature":{"Type":1,"Data":"LVyd5LVrTQrBjd1yWT+5W/hpCCbSgGcAq2+dh4P6ktpPi9dqbSeDodHaj12kljJj3t8UEJNR1PdJDtJLWRy32gA="}},{"Message":{"Version":0,"To":"t1wh2fhzvb5rcfoleedkupov442qp4hw34kzm52ki","From":"t1hw4amnow4gsgk2ottjdpdverfwhaznyrslsmoni","Nonce":102028,"Value":"50000000000000000000","GasPrice":"0","GasLimit":10000,"Method":0,"Params":""},"Signature":{"Type":1,"Data":"TQRG6OyFQWZkVFBXb7pEsEiRCgaMn65rH8DwE4cQNuk+iEdnlHz+0T4VMQc3F36QAkHMQo8MRLKkgfhZPc4IZwE="}}],"Cids":[{"/":"bafy2bzacebehdtyulmuwnli7kehcdzhf7e7gnfxwcdwpelm5q7r63vuelya7a"},{"/":"bafy2bzacedhss6fmspdhupkux454xucetejmi74c7spqm32ddukf6sxiievey"},{"/":"bafy2bzaceah32k363smx3d7ppkkmaho4fonkxhyonpf3wyw4724op652dm4wo"},{"/":"bafy2bzaceafw3xgfjx5phdntsqggwssko3gnt4rwhbftjjzmictodk6qpds6w"},{"/":"bafy2bzacedexamhqmmifkcjfqbfpxo5b3ppjh2b6mdts5o7zvdoygecmdnc2w"},{"/":"bafy2bzaceacuo3phoa45wbpsge7s6f4wxqhfhi2w6koe2pkjmrf2itdj6jl6g"},{"/":"bafy2bzacebomwtsdtlto53hcybq6uqubsxe6q3plqeeualji2dsk3sgy3krfi"},{"/":"bafy2bzacedujiw7264z6iaevm7jmsqkeng4bhajgjdy7mux7zmf47ko2umice"},{"/":"bafy2bzaced7ar6fr266lgpxirdk27kirszxkoeyehvz57xlu5iz4u7gyfgcgq"},{"/":"bafy2bzacea2ryrhfaahz3dtrdccxo5kebhqftx24it3ddssvmuj4n3cgq4oq4"},{"/":"bafy2bzacea37yi33ld7vwpp7q5qactzh6qkcdbrjzi4gdbpvzfxkrgvqkyry4"},{"/":"bafy2bzaceblvbrugqma6gqdb53pwtudd7x65krvecezdb5tndrcsg4s3chudm"},{"/":"bafy2bzaceawpa3nwq6o2xsa6emfs6lb4q6qx2ntxoftbuwu5frxpelgl3qkxu"},{"/":"bafy2bzaceagsy2tap4bgkrwr2kvf2arxqek65wq3g23np4p4npd566xl27xxm"},{"/":"bafy2bzacecmcn4vnkqh3jvlobz6tfmowvkwdxajivyuycuyjr5rjbhgtnhzxc"},{"/":"bafy2bzacedrkd3q6g2jm4zsx3lhocxw3c76ivxv6o7jwnnjbjkus52cr4v67y"},{"/":"bafy2bzacebi6dc7tnfnvniokztzukiydkh2vjtntuxzffzr5h2d2mafc353tm"},{"/":"bafy2bzaceanztjdyehtygknzvsq5hlnebzinvxrdbv3u6tlg7ldhuzhuqbri2"},{"/":"bafy2bzacebxpq7qtous62tixy4qxwdhejvzh2qrblayq43lkitlhbfyo6ugas"},{"/":"bafy2bzacebmgwsmhibuz7hrueg2eiqyxneypuatz4ajhh6pfhc5w5prgrclqk"}]},"id":1}
func (wm *WalletManager) GetTransactionInBlock(blockCid string) ([]*Transaction, error) {
	transactions := make([]*Transaction, 0)

	callMsg := map[string]interface{}{
		"/" : blockCid,
	}

	params := []interface{}{ callMsg}

	result, err := wm.WalletClient.Call("Filecoin.ChainGetBlockMessages", params)
	if err != nil {
		return nil, err
	}

	var transactionIndex = uint64(0)

	for _, blsMessageJSON := range gjson.Get(result.Raw, "BlsMessages").Array() {
		blockTransaction, err := NewBlockTransaction(&blsMessageJSON)
		if err != nil {
			return nil, err
		}
		blockTransaction.TransactionIndex = transactionIndex
		transactionIndex++
		transactions = append(transactions, blockTransaction)
	}

	for _, secpkMessageJSON := range gjson.Get(result.Raw, "SecpkMessages").Array() {
		message := gjson.Get(secpkMessageJSON.Raw, "Message")
		blockTransaction, err := NewBlockTransaction(&message)
		if err != nil {
			return nil, err
		}
		blockTransaction.TransactionIndex = transactionIndex
		transactionIndex++
		transactions = append(transactions, blockTransaction)
	}

	for cidIndex, cidJSON := range gjson.Get(result.Raw, "Cids").Array() {
		transactions[ cidIndex ].Hash = gjson.Get(cidJSON.Raw, "/").String()
	}

	return transactions, nil
}

//// GetTransactionInBlock
//func (wm *WalletManager) GetTransactionAndInfoInBlock(blockCid string) (*FilBlock, error) {
//	callMsg := map[string]interface{}{
//		"/" : blockCid,
//	}
//
//	params := []interface{}{ callMsg}
//
//	result, err := wm.WalletClient.Call("Filecoin.ChainGetBlock", params)
//	if err != nil {
//		return nil, err
//	}
//	blockHeader, err := NewBlockHeader(result)
//	if err != nil {
//		return nil, err
//	}
//	blockHeader.BlockHeaderCid = blockCid
//
//	transactions, err := wm.GetTransactionInBlock( blockCid )
//	if err != nil {
//		return nil, err
//	}
//
//	for transactinIndex, _ := range transactions {
//		transactions[transactinIndex].BlockHash = blockCid
//		transactions[transactinIndex].BlockHeight = blockHeader.Height
//		transactions[transactinIndex].TimeStamp = blockHeader.Timestamp
//
//		str := fmt.Sprintf("%d", blockHeader.Height)
//		transactions[transactinIndex].BlockNumber = str
//	}
//
//	block := FilBlock{
//		FilBlockHeader : blockHeader,
//		Transactions : transactions,
//	}
//
//	return &block, nil
//}

func (wm *WalletManager) GetBlockByHeight(height uint64, getTxs bool) (*OwBlock, error){
	tipSet, err := wm.GetTipSetByHeight(height)
	if err != nil {
		return nil, err
	}
	block, err := GetBlockFromTipSet(tipSet)
	if err != nil {
		return nil, err
	}

	if getTxs && tipSet.Height==height{
		err := wm.SetOwBlockTransactions(&block)
		if err != nil {
			return nil, err
		}
		if len( block.Transactions ) > 0 {
			if block.Transactions[0].Status == "-1" {
				return nil, errors.New("block has no receipt yet")
			}
		}
	}

	return &block, nil
}

// GetBlockByHash
//{"jsonrpc":"2.0","result":{"BlsMessages":[{"Version":0,"To":"t0124395","From":"t3so4ifeweyzvaoy55dnv4g624olsrgomauxeyvzm655piuvi5spkn5fzq3w5dxpnn32p22zsvbbageaj7x2va","Nonce":41,"Value":"0","GasPrice":"1","GasLimit":1000000,"Method":6,"Params":"hgMA2CpYJgABVcIfIE4R/OAhTjHrZ2l8f86gnKFLkmNCdrq0WiJQNrYTrcwzGgAB1ZyAGgCafX8="},{"Version":0,"To":"t0118768","From":"t3qyssqgkgqxa6ims36v3zsfutjhmj5zeljfob7nfsyhkq7ursmkjsdt3bkapbyrhetuobjfzknls2ianmeona","Nonce":166684,"Value":"104451948433383","GasPrice":"1","GasLimit":1000000,"Method":7,"Params":"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"},{"Version":0,"To":"t0118768","From":"t3qyssqgkgqxa6ims36v3zsfutjhmj5zeljfob7nfsyhkq7ursmkjsdt3bkapbyrhetuobjfzknls2ianmeona","Nonce":166685,"Value":"0","GasPrice":"1","GasLimit":1000000,"Method":6,"Params":"hgMaAAHmVtgqWCYAAVXCHyCGAwc/tLEFLBoTa9QmPOY53J2xB6wGJNEgX/ddEAnBNxoAAdfmgBoAmnwB"},{"Version":0,"To":"t0118768","From":"t3qyssqgkgqxa6ims36v3zsfutjhmj5zeljfob7nfsyhkq7ursmkjsdt3bkapbyrhetuobjfzknls2ianmeona","Nonce":166686,"Value":"1000","GasPrice":"1","GasLimit":100000000000,"Method":5,"Params":"hBghgRgggYIIWMCAIFXrvAMKa8nFrKMnYebYpLDlum9iL9bLUHceLkIzGLI8VRjAyd0sdJ2lWOF1EVC3UYLFUe6/HquJ/Dzq8rTz6V+3HzzAG7J23Mk/86oFUAuPmVgp8F16kPCSpepbvuwQkUPPbkjtVWd0ZuIunzx5WBv3gz6FiYpjbPMOvXlaMBa2y/z0Mk7qEZBYMzwDREOk+iBQhsfe7RY89u69PWvflYpCNL4Yuifjwuzklolx4y1QdfJHw7x9ZLp1kfqJ2jpBAA=="},{"Version":0,"To":"t0118768","From":"t3qyssqgkgqxa6ims36v3zsfutjhmj5zeljfob7nfsyhkq7ursmkjsdt3bkapbyrhetuobjfzknls2ianmeona","Nonce":166687,"Value":"104451948433383","GasPrice":"1","GasLimit":1000000,"Method":7,"Params":"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"},{"Version":0,"To":"t09833","From":"t3r6t6gxmstrpxrktzmtx6o6zudnb4hz3rb2u2ms7igjbrljvqghcitvb6zmago3ens2vxwegr2avk5mmexneq","Nonce":32829,"Value":"0","GasPrice":"1","GasLimit":1000000,"Method":6,"Params":"hgMZOnbYKlgmAAFVwh8g3RS4JmalC7TalcEqEBJWVECmQwocV+5nmoSrtG4AZTQaAAHWr4AaAJqASQ=="},{"Version":0,"To":"t0121681","From":"t3q7sm6stkn3ynqkiinxnprijynpwwpbigdq2umzk7w54eii4a6jlzb7yz23u5l2lg4xm43w5zopn74n5aybra","Nonce":106428,"Value":"104452191981305","GasPrice":"1","GasLimit":1000000,"Method":7,"Params":"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"},{"Version":0,"To":"t0121681","From":"t3q7sm6stkn3ynqkiinxnprijynpwwpbigdq2umzk7w54eii4a6jlzb7yz23u5l2lg4xm43w5zopn74n5aybra","Nonce":106429,"Value":"0","GasPrice":"1","GasLimit":1000000,"Method":6,"Params":"hgMaAAaAetgqWCYAAVXCHyDWWkK/X/CUgEz3r/qfYvBdLQNkeFS6X7fZuR4JZmVuZhoAAdLcgBoAFSke"},{"Version":0,"To":"t0121681","From":"t3q7sm6stkn3ynqkiinxnprijynpwwpbigdq2umzk7w54eii4a6jlzb7yz23u5l2lg4xm43w5zopn74n5aybra","Nonce":106430,"Value":"0","GasPrice":"1","GasLimit":1000000,"Method":6,"Params":"hgMaIAB8N9gqWCYAAVXCHyC18m+G2vWuf3zgyHpHHCftH8GozLvDul+lAfgAu6H0FBoAAdBtgBoAFSke"},{"Version":0,"To":"t0121681","From":"t3q7sm6stkn3ynqkiinxnprijynpwwpbigdq2umzk7w54eii4a6jlzb7yz23u5l2lg4xm43w5zopn74n5aybra","Nonce":106431,"Value":"0","GasPrice":"1","GasLimit":1000000,"Method":6,"Params":"hgMaAAXIe9gqWCYAAVXCHyApAkW4p+mhWKC5pf1+fqO8Qp6x3mh8ziYBW9XqON8wTRoAAdLDgBoAFSke"},{"Version":0,"To":"t0121681","From":"t3q7sm6stkn3ynqkiinxnprijynpwwpbigdq2umzk7w54eii4a6jlzb7yz23u5l2lg4xm43w5zopn74n5aybra","Nonce":106432,"Value":"0","GasPrice":"1","GasLimit":1000000,"Method":6,"Params":"hgMaAAY4ddgqWCYAAVXCHyDk+cWlR3q4nHxgF8NKx7dLhSoZWT2cf/QQnWitbXesCxoAAdMggBoAFSke"},{"Version":0,"To":"t0121681","From":"t3q7sm6stkn3ynqkiinxnprijynpwwpbigdq2umzk7w54eii4a6jlzb7yz23u5l2lg4xm43w5zopn74n5aybra","Nonce":106433,"Value":"104452191981305","GasPrice":"1","GasLimit":1000000,"Method":7,"Params":"ghoABaS4WQeAkjPposHIgekfT4r6ANiMhTAjs6O2SpZhF5iDN/5/k8+nJ8PQHIJPBiy/vYoOncOul71PogLPatXAhMMq0Qq3qeWEHVejcOWqNbVHarOzuqAHjH3z6JTeR2f4USdg4jzPFNPWWJxHKgzTtOetHUSd6teTYTkHqZb2HJy4b1rEtEE5Hk8IgLx6W2HhSGSyV42MudilP/k6eaZ3hN+8W5NJna+VGCErYO2uFADmID4zO+/AHrhhyLXKp0enpLkUrpb8rCEjV4waoAo2FLORb679+Nx4aTRqqpAdck3ph6KwiNnrW/JsKMUh/xl2FCKhWf4utomHSbI3fVH3WURHcnk4vlJO9NVtY3zYl54uiy2fifqX1PDlJbCmk+anuz/hPMAVDBC6Oh7lvkwinHh2VIxYNJNdkv1w8cF4JyaIa5tcw6CFyhAfYzOamRor0rto3I/3i62nSSoS2Na6HbyvUW8fTTgxhpAYj2jmNEsfxu8Anyi5asjy/VwsVm5mvDUm5R20lLCZNXIKyNwyw1nMs46ZKuTCXZG0Gz7DsidIKaN2Tbx/XYEhdYRHDLft0dNGmv3ji8KLHY5e6turhs6leo+egeNJ/6X9d1DsgZMGud0oJesJUii6FTsXK+CotMDcuvW2CsRJYTvAiDfiN7Wzi4Xuwrlc7c1KPFjS+63KBQX4dIiPAlFV/T7Re9GJ1SPnFApzp6VP0IrcWN0YgdyAhmkfhzlndTAZGsEleeNQH+OyUM3VmYJmG6IDz7E7/+zb8A6tkBooFapJ2hVUkQHX0bm3+xjSOZPi+9K8sAIB7p7mbBBmYCbU8GzUe/ww1IaQd5S1mEP2tvUYjM6jjUO4W0fsQfiwUxjfZYlEjjcY7IUHveLu6QRT0DXBbQHGDhviohJmCMV9IK3Ejn+5pf0dnLJ6kChFjN/BGG/svdnpedTAyTOGRCs9ZyeS3j7d7LHqLCxegr1HsH4OyXhyjP1mc+sL7C48qSHHocOQQrBMfDURZWewREVuSoPo8VRrBmZLYlBsrNNuzzdw+Un9Jf3EeCSg5x3atqkjI0LJd9PHXb/qFKpSVX0K9+skPwIl2hxWsmFMo7BvIJ2z3GgTKlJKF5XlHtGoyc1rkDbj8tlhp2ikpThY08K/IqJlCItN6nigtpO8CoCRU0GnhOLymFumMJhYfLINjV1nNas2SXGnnJULbDUZ8bWinOrXKr5VsAQqHzuxiVAFiBRgQQluLnMub2dCqgR+A/ia9vp3pwh5orLKcUS7ln5USzFDl9lAZ6Z834jKrRwsakGlUuJU/kZFI3M4D1YPq59jckL+awI46OvEVdLLT3yyNFEe8YQDgpNgcVpKmGKwGem6Jbadj0Q2jEVU3tfPiJIARvFIW+Dr4f/Y7RGqGsz7/kpc7Dgt9jNfD9qdDMhciaK5rRvIEPL8bO3OK1Ll6/2ow3kLBOLo9rjVgu6X4cj9+RCrUOhVecwX9FuSmY9OW8Q17omOHMHQe5WSqiZLCTsSm74ic5DDakWB61DpOC8rEfgwpA/2r2uFOr0ag7vPMZraHyZjjR8U2XX86GyhSopDld6ffmCbd8Fsg+zdHwWn3XzaV3hbcOLC2b+4gDUJad56jYjKOV/rURLcufKmyjrame7poV4CnYBsOzZi03OVZ4YFh2/lOssaoaW4GEGwG7S4tCcCzWp/QU8Xyd/TVGO2KQKDiEugqJ39IG1ajb+10ZAZrUsphXkVWFIAgfIj9zETHcVVLTBau5Si/C61J2fvbCDlCvOkoZVp3x7Pcli2+aIdmgi4OvHkbGvzjXwUjh/1TtfnyQ6k/wXNzrIKZ9Bm7Ez4Onp4vLOxlm9soEC2W/0GVCBMDkb/7hettT6KMM24NJB2hG5/3BkUP354hW+1LWIK4bGVXdhkwpXqRhGnrWBTSnUmNAe0F2xMAJZAFHwwolHbYKvDQE7KYd/aSpQuhCSYkys7UwRHbk5OzE2TfBgxCqZCJ6hw3r1ut/D5WbPHdx3xnSpWdraDDYy6yEk29mkB9QPc7f42220zPZeKOdzFyp2XbPPjK8xfjQARd528s8Zi0ZSm2IKCWsor6O6pCnOlh8lngtzYLyJyZCwVdQEiRliME90KyrX8pJ2nIM3cH4sVE8U6MQjKPMc0EZwY61LYqy0Iki7Zcg8TQjvhGylQqaWhJ5kUYRlSBKvdxNpwDVIQZeU7Y67a2SJeFpCnxgS1KdBVv48dBmWE6qtkRaewrNYPOs8o18LujOtTfxrx6ffFY0cyE9QhZ+Pg0KRqB/JrvVfKMt+YAOmv2n9+pFdYmL8XkdEtMXGxmMuuJQ0jTGvggS668cq5ZHco8UNSlvzziIA7VuqPSGuKf1wgfUdj1rVaxSVOKf8+lB8dFz4sc61eR7xY2IMMD8sRyWuE3lM8A6qX8N3JaIRGWMFkQYQ0EefNV3D0NRETDmmgIbHf3IO7EdXwVLb1i6qlb3ANht+tWNXQ9jFu9F5gEybmUBuw4gpSTR9ifqyhiIsp5nfLNPTzppjNS9fIvwz7oWY/S0cloi7f1fnco/HWIzgGCLgcupggEWtrZKQ7"},{"Version":0,"To":"t0121681","From":"t3q7sm6stkn3ynqkiinxnprijynpwwpbigdq2umzk7w54eii4a6jlzb7yz23u5l2lg4xm43w5zopn74n5aybra","Nonce":106434,"Value":"0","GasPrice":"1","GasLimit":1000000,"Method":6,"Params":"hgMaAAZEdtgqWCYAAVXCHyBh8Fm/5h6ixKDphYrT7ByUbf+z8MDotl3a4utG/soBEhoAAdKQgBoAFSke"},{"Version":0,"To":"t0123492","From":"t3tb6infhjkhzehmtloajxe6l4ndhbquqtzshzu2so35mzcqikyzo7htf6to53cqdtlukvpbsbto2dhmuxhq3q","Nonce":1438,"Value":"104452191981305","GasPrice":"1","GasLimit":1000000,"Method":7,"Params":"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"},{"Version":0,"To":"t0123492","From":"t3tb6infhjkhzehmtloajxe6l4ndhbquqtzshzu2so35mzcqikyzo7htf6to53cqdtlukvpbsbto2dhmuxhq3q","Nonce":1439,"Value":"104451948433383","GasPrice":"1","GasLimit":1000000,"Method":7,"Params":"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"},{"Version":0,"To":"t0123492","From":"t3tb6infhjkhzehmtloajxe6l4ndhbquqtzshzu2so35mzcqikyzo7htf6to53cqdtlukvpbsbto2dhmuxhq3q","Nonce":1440,"Value":"0","GasPrice":"1","GasLimit":1000000,"Method":6,"Params":"hgMZAeLYKlgmAAFVwh8gIfr8uBly5F9N6DP1OWy/5yzS5whgBsoXOhiOGjtIPl0aAAHFbYAaAJqCtQ=="},{"Version":0,"To":"t0122838","From":"t3sbk6d6dsslg6wnm3btqbvyexjuvcgl3juzc3tkmnjxos5ucsm73svvnjkie4vvrfxlqdmnw5mhvubxd6fgdq","Nonce":883,"Value":"104451948433383","GasPrice":"1","GasLimit":1000000,"Method":7,"Params":"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"},{"Version":0,"To":"t0118797","From":"t3u7zjnzvigkpwokge5tu5jkvdtdlwlo2je4uub6uz6guj4vro2jp6ktzh2ksyft36hyjwwocbpca435p2defa","Nonce":35080,"Value":"104451948433383","GasPrice":"1","GasLimit":1000000,"Method":7,"Params":"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"},{"Version":0,"To":"t0118797","From":"t3u7zjnzvigkpwokge5tu5jkvdtdlwlo2je4uub6uz6guj4vro2jp6ktzh2ksyft36hyjwwocbpca435p2defa","Nonce":35081,"Value":"104451948433383","GasPrice":"1","GasLimit":1000000,"Method":7,"Params":"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"},{"Version":0,"To":"t011101","From":"t3qul4w7vki7xqn4ehtyjgn2hc7t3s2vydt5yy4xzbtmw4c4iqqbvz6pbu4vdhswnvbdxnfjtdzbaq6kaped5q","Nonce":142171,"Value":"0","GasPrice":"1","GasLimit":1000000,"Method":6,"Params":"hgMZ5abYKlgmAAFVwh8gE82cbXvoBUBNROszrfN2ivU2l9kDzUi8aX1UdxDpnkEaAAHLFYAaAJp4Ig=="}],"SecpkMessages":[],"Cids":[{"/":"bafy2bzaceay6avlblsqnrlzqsjxp6mwhwmjy4wu4u6bybb2r4ussfviqphrwe"},{"/":"bafy2bzaceaqf5aetbxilhvmsu3klawhsvlhgvkojpxduip7ih6ofe572pyg5q"},{"/":"bafy2bzacecm447fxwsoy24gtvnpggojgda5iqqu5c5p35ixzklitv7onxyues"},{"/":"bafy2bzaceatr52sb77wvaebjo7cpax5pg6t2k3habkc275gcboenra375kmyi"},{"/":"bafy2bzaceclutslq5pjaj4bx72iiptd4mbvzy3yk36kpetott7egnvnvmstkw"},{"/":"bafy2bzacedomieta3wpvanzmacqibncksed3l4ybwvvjo2wq24g2u4bhrcpio"},{"/":"bafy2bzaceag6lxquwrp6aiyuqnjzte3ymqyon6sbetyq2dpmxvopf2tp5bk42"},{"/":"bafy2bzacebm6rmbnf4mwnbw7vl3xgorocgpjqfswyyzjyfxjewxwpe65np4jo"},{"/":"bafy2bzacecg4tirwlyd3o6bvdmfva7mn76mnmu4gacoumzeqk66acclnb5izw"},{"/":"bafy2bzacebtl6df7fwf7unkufrqzcbgxm325ucgnx6sehkovffcg5e5tpl3sm"},{"/":"bafy2bzacebdeaim4hiizmciizynmt26rskcouonvbwdjwyub34f6vxxlgv3sg"},{"/":"bafy2bzaceauqgdzkdmya5usz63c2s3fqxjy2odp6k5enxmqainjaqcmwuvz3i"},{"/":"bafy2bzaceai6fuw47n53rq4ursf6ldqzjl6qtk4o2rl3pn53pl4kl5l3terdy"},{"/":"bafy2bzaced2e4de7vecjkrgo7h6orlu47l44bb5h27mqcmm4qosqosquokwhq"},{"/":"bafy2bzacedcrkqk3wbnfwb6e44xcipdwiot27jlxpjsevowddpcrem6qevfy6"},{"/":"bafy2bzacecu7ugrfmsi4w3qbsgnzu4wte4tdc2e5vfrph3dko4hu57y5n52tc"},{"/":"bafy2bzacecxewfdo6sla6qekczkljkksqp4fth2tulpodytukrk664j37h23s"},{"/":"bafy2bzaced4wpmjy64zkgdtakwiyz2niy7g7nk33fbqxlspqjixctsbf3vzho"},{"/":"bafy2bzacea4hdjhvbtdiw65gxdfxzvprrour4t7hhcesnrdbcj3x7tqesrhs6"},{"/":"bafy2bzacecsregyrce4qtis6o3c7i23iuxoveqanvhgb7mmy3h66r5tlp4cy2"}]},"id":1}
func (wm *WalletManager) GetBlockByHash(hash string) (*OwBlock, error){
	cids := GetTipSetKey( hash)
	params := []interface{}{ cids }

	result, err := wm.WalletClient.Call("Filecoin.ChainGetTipSet", params)
	if err != nil {
		return nil, err
	}

	var tipSet TipSet
	//高度
	tipSet.Height = gjson.Get(result.Raw, "Height").Uint()

	//blocks
	tipSet.Blks = make([]FilBlockHeader, 0)
	tipSet.BlkCids = make([]string, 0)

	for _, blockHeaderJson := range gjson.Get(result.Raw, "Blocks").Array() {
		blockHeader, err := NewBlockHeader(&blockHeaderJson)
		if err != nil {
			return nil, err
		}
		tipSet.Blks = append(tipSet.Blks, blockHeader)
	}

	for cidIndex, cidJSON := range gjson.Get(result.Raw, "Cids").Array() {
		tipSet.Blks[ cidIndex ].BlockHeaderCid = gjson.Get(cidJSON.Raw, "/").String()
		tipSet.BlkCids = append(tipSet.BlkCids, tipSet.Blks[ cidIndex ].BlockHeaderCid)
	}

	block, err := GetBlockFromTipSet(&tipSet)
	if err != nil {
		return nil, err
	}

	return &block, nil
}

// GetTransactionInBlock
// {"jsonrpc":"2.0","result":{"BlsMessages":[{"Version":0,"To":"t09833","From":"t3r6t6gxmstrpxrktzmtx6o6zudnb4hz3rb2u2ms7igjbrljvqghcitvb6zmago3ens2vxwegr2avk5mmexneq","Nonce":32829,"Value":"0","GasPrice":"1","GasLimit":1000000,"Method":6,"Params":"hgMZOnbYKlgmAAFVwh8g3RS4JmalC7TalcEqEBJWVECmQwocV+5nmoSrtG4AZTQaAAHWr4AaAJqASQ=="},{"Version":0,"To":"t0121681","From":"t3q7sm6stkn3ynqkiinxnprijynpwwpbigdq2umzk7w54eii4a6jlzb7yz23u5l2lg4xm43w5zopn74n5aybra","Nonce":106428,"Value":"104452191981305","GasPrice":"1","GasLimit":1000000,"Method":7,"Params":"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"},{"Version":0,"To":"t0121681","From":"t3q7sm6stkn3ynqkiinxnprijynpwwpbigdq2umzk7w54eii4a6jlzb7yz23u5l2lg4xm43w5zopn74n5aybra","Nonce":106429,"Value":"0","GasPrice":"1","GasLimit":1000000,"Method":6,"Params":"hgMaAAaAetgqWCYAAVXCHyDWWkK/X/CUgEz3r/qfYvBdLQNkeFS6X7fZuR4JZmVuZhoAAdLcgBoAFSke"},{"Version":0,"To":"t0121681","From":"t3q7sm6stkn3ynqkiinxnprijynpwwpbigdq2umzk7w54eii4a6jlzb7yz23u5l2lg4xm43w5zopn74n5aybra","Nonce":106430,"Value":"0","GasPrice":"1","GasLimit":1000000,"Method":6,"Params":"hgMaIAB8N9gqWCYAAVXCHyC18m+G2vWuf3zgyHpHHCftH8GozLvDul+lAfgAu6H0FBoAAdBtgBoAFSke"},{"Version":0,"To":"t0121681","From":"t3q7sm6stkn3ynqkiinxnprijynpwwpbigdq2umzk7w54eii4a6jlzb7yz23u5l2lg4xm43w5zopn74n5aybra","Nonce":106431,"Value":"0","GasPrice":"1","GasLimit":1000000,"Method":6,"Params":"hgMaAAXIe9gqWCYAAVXCHyApAkW4p+mhWKC5pf1+fqO8Qp6x3mh8ziYBW9XqON8wTRoAAdLDgBoAFSke"},{"Version":0,"To":"t0121681","From":"t3q7sm6stkn3ynqkiinxnprijynpwwpbigdq2umzk7w54eii4a6jlzb7yz23u5l2lg4xm43w5zopn74n5aybra","Nonce":106432,"Value":"0","GasPrice":"1","GasLimit":1000000,"Method":6,"Params":"hgMaAAY4ddgqWCYAAVXCHyDk+cWlR3q4nHxgF8NKx7dLhSoZWT2cf/QQnWitbXesCxoAAdMggBoAFSke"},{"Version":0,"To":"t0121681","From":"t3q7sm6stkn3ynqkiinxnprijynpwwpbigdq2umzk7w54eii4a6jlzb7yz23u5l2lg4xm43w5zopn74n5aybra","Nonce":106433,"Value":"104452191981305","GasPrice":"1","GasLimit":1000000,"Method":7,"Params":"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"},{"Version":0,"To":"t0121681","From":"t3q7sm6stkn3ynqkiinxnprijynpwwpbigdq2umzk7w54eii4a6jlzb7yz23u5l2lg4xm43w5zopn74n5aybra","Nonce":106434,"Value":"0","GasPrice":"1","GasLimit":1000000,"Method":6,"Params":"hgMaAAZEdtgqWCYAAVXCHyBh8Fm/5h6ixKDphYrT7ByUbf+z8MDotl3a4utG/soBEhoAAdKQgBoAFSke"},{"Version":0,"To":"t0118768","From":"t3qyssqgkgqxa6ims36v3zsfutjhmj5zeljfob7nfsyhkq7ursmkjsdt3bkapbyrhetuobjfzknls2ianmeona","Nonce":166684,"Value":"104451948433383","GasPrice":"1","GasLimit":1000000,"Method":7,"Params":"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"},{"Version":0,"To":"t0118768","From":"t3qyssqgkgqxa6ims36v3zsfutjhmj5zeljfob7nfsyhkq7ursmkjsdt3bkapbyrhetuobjfzknls2ianmeona","Nonce":166685,"Value":"0","GasPrice":"1","GasLimit":1000000,"Method":6,"Params":"hgMaAAHmVtgqWCYAAVXCHyCGAwc/tLEFLBoTa9QmPOY53J2xB6wGJNEgX/ddEAnBNxoAAdfmgBoAmnwB"},{"Version":0,"To":"t0118797","From":"t3u7zjnzvigkpwokge5tu5jkvdtdlwlo2je4uub6uz6guj4vro2jp6ktzh2ksyft36hyjwwocbpca435p2defa","Nonce":35080,"Value":"104451948433383","GasPrice":"1","GasLimit":1000000,"Method":7,"Params":"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"},{"Version":0,"To":"t0123492","From":"t3tb6infhjkhzehmtloajxe6l4ndhbquqtzshzu2so35mzcqikyzo7htf6to53cqdtlukvpbsbto2dhmuxhq3q","Nonce":1438,"Value":"104452191981305","GasPrice":"1","GasLimit":1000000,"Method":7,"Params":"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"},{"Version":0,"To":"t0123492","From":"t3tb6infhjkhzehmtloajxe6l4ndhbquqtzshzu2so35mzcqikyzo7htf6to53cqdtlukvpbsbto2dhmuxhq3q","Nonce":1439,"Value":"104451948433383","GasPrice":"1","GasLimit":1000000,"Method":7,"Params":"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"},{"Version":0,"To":"t0122838","From":"t3sbk6d6dsslg6wnm3btqbvyexjuvcgl3juzc3tkmnjxos5ucsm73svvnjkie4vvrfxlqdmnw5mhvubxd6fgdq","Nonce":883,"Value":"104451948433383","GasPrice":"1","GasLimit":1000000,"Method":7,"Params":"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"}],"SecpkMessages":[],"Cids":[{"/":"bafy2bzacedomieta3wpvanzmacqibncksed3l4ybwvvjo2wq24g2u4bhrcpio"},{"/":"bafy2bzaceag6lxquwrp6aiyuqnjzte3ymqyon6sbetyq2dpmxvopf2tp5bk42"},{"/":"bafy2bzacebm6rmbnf4mwnbw7vl3xgorocgpjqfswyyzjyfxjewxwpe65np4jo"},{"/":"bafy2bzacecg4tirwlyd3o6bvdmfva7mn76mnmu4gacoumzeqk66acclnb5izw"},{"/":"bafy2bzacebtl6df7fwf7unkufrqzcbgxm325ucgnx6sehkovffcg5e5tpl3sm"},{"/":"bafy2bzacebdeaim4hiizmciizynmt26rskcouonvbwdjwyub34f6vxxlgv3sg"},{"/":"bafy2bzaceauqgdzkdmya5usz63c2s3fqxjy2odp6k5enxmqainjaqcmwuvz3i"},{"/":"bafy2bzaceai6fuw47n53rq4ursf6ldqzjl6qtk4o2rl3pn53pl4kl5l3terdy"},{"/":"bafy2bzaceaqf5aetbxilhvmsu3klawhsvlhgvkojpxduip7ih6ofe572pyg5q"},{"/":"bafy2bzacecm447fxwsoy24gtvnpggojgda5iqqu5c5p35ixzklitv7onxyues"},{"/":"bafy2bzaced4wpmjy64zkgdtakwiyz2niy7g7nk33fbqxlspqjixctsbf3vzho"},{"/":"bafy2bzaced2e4de7vecjkrgo7h6orlu47l44bb5h27mqcmm4qosqosquokwhq"},{"/":"bafy2bzacedcrkqk3wbnfwb6e44xcipdwiot27jlxpjsevowddpcrem6qevfy6"},{"/":"bafy2bzacecxewfdo6sla6qekczkljkksqp4fth2tulpodytukrk664j37h23s"}]},"id":1}
func (wm *WalletManager) SetOwBlockTransactions(owBlock *OwBlock) (error){
	allTransaction := make([]*Transaction, 0)

	transactionCidMap := make(map[string]string) //防止重复用的map
	transactionNonceMap := make(map[string]string) //防止重复用的map，key value = from_nonce
	for _, filBlock := range owBlock.TipSet.Blks{
		itemTransactions, err := wm.GetTransactionInBlock( filBlock.BlockHeaderCid )
		if err != nil {
			return err
		}

		hasFindReceipt := false
		for transactinIndex, _ := range itemTransactions {
			transaction := itemTransactions[transactinIndex]

			//暂时只收集transfer方法，method==0
			if transaction.Method != 0 {
				continue
			}

			_, ok := transactionCidMap[ transaction.Hash ]
			if !ok { //找不到的时候补充
				itemTransactions[transactinIndex].BlockHash = owBlock.Hash
				itemTransactions[transactinIndex].BlockHeight = filBlock.Height
				itemTransactions[transactinIndex].TimeStamp = filBlock.Timestamp

				str := fmt.Sprintf("%d", filBlock.Height)
				itemTransactions[transactinIndex].BlockNumber = str

				//判断，如果value大于0，还要判断状态
				amountBigInt, bigIntOk := big.NewInt(0).SetString( transaction.Value, 10)
				if bigIntOk && amountBigInt.Cmp( big.NewInt(0) )==1 {
					if hasFindReceipt==false{
						exitCode, _, err := wm.GetTransactionReceipt( transaction.Hash )
						if err != nil {
							wm.Log.Std.Error("transaction get receipt error, hash : %v, err : %v", itemTransactions[transactinIndex].Hash, err )
							continue
						}
						if exitCode!=OK_ExitCode{
							//continue
							itemTransactions[transactinIndex].Status = "0"
							//wm.Log.Std.Info("transaction, hash : %v, to: %v, status: %v", itemTransactions[transactinIndex].Hash, itemTransactions[transactinIndex].To, itemTransactions[transactinIndex].Status )
						}else{
							itemTransactions[transactinIndex].Status = "1"
						}
						if exitCode==-1{
							itemTransactions[transactinIndex].Status = "-1"
						}
						hasFindReceipt = true
					}

					//itemTransactions[transactinIndex].Gas = strconv.FormatInt( gasUsed, 10)
				}

				itemTransactions[transactinIndex].Gas = "0"
				itemTransactions[transactinIndex].GasPrice = "0"

				nonceKey := transaction.From + "_" + strconv.FormatUint(transaction.Nonce, 10)
				_, nonceFound := transactionNonceMap[ nonceKey ]
				if nonceFound {
					wm.Log.Std.Error("transaction equal transaction, hash : %v", transaction.Hash )
					continue
				}

				allTransaction = append(allTransaction, itemTransactions[transactinIndex])

				//加上在map中，防止重复
				transactionCidMap[ transaction.Hash ] = transaction.Hash
				transactionNonceMap[ nonceKey ] = nonceKey
			}
		}
	}

	owBlock.Transactions = allTransaction
	return nil
}

// GetTipSetByHeight
// {"jsonrpc":"2.0","result":{"Cids":[{"/":"bafy2bzacea24lcnctzolxc24k5utvvn33mgub4a3bfka7436jv32h3fviechg"},{"/":"bafy2bzaceaeqajdqvcfjgroissn5jhwcsk6yyyo5gh5h34igt5kdbmdmpny24"},{"/":"bafy2bzaceceld4xndb7tzrhmhoewlqhqstdjwtpfrenwpao7hzuhhf5s7hlzm"},{"/":"bafy2bzacebatzr6572caj66bjnodkkntvszmwyudujjrgl75vxlbl2hofxctk"},{"/":"bafy2bzaceb7roytvblia5jdl4g6z2jdoqzabqfepuef2speygklzrrba6evy2"}],"Blocks":[{"Miner":"t0118133","Ticket":{"VRFProof":"q35savuXm3c/wTaqXHQAYkGcSvrHoh6ueizEVr1nvNNxj2VyGiMrwB5LIw7NSt6dDynNn5P+7qcNDHlFgl98OPHAPXkpQwwfBYVGqxiJK/O4Caqxf6HbAwNKiLY/Q7HH"},"ElectionProof":{"VRFProof":"jPhclWtAhBk5zFvkj2+B7Of/NuAkgUJTWesyfcyFkbjmCqdcil0faJx2AhzxH7izEWlsgDEMCWPnA1PMzQU9PYI9vqYcybDYJqIPSBAi3yvUiKkVao5bwi9TlOKYfr3E"},"BeaconEntries":[{"Round":205552,"Data":"k0WKvac/9106OZt5ECA6vWMOCiAHOzm7NOmS0tc5r/E3sV2aWKZZzQgfZ+IE7s5HB6pODv6RmOXpXpVqFYyloTG9Vf2s8bb5J179+GGcs37b3gcvYfrLG5JQUu3Z+5B7"}],"WinPoStProof":[{"PoStProof":3,"ProofBytes":"tTi4xOcyU7zCoGHYxTXr5bJ49nG5Fo0DkryEHuub8HQYHHHk+GFrJkj3g3cZSHErjgriNL9dz9Zyg4q8E625lrzVZRbsHFMpYwRfcqtl38LB1E57N7sTAF2i9XnP6RhsDXwpuy432OO0zVtyvx4W77Bpc0FTs71UGBQkjV04H5TEEWH+ZI868SY16EQ0D93euUZhNIl7eOSG0P4FgBDPUV6FOHGQGoVzD2SHzIauw02A+CeUy9+yN8t6twHOddQZ"}],"Parents":[{"/":"bafy2bzacecvqw436kn24hoa6q24opilkfpowz2tepzu2tp55rpxtephcg2iya"}],"ParentWeight":"2095848573","Height":122368,"ParentStateRoot":{"/":"bafy2bzaced35p3xlzah7tveoodnvgi4uh2v3r27r5jzsunqexq7tkmwqtyzsa"},"ParentMessageReceipts":{"/":"bafy2bzacede7lnmwuv6lcdpk3jeuilhljklzvypn7fupne43tb7lllfqcnf62"},"Messages":{"/":"bafy2bzacedqmi3dpzhwfvl2xggrp6o645ntkw7zw7xowho3vr6c6xrdgdcbow"},"BLSAggregate":{"Type":2,"Data":"iW80Y03L+QLfsszSMpP3nNJiN5bmeXrMtwHquUGhkToTYQGlKsJoJ/W7dmS9920AEf1smz01FOwYPHJ+GycYe9O0aKeuytdjiwnoGn4DtEbaVuISduX2nm/T1R0PTtoe"},"Timestamp":1595584000,"BlockSig":{"Type":2,"Data":"tm1mrtyE5ps9hy6ZxBdHcmQq9AasgHdJJLggh71ozSn1M/3Du9IvKMeMWQQzI5wADaHxQg2pFWljdlAtCnuBuiisFfMZQ7Mfkpi1pJsxOXZISOOItFYIMXuGAjWCDJNf"},"ForkSignaling":0},{"Miner":"t0120409","Ticket":{"VRFProof":"iXcq9cY3A9diTQBB6ZYMnFG9Eq61KBmv1AvQHPXinIIxu3CGukgke7yBdB2vlJq+BjDzJ9II2EaOWx1Q1OcgQFb1pZoXiV4dKcH8vgM1FeFxbtg3hsa6d3/tse2nvmJy"},"ElectionProof":{"VRFProof":"kQrVZY8ikMz486HNRGIazbVpfc6o2SuiFpBX99Mm2WGxB5LXvl1aZiI/OKEPCJguB02aiQJTqpwnCFRi9uqjPpsMtFXcnbFUvaUaQsgILXg9LfaPjySrQR38iy0J42t7"},"BeaconEntries":[{"Round":205552,"Data":"k0WKvac/9106OZt5ECA6vWMOCiAHOzm7NOmS0tc5r/E3sV2aWKZZzQgfZ+IE7s5HB6pODv6RmOXpXpVqFYyloTG9Vf2s8bb5J179+GGcs37b3gcvYfrLG5JQUu3Z+5B7"}],"WinPoStProof":[{"PoStProof":3,"ProofBytes":"pf/tMJSPWsV1dD/v2SmCJRBrDa/V1qOhUcylmCnD17NvZipBNmh2Htf0gfI+QHMZjouVX1bkoFnpQsm69c4UpjTEsTsCoKBeBOpqxzKRD11Gf8jMU2Dda/hlJQRoFNgNGZWc0Kd9LiwEQUqjBUrGHXBJa2kOVcPK7ftPydfYLWLJ1NjuGyO/aTcv95Zun/SyrB6oqo8y2krK5nFQNhNy7Tj5MCQoL8c4dyiuiDYk5ISRw8TSmJWpl/tyNVVHmZFo"}],"Parents":[{"/":"bafy2bzacecvqw436kn24hoa6q24opilkfpowz2tepzu2tp55rpxtephcg2iya"}],"ParentWeight":"2095848573","Height":122368,"ParentStateRoot":{"/":"bafy2bzaced35p3xlzah7tveoodnvgi4uh2v3r27r5jzsunqexq7tkmwqtyzsa"},"ParentMessageReceipts":{"/":"bafy2bzacede7lnmwuv6lcdpk3jeuilhljklzvypn7fupne43tb7lllfqcnf62"},"Messages":{"/":"bafy2bzacec6l3ka2oo6s7kmfjhfyvcmijr7hfv5nl76ih3rk6b3zporqtuycs"},"BLSAggregate":{"Type":2,"Data":"tPHFOXE8b7vjxiBFctzmQYjm84Na3D2gzW/vufVUtGG00I9lpyTwofrGW/vUMdDqBO3Xn2WLrHs96pWTkBGRoJ8G/nf+r7ee5NrKEGyd9C8XHe78Achwlbr1DpAnGQzU"},"Timestamp":1595584000,"BlockSig":{"Type":2,"Data":"rdZrel1sKKzxKlfJFshSJNe57fisNRrdZ6k0tQQMo9n+xw2pcGStU9krwC5iFn8zBhL6m1wkvBFhsUhFLZQfBybQ7BERCEfqtw0hFB7bFaILe4U4nAQCwXD431g1f/b1"},"ForkSignaling":0},{"Miner":"t02020","Ticket":{"VRFProof":"rkZUHLhKeH6s8dPstHAq8otpLCmppR21JwSfQOpDpxlCrXkn2JdPYNGPPudu+gqFBowMCn/1qzG1gBr9Rn39Czk/7GCw42ukdbwdte0ACHiwyEmhK+Ft7IzSHvxwIAWX"},"ElectionProof":{"VRFProof":"mFqiIyinJAYlMzYJ/jSNVmM8QWVtHQloNF1bBzGZ4IUPTjYgzpiBvRUp2Ff8r2T5D3OTk2xnu/R7fJxh717VwU7y6hK3jAoVCWE7BXXomCkouF1iysclvgQMq0eRrQgc"},"BeaconEntries":[{"Round":205552,"Data":"k0WKvac/9106OZt5ECA6vWMOCiAHOzm7NOmS0tc5r/E3sV2aWKZZzQgfZ+IE7s5HB6pODv6RmOXpXpVqFYyloTG9Vf2s8bb5J179+GGcs37b3gcvYfrLG5JQUu3Z+5B7"}],"WinPoStProof":[{"PoStProof":3,"ProofBytes":"k8VCPiStbIqQyS5rQam3ONmeYGaZhyqS0esfOL+lj0C3waD0BvI4mdiztBSahXpirMhBS8cGNWSto8B5aWVFQ0X2n35/EBWm3XsG1TeR5ZEsxwdSbiWKFH3Dm0B9X0u+GUO/XPH0uupP71Zft7gqfCAvX5tYtlZKClWvmAakKaFt+IndW59vfVr3xlKgE1Gfi4QfRhuKFx5vdedXXfdz2BHKfjUFWpVQChWZf9RH6TTVeVUeShGbQuIqTRFp/pU/"}],"Parents":[{"/":"bafy2bzacecvqw436kn24hoa6q24opilkfpowz2tepzu2tp55rpxtephcg2iya"}],"ParentWeight":"2095848573","Height":122368,"ParentStateRoot":{"/":"bafy2bzaced35p3xlzah7tveoodnvgi4uh2v3r27r5jzsunqexq7tkmwqtyzsa"},"ParentMessageReceipts":{"/":"bafy2bzacede7lnmwuv6lcdpk3jeuilhljklzvypn7fupne43tb7lllfqcnf62"},"Messages":{"/":"bafy2bzaceboyaxnttplor7qgdjf64grkm72pxafwwo2pwrgqxxfsan4spwz4q"},"BLSAggregate":{"Type":2,"Data":"mae1h4T9egg968XZXgy+jwGomTVijPw4YuoEw+RX3VtL6qOqkPbVdr67vDkabrnYFXKExV/wYldpV5+1k2ZCZfVXhZnVO2Fdne1MWcnRMS493smNuqo3qbAphu3/uUUb"},"Timestamp":1595584000,"BlockSig":{"Type":2,"Data":"rvsvyZWl8RON4p1UTE86jXTh10sd/WYUJf4y2Jlgs1Iy1R+rl2XjsBRcQcEjXaOTFxkr1jP8uCtFUTo84xlgy1CUuMnxOiLFzPkWa6COtWWDlCrcvMkfMAuQTsEHyuX5"},"ForkSignaling":0},{"Miner":"t011101","Ticket":{"VRFProof":"sCKZ6obvGdKnBB5PQrAEAwXIkhRnG98XXa00whTaAl0/s1b8AViTdA46xTxw52AQF24LCQiBvapfhuQpYq9R3/85NlmfuHvTn7ndf/QbMvtewECvhNG9EAxmRPi2C1kO"},"ElectionProof":{"VRFProof":"qzeOFpArYKpuIX1sun6BfWnd0Ziyn20G8q/iSt0t4hSxFBAbnlN0L5vJrZLHJnogGBEXZWTrD1B9EId/UKzUaMGEBtp1speUTIzF4IcLy0U1uiYH0H3f4KHt8d6ob2PE"},"BeaconEntries":[{"Round":205552,"Data":"k0WKvac/9106OZt5ECA6vWMOCiAHOzm7NOmS0tc5r/E3sV2aWKZZzQgfZ+IE7s5HB6pODv6RmOXpXpVqFYyloTG9Vf2s8bb5J179+GGcs37b3gcvYfrLG5JQUu3Z+5B7"}],"WinPoStProof":[{"PoStProof":3,"ProofBytes":"tluECmrVPYcvmQXubY3b5T3b0J5aanDfJkzcpHd+N0aVHaYLjQh1wdcrInLFZg95iK6zqugZ7oWwXV1JqSsoDBigC6eSZyDA3DPvnv5XDoOphHC6ID2RaOhHO2MthnkQCrNkXUbM0rM88StQaReaBT+7FF5mUmaygtFx7HwOaC/6g1E22OxFhK8jlF/WCFj1jji6Tciwtpconi7/YxntsSmoQcdiFd7breTDwJb3+0nZjAQVtfqdOzOVV2hrwiij"}],"Parents":[{"/":"bafy2bzacecvqw436kn24hoa6q24opilkfpowz2tepzu2tp55rpxtephcg2iya"}],"ParentWeight":"2095848573","Height":122368,"ParentStateRoot":{"/":"bafy2bzaced35p3xlzah7tveoodnvgi4uh2v3r27r5jzsunqexq7tkmwqtyzsa"},"ParentMessageReceipts":{"/":"bafy2bzacede7lnmwuv6lcdpk3jeuilhljklzvypn7fupne43tb7lllfqcnf62"},"Messages":{"/":"bafy2bzaced6w47igyyia3wpsp2uls3y2wnxgrf2qwqbq52fp5fxvnzl5uyhbk"},"BLSAggregate":{"Type":2,"Data":"mae1h4T9egg968XZXgy+jwGomTVijPw4YuoEw+RX3VtL6qOqkPbVdr67vDkabrnYFXKExV/wYldpV5+1k2ZCZfVXhZnVO2Fdne1MWcnRMS493smNuqo3qbAphu3/uUUb"},"Timestamp":1595584000,"BlockSig":{"Type":2,"Data":"sgl014GmhIWcYreLCm2CJicOFdfZiXoQHacW7yzRIfwEYjqGX73L0WsxMhiuzN52CJidm6cPxSBo/Z9So4WrJxUPfROEMD5N6DAz3Zyy7badhbYvwmyN8ASSvxxwPe4G"},"ForkSignaling":0},{"Miner":"t0118768","Ticket":{"VRFProof":"iyB9N/1zM1y3YwX7I3o89pG2pVouzIzyKdGc7hmEz9JKE9kIy8EoErpip7cbaCYIEBLwLQgUUFRXWHjhxWhE03CU48xjiMlK1LngQXuXDef1+6OWGjRJw+TEo6QeBwjU"},"ElectionProof":{"VRFProof":"h1HmT8d0ZgqL1/srcDfz6jl5sKJFtHpOn1edpKg0mwzX1OTYo7qSh2jqUUJqBAPyFf9yevXB9tN9T3OKOwqcUA/wlHFDg7MyvJtfxQL1x4fJL1P8lKqzOLkQYQCfsgi6"},"BeaconEntries":[{"Round":205552,"Data":"k0WKvac/9106OZt5ECA6vWMOCiAHOzm7NOmS0tc5r/E3sV2aWKZZzQgfZ+IE7s5HB6pODv6RmOXpXpVqFYyloTG9Vf2s8bb5J179+GGcs37b3gcvYfrLG5JQUu3Z+5B7"}],"WinPoStProof":[{"PoStProof":3,"ProofBytes":"rR/xajira/NzetPCrzQby91ELzBYJ8h+c7Z+YzkMCWbpLfxQhFQJKxusLvdAVDn1oxfVM+zDEFqbvJHadAIET2wpKm30zR58lKV8qkjqomVSf8NiW48b2y1Y8GWYjFItAba7itS1k1+26iNs1MspBsEXjxxGs+KkaM8KtDeoQpFxynEqJZpwFtz7TlDmgN3auDchpC/Qqx2sQUtJQpAHDeVemdmGgZcfEw1zWoJVW/SX5GjGTcI4h7SO0OtBd5WD"}],"Parents":[{"/":"bafy2bzacecvqw436kn24hoa6q24opilkfpowz2tepzu2tp55rpxtephcg2iya"}],"ParentWeight":"2095848573","Height":122368,"ParentStateRoot":{"/":"bafy2bzaced35p3xlzah7tveoodnvgi4uh2v3r27r5jzsunqexq7tkmwqtyzsa"},"ParentMessageReceipts":{"/":"bafy2bzacede7lnmwuv6lcdpk3jeuilhljklzvypn7fupne43tb7lllfqcnf62"},"Messages":{"/":"bafy2bzaceajnsbuhxxx7nuot7pt5cwdfymxaak2jkbp2yiihvajgn6do5uso2"},"BLSAggregate":{"Type":2,"Data":"t4cXmPQhTSK5xL4SfQ0hpXhVKjQ4a4neZThLoQYmMVWYzBvnhB3d3Ade4Lg7X/68EBuiDkbgdih7GUdda7fev+l+auokAsleSvRlxK3NuHMGGKAbpnjwQoyu3Ojbk51s"},"Timestamp":1595584000,"BlockSig":{"Type":2,"Data":"txF5JIZgoM8fG4Ifa3L00X759ujXb8C0CUJfDB5Eoa82EhK0m6tpem69k12g3B9bClaNQlxoW7mK8H3IO7KQy2TzRV50dwGqv8HWL9XLW2P1phMrCkd5ynaCVriIr5uq"},"ForkSignaling":0}],"Height":122368},"id":1}
func (wm *WalletManager) GetTipSetByHeight(height uint64) (*TipSet, error) {
	params := []interface{}{ height, nil}

	tipSet := &TipSet{}

	result, err := wm.WalletClient.Call("Filecoin.ChainGetTipSetByHeight", params)
	if err != nil {
		return nil, err
	}

	//高度
	tipSet.Height = gjson.Get(result.Raw, "Height").Uint()

	//blocks
	tipSet.Blks = make([]FilBlockHeader, 0)
	tipSet.BlkCids = make([]string, 0)

	for _, blockHeaderJson := range gjson.Get(result.Raw, "Blocks").Array() {
		blockHeader, err := NewBlockHeader(&blockHeaderJson)
		if err != nil {
			return nil, err
		}
		tipSet.Blks = append(tipSet.Blks, blockHeader)
	}

	for cidIndex, cidJSON := range gjson.Get(result.Raw, "Cids").Array() {
		tipSet.Blks[ cidIndex ].BlockHeaderCid = gjson.Get(cidJSON.Raw, "/").String()
		tipSet.BlkCids = append(tipSet.BlkCids, tipSet.Blks[ cidIndex ].BlockHeaderCid)
	}

	return tipSet, nil
}

// GetBlockNumber
// {"jsonrpc":"2.0","result":{"Cids":[{"/":"bafy2bzaceczb5gpwzmfihank53ba43h4rjl23k5pcx7z43xdmo4p5fofh22qa"},{"/":"bafy2bzacedqopvxh452cn3quwua4qewpadmhlcciblpk653tsgni6grzatewi"}],"Blocks":[{"Miner":"t02020","Ticket":{"VRFProof":"tBAnDLr5XJx7uxmwrq17NeesRdY+dgXxBlLN2pPyGAMTaC+UPw8xPjh5PQ5GKX5EDFKNAN4yZFp/fb73F6i3Pato34Q/yuoj/SZ7sAItpiL/JeSgVjIOOZFBEv4u+K0K"},"ElectionProof":{"VRFProof":"obznWuJs20p9H/jpc9WiYdiHx8qzUusCJQ7mg1L8i8S4aj38i93vpcOO9eU1w3CpCPyEiArDNtOpDycWOyC/a/TxKPT6B6dcsI2sPqPfkVeI56aXErE86on0yohHiQ2P"},"BeaconEntries":[{"Round":205553,"Data":"gSneVmV85fYyZqj1ZFs1IAm/E9pz2BF17nllLx2zEMRFAEWxAm/hojE0tdypnZHGEvZvF5iZcfE7bmDGOv/MqL68pbS0cq/hLAmzLm9cBmIT5pBB3UsL8CeH6sirqxGC"}],"WinPoStProof":[{"PoStProof":3,"ProofBytes":"ssRkJlKHWssmh8abFvrN5x1rV3FQC9WAM2hYK2NNjDQEelUMx+CJ0P941uQylqCzkf7MQPJ0r3DjJgAPDOgPOBKsdPR4BXOurw1mY53Lx7xXpfwwROJqdea29tr3ktxHBZsArq3Sl5cCE43igpJsVcGhP6iD7Kb3L9w1gXcRkugnFgYoNZL3OEP5jLQY1z86opdTp1uwkr5eivEt+sqRSK+af7bC7GiQZ5Q61b2Vxs2+yuyd7+JsotDDdLwp5fXs"}],"Parents":[{"/":"bafy2bzacea24lcnctzolxc24k5utvvn33mgub4a3bfka7436jv32h3fviechg"},{"/":"bafy2bzaceaeqajdqvcfjgroissn5jhwcsk6yyyo5gh5h34igt5kdbmdmpny24"},{"/":"bafy2bzaceceld4xndb7tzrhmhoewlqhqstdjwtpfrenwpao7hzuhhf5s7hlzm"},{"/":"bafy2bzacebatzr6572caj66bjnodkkntvszmwyudujjrgl75vxlbl2hofxctk"},{"/":"bafy2bzaceb7roytvblia5jdl4g6z2jdoqzabqfepuef2speygklzrrba6evy2"}],"ParentWeight":"2095869309","Height":122369,"ParentStateRoot":{"/":"bafy2bzacedcsprjo7ipsouwhesmuy76aqu4mq4kwedc3cvq22wekv3bbbzbgg"},"ParentMessageReceipts":{"/":"bafy2bzacebadtki4dspp4edeuq4vekzfxvuohyhue6v2srovxxhxqfyp7gama"},"Messages":{"/":"bafy2bzacea6jmp6dzwdabjb6mo2mu6irgxubsiinoyc5vbfhtivqtqsw4jc7i"},"BLSAggregate":{"Type":2,"Data":"gkkY1vzxrXhhOL8PoGMhZ/9ZUuxXf0Q4zNey4PICHv89jDG4ylpGTUZ7WI/S+Yw6AvA5RCtzsqJ5cScF8JvkZhbpWjKMGhUKchZToR8sIYozlekY7QTSpPnMgRViDc+u"},"Timestamp":1595584025,"BlockSig":{"Type":2,"Data":"tfXovluNzXPsXJAr3CDWQtnXFTAJvSBjyyTZ2qghhBJxITKuDztqtCtmcFW2qq5VBsW7xAYNWZe3XmaXxtAo+10DC+TK0p45ZFzJYzd0+7mHwJZxS7pcu4nRERZCpgY8"},"ForkSignaling":0},{"Miner":"t01025","Ticket":{"VRFProof":"kiKgR5Yzty19xVnJRH3KijKm6Puzsp0ShK1ZKiJzxZI1Z6rZoqitdiUiZ4Z9xmSWCPlWfw5LFOEx5GcBPEvxff1nTMxO1l0/Mfz8+qZ2cLRCv6XGCBQKsuUYI3u/Esac"},"ElectionProof":{"VRFProof":"q+HGyokBxH7b3kv+VjfVPUJbA6OCBXsZ3Hbo0/yfjKVp3W1M/Xm5kki/C9wSr0yICzFEKoSXQrUDOkxFL5xKzY9doYDwJJX9/NSpDNQGs5g7bjwykdjdz+YIJm9J+X08"},"BeaconEntries":[{"Round":205553,"Data":"gSneVmV85fYyZqj1ZFs1IAm/E9pz2BF17nllLx2zEMRFAEWxAm/hojE0tdypnZHGEvZvF5iZcfE7bmDGOv/MqL68pbS0cq/hLAmzLm9cBmIT5pBB3UsL8CeH6sirqxGC"}],"WinPoStProof":[{"PoStProof":3,"ProofBytes":"ioNeFiUGuS52fHpowLovZGwn/DVjQDnTeheZAwDkw+L+FNov9EageJlwkI5kN8RMmKH+0uJgQQDi8MdoCrfxWcnd73aKxFQrlUmvFXHTlBtQPc1kaG1MDVbfpoXNFNnVC6n5MiH7+UFgvLMpNe06+xsIhCoouJwRDua67mZs0whl5vOermhVd65kiVKLHsCQhdJP/Bu+klrDA0iwGo64TD+p5psi8uxBASUqR6ffJ0YfBBAZYfoiiVHo7zrsaQ0N"}],"Parents":[{"/":"bafy2bzacea24lcnctzolxc24k5utvvn33mgub4a3bfka7436jv32h3fviechg"},{"/":"bafy2bzaceaeqajdqvcfjgroissn5jhwcsk6yyyo5gh5h34igt5kdbmdmpny24"},{"/":"bafy2bzaceceld4xndb7tzrhmhoewlqhqstdjwtpfrenwpao7hzuhhf5s7hlzm"},{"/":"bafy2bzacebatzr6572caj66bjnodkkntvszmwyudujjrgl75vxlbl2hofxctk"},{"/":"bafy2bzaceb7roytvblia5jdl4g6z2jdoqzabqfepuef2speygklzrrba6evy2"}],"ParentWeight":"2095869309","Height":122369,"ParentStateRoot":{"/":"bafy2bzacedcsprjo7ipsouwhesmuy76aqu4mq4kwedc3cvq22wekv3bbbzbgg"},"ParentMessageReceipts":{"/":"bafy2bzacebadtki4dspp4edeuq4vekzfxvuohyhue6v2srovxxhxqfyp7gama"},"Messages":{"/":"bafy2bzacec7r25pykl6mv4emdfetuzqfy7nr6ktqtfkn3jfbsxep2s5m5zqpu"},"BLSAggregate":{"Type":2,"Data":"pIa4n+boNM3ASENYFW6w8COwb+QG721i1E4Y26UY9Lv3s/8KAleTPEGAe0KLLzn1Ge3n694EUOzvtbDlUdkDdta4Em6RYHkwCdkuowneC6+qpzaWqPPFk64BeYQ8Zux/"},"Timestamp":1595584025,"BlockSig":{"Type":2,"Data":"kKXJ1TFhUvkEgu4d3O00GdzHYDKf1c+64R8SdLBt0OX7onJUj+Wd0LJBC7GK7bZDEV4JYJK4109i+Q+W58cK8H8LtPfQvWWG9FQMNufJn1nGw2K61PoUL+cIq4pSgR3Z"},"ForkSignaling":0}],"Height":122369},"id":1}
func (wm *WalletManager) GetMaxTipsetHeight() (uint64, error) {
	param := make([]interface{}, 0)
	result, err := wm.WalletClient.Call("Filecoin.ChainHead", param)
	if err != nil {
		return 0, err
	}

	var tipSet TipSet
	err = json.Unmarshal([]byte(result.Raw), &tipSet)
	if err != nil {
		return 0, err
	}
	return tipSet.Height, nil
}

// {"ActiveSyncs":[{"Base":{"Cids":[{"/":"bafy2bzacecnamqgqmifpluoeldx7zzglxcljo6oja4vrmtj7432rphldpdmm2"}],"Blocks":[{"Miner":"t00","Ticket":{"VRFProof":"X4oDOWswmmD7fT0z3RNIPQVGS85f2dBhceeowoDiQhY="},"ElectionProof":{"WinCount":0,"VRFProof":null},"BeaconEntries":[{"Round":0,"Data":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="}],"WinPoStProof":null,"Parents":[{"/":"bafyreiaqpwbbyjo4a42saasj36kkrpv4tsherf2e7bvezkert2a7dhonoi"}],"ParentWeight":"0","Height":0,"ParentStateRoot":{"/":"bafy2bzacech3yb7xlb7c57v2xh7rvmt4skeidk7z2g36llksaz4biflblbt24"},"ParentMessageReceipts":{"/":"bafy2bzacedswlcz5ddgqnyo3sak3jmhmkxashisnlpq6ujgyhe4mlobzpnhs6"},"Messages":{"/":"bafy2bzacecmda75ovposbdateg7eyhwij65zklgyijgcjwynlklmqazpwlhba"},"BLSAggregate":null,"Timestamp":1598306400,"BlockSig":null,"ForkSignaling":0,"ParentBaseFee":"100000000"}],"Height":0},"Target":{"Cids":[{"/":"bafy2bzacedp5jq2a4hprhksnz6cwzcbar7pmpf2ytsr67bwtcadc3t3ddvfw6"}],"Blocks":[{"Miner":"t02838","Ticket":{"VRFProof":"haWIp6tSyXhePXyNclQaHH8jgTmMDdT6xlJvW4Ked//UymUbF081Rixu5rh079upA5wIU8XCgUO2l8oo3PEl21SG0EdEY/2eNtbLj/78+IKRQEVkQ0sGwz/21jnp/RsY"},"ElectionProof":{"WinCount":1,"VRFProof":"peUWvR2CnnliM3hGK9aqpgv9ILxbflMuvbaC/7me242IvS5WYZyc4Qq0t8DmugCmF3b4ecQCPipFz2vBYnfQDtnLQrLf07GJizVj/ReAL5yUl7N8LO8IBCiK+1bxpKy6"},"BeaconEntries":[{"Round":226789,"Data":"i3mg89T0ZxyZ4sIFVUzOamdKQx/tCyuKbXi2x5lhXpslar3WdDXp8+nhGjsNQ71JC0BTBHfbVdBmWTGU3trpoVwKl1e7E8Uil4PjHOtuIsqfW8aJPxlCGXs87eeDfA9R"},{"Round":226790,"Data":"oLw9Jn0NQ3zMlnxZMnVdcRpVamAsA6SVShR1KKXj8MsISJfIy4A4rcosVegdH/1ODwiNKxSEJcungO7An9XTHHpppEkqK+nHMoI/q7ZSdkH72B4EiaPgjbY8L05uvp1w"},{"Round":226791,"Data":"tb56SYtPvSlheGkf3FSITxjgy9hoJHqZZcMr8/Puapzr17h/JPLw62/XSPBzRYNPEkdqJ2qrY2vs2kDor+nymZZZ5DpXBynXcINa6wXTQoLmFWb/kqIpS8S+0T7baB1j"},{"Round":226792,"Data":"jGgofcxNF1r0B1GHeMuCqR5SDjYUyIzBKI1hAH4Mw/OSJjXGlp0KM99aRttTelwdA6oUp3QkGQdDQVdPfD/KOe5n8wRcXMe8TanlkdQhDPH16YPYowjXWNMKwLgj23Sm"},{"Round":226793,"Data":"gy9nxiqogVzhqK+ycrKoQzx+yXkoprRFnqUmRt4+uJGuUPszD6OK61ZumAnNbJERAowfCttMaQNMdNxB6Zru7qh87DZQ8KfvODdtewXye0aT2Ust8BR8/zeF2ttNsS88"},{"Round":226794,"Data":"jpJy3d1oeWjqj44OaI+dzWj4g9GGmBSP27T6MArf9J8nbGgUxTcudwc7FQhyXmrWB3G6tvQzGWTNZ5C/O+LdNy/VT26WlMlftG88TDFtB9IuYkxHAYg2cFPrGnh931cs"},{"Round":226795,"Data":"uKEFx9Pcq91HDeXjH0ZiBzeX9f7EVIUz52DsJ5U7F3wJ1+6LicvF0RCnmCyL1A8KC3Zp3Xd+62XryfPqpX4ib67/1v1d0lyOSE+GAOVtUDxUJ4R5m8HSQc+W+52v0rWk"},{"Round":226796,"Data":"ryd4QY/kAtIhiiJi2AEWDSqg4wiXIUcja5nosHdc61VV7tPO6mqHGrcUiTD08aNLC1njTCuniShoJx+XbQnnwcvwKjvNreDdg8vQ8ESjZkeZ8YPtrDikc754DYjOuIKb"},{"Round":226797,"Data":"rffVY9niX57wGC2IlB2iZSfmr9J+p512Voc/kCqxcLIqcLkDbNrrqSvNm3R0lMFDEG0zSCCOqrb6qj37hmyt6dXLoezyeXZY0XuJjwPCSFe1iJvXrzIJUMVgmDS5LsVq"},{"Round":226798,"Data":"i9FGszNlBW2u/fzYM9fLRsKeffrFl5+udFdxsILEVlgHjRmepH00TIX0JojQdO1+Ea04tq751zRspCE/IHX7QQgLQPytvd21kHmRcqEPgNE0LUYCYfsd6tl5T+aTjUzu"},{"Round":226799,"Data":"megQWojVn2BbNq1E80nX2cNRJgVlPiAneSiPV120qbYsG9niQG8WmeNsHudhaFUgC5MdUIxoMh0YDGrJcBuFIGCfio0lTGhs/yaFY9OwvyOGf4r78SCj0RqTntN+1Dh+"},{"Round":226800,"Data":"l13gG996+2oQrKxridMJh3yeJek+t8HEP7oAiy7d1MdM+zIBwQ8JwcOEjoUg65YbEk1zgbMgn87EYrKRYq3hE1fXQdfjDu9vhqYb7jhjCImKxXNYSddQmKWt/4/zmgcA"},{"Round":226801,"Data":"sWHZ/dlAwDTTX/FBbjqsD4xuHWe6xKU2fSufM5gB6PWql5MPga4wRcUs3OOqHxw/FIkHzHYAHEa8kY9/jhkNonMJhR1D3S2faZw0m34LC87Tj2ZoPaS/XwGhqEwQ3OJ0"},{"Round":226802,"Data":"pyBvRuYwGMWJKY5ojMcltTVw3JTX4POZym+HldlaLQaqfU399ZLHPopbB/384FFVChzqt2Fv0TuJCS90+2EOhiTSV89bpoHLDpSwl/+ixDKT8ndpcyIqhAep/YDOKF2B"},{"Round":226803,"Data":"sVfVMkbRr2791fo9H3cMLpH7tyntpPl+VXQC0F462EWaD+VVp7+Mcw8PHT8GcW7uEV4Uuh0WZBq+6xYyt7D/dEWCWh+glU/VwVb/9xMfZsqswjmIT/foahCSCh0qno78"}],"WinPoStProof":[{"PoStProof":3,"ProofBytes":"hA5WOmBzZAyVT0M4NkIaCbvMVeRXklZ60rTKMdVZMWsVbtmorgNwkDpsd/mootO1sREnijTJb6Ef4KwFhFYIn91tNW3pmbrRKXHF3MxnOcu4FsurD2OY2FO5Wfm94R93FTu4AY7HnKN3pz8BRF/geg/R59FANvrNZ1QWADCN6X3kzC+XeTpHh33vtE7tEa4dkP/zp+N0FFvgTBO3mPLz7eS3RZ+0wd1+n0bJtRhjuqBEeu4LV+itmZMaNWGttb56"}],"Parents":[{"/":"bafy2bzaced2ibvl5txexdx7rl3kmzlmzblzhz42xs6g3k56k45v4n4lnbeqly"}],"ParentWeight":"2617125071","Height":130959,"ParentStateRoot":{"/":"bafy2bzacearip4rzokw3vqff4hsunlzgyng45htraasf2d4bv7tui7lk47t3c"},"ParentMessageReceipts":{"/":"bafy2bzacedrjkcitpysoa6ohdc332zui37ltgkn6pj35dw6hu7wujx5dklfim"},"Messages":{"/":"bafy2bzacebt3qdn3sy2tzqigwdm3ihrjfwf3qovsg6orx37er7iphwydmgkpq"},"BLSAggregate":{"Type":2,"Data":"prQgMJCU3UGxsmuUr31fmjF8L2iqwJF/eOvXriJhZSNU/oc6rZesCTTZR7cuCZrzCzMHu0SwYvAq25bQzcQ6VRFpVySiyiDlua9OhQSgDthbbHmp8o1szebLkIMcq0IV"},"Timestamp":1602235170,"BlockSig":{"Type":2,"Data":"j1Es240u1TvBG7mlS6UjAmPbYffnvOjwuArdoS+JLj4MBZOfbtyfxHz4nrqWFacnAItpvBAsJGKXWQo3IY8jxghR0X3upUSsxqcSn8bjytOLGwtWLf2QNl5oIcG1D+5C"},"ForkSignaling":0,"ParentBaseFee":"400988190"}],"Height":130959},"Stage":3,"Height":5320,"Start":"2020-10-12T15:26:21.442312647Z","End":"0001-01-01T00:00:00Z","Message":""},{"Base":null,"Target":null,"Stage":0,"Height":0,"Start":"0001-01-01T00:00:00Z","End":"0001-01-01T00:00:00Z","Message":""},{"Base":null,"Target":null,"Stage":0,"Height":0,"Start":"0001-01-01T00:00:00Z","End":"0001-01-01T00:00:00Z","Message":""}],"VMApplied":0}
func (wm *WalletManager) GetSyncState() (uint64, error) {
	param := make([]interface{}, 0)
	result, err := wm.WalletClient.Call("Filecoin.SyncState", param)
	if err != nil {
		return 0, err
	}

	var list ActiveSyncList
	err = json.Unmarshal([]byte(result.Raw), &list)
	if err != nil {
		return 0, err
	}

	syncHeight := uint64(0)
	for _, activeSync := range list.ActiveSyncs{
		if activeSync.Height > syncHeight {
			syncHeight = activeSync.Height
		}
	}

	return syncHeight, nil
}

// GetAddrBalance
//{"jsonrpc":"2.0","result":"253178184999999999989664580","id":1}
// Filecoin.StateGetActor , result: {"Code":{"/":"bafkqadlgnfwc6mjpmfrwg33vnz2a"},"Head":{"/":"bafy2bzaceaok4ygzwpbhvxilmtazy66shipkm3p5ko6t2eu6ymi63pf55wvui"},"Nonce":8,"Balance":"242838089036848770421"}
func (wm *WalletManager) GetAddrBalance(address string) (*AddrBalance, error) {

	blockCids := make([]interface{}, 0)

	params := []interface{}{
		address,
		blockCids,
	}
	result, err := wm.WalletClient.Call("Filecoin.StateGetActor", params)
	if err != nil {
		return &AddrBalance{Address: address, Balance: big.NewInt(0), Nonce: uint64(0)}, nil
	}

	balance, _ := big.NewInt(0).SetString( result.Get("Balance").Str, 10)
	nonce := uint64(result.Get("Nonce").Uint())
	realBalance := common.BigIntToDecimals(balance, wm.Decimal() )
	return &AddrBalance{Address: address, Balance: balance, RealBalance: &realBalance, Nonce: nonce}, nil

	//params := []interface{}{
	//	address,
	//}
	//result, err := wm.WalletClient.Call("Filecoin.WalletBalance", params)
	//if err != nil {
	//	return &AddrBalance{Address: address, Balance: big.NewInt(0), Nonce: uint64(0)}, nil
	//}
	//balance, ok := big.NewInt(0).SetString( result.Str, 10)
	//if !ok {
	//	return &AddrBalance{Address: address, Balance: big.NewInt(0), Nonce: uint64(0)}, nil
	//}
	//realBalance := common.BigIntToDecimals(balance, wm.Decimal() )
	//return &AddrBalance{Address: address, Balance: balance, RealBalance: &realBalance, Nonce: uint64(0)}, nil
}

// MpoolGetNonce
//rpc返回结果 : {"jsonrpc":"2.0","result":170152,"id":1}
func (wm *WalletManager) GetAddrOnChainNonce(address string) (uint64, error) {
	//params := []interface{}{
	//	address,
	//}
	//result, err := wm.WalletClient.Call("Filecoin.MpoolGetNonce", params)
	//if err != nil {
	//	return 0, err
	//}
	//nonce, err := strconv.ParseInt(result.String(), 10, 64)
	//if err != nil {
	//	return 0, err
	//}
	//
	balanceItem, err := wm.GetAddrBalance(address)
	if err != nil {
		return 0, err
	}

	return balanceItem.Nonce, nil
}

// GetTransactionReceipt
// //{"jsonrpc":"2.0","result":{"BlsMessages":[{"Version":0,"To":"t021661","From":"t3vpdi3tg2oppc4bicav723n3e7bzlfaxhvcxjbl72qpe7hra4aggh6oekit3saswmrq5trjap2kdyubpfwxcq","Nonce":68773,"Value":"0","GasPrice":"1","GasLimit":1000000,"Method":6,"Params":"hgMZlyzYKlgmAAFVwh8g78Wdao2sMOq1YR/WjzB33viwbeHwYyfenrp2Qq9pGU4aAAGhw4AaAJpCbw=="},{"Version":0,"To":"t021661","From":"t3vpdi3tg2oppc4bicav723n3e7bzlfaxhvcxjbl72qpe7hra4aggh6oekit3saswmrq5trjap2kdyubpfwxcq","Nonce":68774,"Value":"0","GasPrice":"1","GasLimit":1000000,"Method":6,"Params":"hgMZlzLYKlgmAAFVwh8gzIjaV/osvdivwN4QjAQEd8Y9LwlB/ASCwzm3IVOW3jYaAAGhyIAaAJpCbw=="},{"Version":0,"To":"t0122507","From":"t3ufwpcpo5f4goc2xn3q3rnj32dgrzcud5q7uj3sctxdx2jcr2as7rslztiky7lv6b5vhup75s3cjwhmzwv2va","Nonce":7,"Value":"1000","GasPrice":"1","GasLimit":100000000000,"Method":5,"Params":"hAGBAIGCCFjAorsoEG0M4kuBcV/fOIsqUK42nFBdUXC3biMtKrLjXjo/3FDKZweFSAGswEyYcusEtt1OZM3tO+4DM0OKgmp/qsvgoSVEnbzRxOwdcuAgOiTKVCVMIANb9oVjTh+yMFhGFW1Sy2+u2chgCTMkDQXJcAJ1hQbxyPh6l6wFZPQdZkfJxlk/MYb+3DThTlRFdIvfoYF4T7afAg1ti/APHpi0ii8+QwZnVl0/abBUnw0BRUqxfB657tN9H6eoAJhvvvqdQQA="},{"Version":0,"To":"t0121681","From":"t3q7sm6stkn3ynqkiinxnprijynpwwpbigdq2umzk7w54eii4a6jlzb7yz23u5l2lg4xm43w5zopn74n5aybra","Nonce":33076,"Value":"0","GasPrice":"1","GasLimit":1000000,"Method":6,"Params":"hgMaAAJUfdgqWCYAAVXCHyCO+B0AptwJsd26rx1OMo2AVXqTsYSnW2mmfNPgX8MAOxoAAZ1KgBoAFPMe"},{"Version":0,"To":"t0121681","From":"t3q7sm6stkn3ynqkiinxnprijynpwwpbigdq2umzk7w54eii4a6jlzb7yz23u5l2lg4xm43w5zopn74n5aybra","Nonce":33077,"Value":"0","GasPrice":"1","GasLimit":1000000,"Method":6,"Params":"hgMaAAKIcNgqWCYAAVXCHyCFtFiQ216K21jSpt+ZHGgPoziR3ddeCq0p7iy6GNF7KBoAAZTYgBoAFPMe"},{"Version":0,"To":"t0121681","From":"t3q7sm6stkn3ynqkiinxnprijynpwwpbigdq2umzk7w54eii4a6jlzb7yz23u5l2lg4xm43w5zopn74n5aybra","Nonce":33078,"Value":"0","GasPrice":"1","GasLimit":1000000,"Method":6,"Params":"hgMaAAPUEdgqWCYAAVXCHyA/M7w8EUwOz181SnDm3B3LcevtfWywHGcOnY3UVvVaHxoAAZ05gBoAFPMe"},{"Version":0,"To":"t0121681","From":"t3q7sm6stkn3ynqkiinxnprijynpwwpbigdq2umzk7w54eii4a6jlzb7yz23u5l2lg4xm43w5zopn74n5aybra","Nonce":33079,"Value":"0","GasPrice":"1","GasLimit":1000000,"Method":6,"Params":"hgMaAAWoAdgqWCYAAVXCHyCZ0tQsh8BZvvCW9/8CJbH7sJRVZZ+9rNiQvRrOgkv+OBoAAZzhgBoAFPMe"},{"Version":0,"To":"t0121681","From":"t3q7sm6stkn3ynqkiinxnprijynpwwpbigdq2umzk7w54eii4a6jlzb7yz23u5l2lg4xm43w5zopn74n5aybra","Nonce":33080,"Value":"0","GasPrice":"1","GasLimit":1000000,"Method":6,"Params":"hgMaAAV4B9gqWCYAAVXCHyAZ3aOFbRNBm8AJImJIqYog5qZ18BHXnny9/2VthSFWDhoAAZ0jgBoAFPMe"},{"Version":0,"To":"t0121681","From":"t3q7sm6stkn3ynqkiinxnprijynpwwpbigdq2umzk7w54eii4a6jlzb7yz23u5l2lg4xm43w5zopn74n5aybra","Nonce":33081,"Value":"0","GasPrice":"1","GasLimit":1000000,"Method":6,"Params":"hgMaAAYsB9gqWCYAAVXCHyADT9DB2XFFdRoHcpbcMHeSvSWTTY9Y9TqG+VKqMmBSSRoAAZ1GgBoAFPMe"},{"Version":0,"To":"t0121681","From":"t3q7sm6stkn3ynqkiinxnprijynpwwpbigdq2umzk7w54eii4a6jlzb7yz23u5l2lg4xm43w5zopn74n5aybra","Nonce":33082,"Value":"0","GasPrice":"1","GasLimit":1000000,"Method":6,"Params":"hgMaAAVMB9gqWCYAAVXCHyCw6KUqZOqO8X8sHKKB2QlOjtHuuFIuTsfi2TEq/h4CGBoAAZlXgBoAFPMe"},{"Version":0,"To":"t0121681","From":"t3q7sm6stkn3ynqkiinxnprijynpwwpbigdq2umzk7w54eii4a6jlzb7yz23u5l2lg4xm43w5zopn74n5aybra","Nonce":33083,"Value":"0","GasPrice":"1","GasLimit":1000000,"Method":6,"Params":"hgMaAAXgAtgqWCYAAVXCHyA/5vTQ7aiZ/LWlh0UZxUhDi6RLTS6EJz2bSbD1wxXVRhoAAZz9gBoAFPMe"},{"Version":0,"To":"t0121681","From":"t3q7sm6stkn3ynqkiinxnprijynpwwpbigdq2umzk7w54eii4a6jlzb7yz23u5l2lg4xm43w5zopn74n5aybra","Nonce":33084,"Value":"0","GasPrice":"1","GasLimit":1000000,"Method":6,"Params":"hgMaAAQwHNgqWCYAAVXCHyDRyA65srgBUmzA6Mv9t+OG6B5d1gXMlo9pJ1nRWWZuBxoAAZ4tgBoAFPMe"},{"Version":0,"To":"t0121681","From":"t3q7sm6stkn3ynqkiinxnprijynpwwpbigdq2umzk7w54eii4a6jlzb7yz23u5l2lg4xm43w5zopn74n5aybra","Nonce":33085,"Value":"0","GasPrice":"1","GasLimit":1000000,"Method":6,"Params":"hgMaAAb4BNgqWCYAAVXCHyBkmk85/8Fhm8KoXNBOu5q/5OlIX38umspxhOiqP0fnYBoAAZ0ygBoAFPMe"},{"Version":0,"To":"t04067","From":"t3rbq4ijj5owypedib5txu7fbmeq3x27znjgd3ctpchmuu27gilaqnmqe5vw3v6emtv7tzj3trjbhpux4j4m4a","Nonce":13414,"Value":"113624963406258","GasPrice":"1","GasLimit":1000000,"Method":7,"Params":"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"},{"Version":0,"To":"t0120843","From":"t3w3ycyp42virtnhxrh3l7zt3nvvydyyofyfwvytiqwye2alxgul4oecnenahaer3ydki33dwzndh72dt2hrfa","Nonce":1722,"Value":"0","GasPrice":"1","GasLimit":1000000,"Method":6,"Params":"hgMZBY/YKlgmAAFVwh8gBPNNDQ60nk5YyiGwwdHnaD2XKU1fYHi//Fhq5nG3aSIaAAGhEYAaAJpIaQ=="},{"Version":0,"To":"t0118768","From":"t3qyssqgkgqxa6ims36v3zsfutjhmj5zeljfob7nfsyhkq7ursmkjsdt3bkapbyrhetuobjfzknls2ianmeona","Nonce":122753,"Value":"113624963406258","GasPrice":"1","GasLimit":1000000,"Method":7,"Params":"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"},{"Version":0,"To":"t0121289","From":"t3v7rz7tkyhtscerb3hwyloyh5424yxj4tc2kjc7wg7ied35bxodrlnfbypomg7zetkomvplmpxlcjvz6jqabq","Nonce":9726,"Value":"113624963406258","GasPrice":"1","GasLimit":1000000,"Method":7,"Params":"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"},{"Version":0,"To":"t0121289","From":"t3v7rz7tkyhtscerb3hwyloyh5424yxj4tc2kjc7wg7ied35bxodrlnfbypomg7zetkomvplmpxlcjvz6jqabq","Nonce":9727,"Value":"113624963406258","GasPrice":"1","GasLimit":1000000,"Method":7,"Params":"ghkTYFkHgJoACQDQrl0T4XfjUxLZEAdBGK1uI6ozwcXz5+2wpGl0Ktlv2JqPbzjR7zHnNVzYTJhCYY4bcL7bU2ZnC1zVDsGfrPiXbveI1lMKJ3+Fmi48gBirQwKzE31kGv7bhGtnrgB8aimm0i2TTuxyufe04lIZpXXJpoeAg+uXKZ7N4niCHr4F0QEAZKEij6qSqxatD7ENYARyNe7dwIZ1u0KnHYmGqmbbxYcBggCRO9oRQvrRrrVD1s6AnljSNEqzPBBlJY/oUxphQ9NzDYa6ve+g6O4bhv+8a/xCjHkAWOO3Fq+2tD85ImkeyptIYYytYsSPtYOitWarIstAZZHSVMAXUdSoRCEtnkn9cHpvFLEACbqy20+Kvw5aqapmyQ/UDfrNygqUW9brlgv8k3zKQJQttYVAM/959Gzsxyf1TTcv45uLLpkdFbkxeC6XUaUTg0geBITSJzxDXltFssZhXNNcJe2yMGHEC5fefhqLpcwc5carDe5ufDYa5yxU/cIIJw5loKW2P8da2QfNlBlUdkA0a786eABX9hqEs43kY4xkoJP63niM1qKM745bwmCKW99+MaRB0e8EAjnc05CHiJL6KFvgWsurkqfay0O3RtIerbsuoTC8g9+JxvUjsTpkzSIZ1gBAzEHEPAwe7ZXYz9XeJDIT0afZjNZaTWijMljewCLFQ/lo8GXTkJHEaClpq7ZrP6Fud/beUHVPNpYOBIkaYK07Ccq/kbPdLVUvoDaoccPIp84bwTK/uOhbazv2Opylkai2S/jyntzgajqeQknl2BYMmhKzAmrEt8dGSvVirzstAcX/rt/bndikToCFUK5r/YHiGL4Jw5SG4TDqCCm5cSIyqA92eWiFJj6K08quMwiwEgJqgiU5iSkOoB4PejyivA6si/ca1EPgd36/N+V64MORBkG1g2WzgwNqS0Gzv7w9vv16/b987HX/nB+P/UnV96x6NZV6hbzAiRrYQF24KrN9+tlmFMrsz74B0y1+zW9D3eYCuI7Y7VGJY3xqVDqNj4Yk4m4+PoLyEZbuY+wXXXMUUYo/yERGmriRGsBmT+3HUp11vWtJ9kKAQ01HaPqCVpTm4AZdUvPf8aRYJ4f48W9Fq6Bsi8E18mIJJx9TEnmX/Ezp2Ktr4xZqeW6hOQs4aQydY8Lgr8iaI3OsGVUqvjl00r/F2pTNRFOyBmIQF/JxYLYNrKvHMoGluLS2xTB9sLaWAdd9EZZVWB3NxfW73q8NFMA8zMjv7lovbtXdfh3SWn1hhLeKGdYPQLV3Tr3CNbnYL00fQJhRFj/FQV7bQz27O3J7quwhA2lk653POYgQNv10YtWOe6HUMpnN4np4J5a8WW8FkkNO3ScP95RL8xTfEnzy9cjxmO7kx2BIwSlrZL8Z2b95f/eQt2m6uHz2+QJoZZMY99t6Z+Ky45PlPPDt4tlRU8L1CG0hxtvCGmDW242pLis/imi/lmzVp2d516GsdqIG4aGhr3Z/2UcrAD2i/VKilDckKc+qNDU82UgyQVS27kL3G4CNc/G0dyAcLIUCO3U7DnfAGxsn1PVdrzzlmlU4CS9gb4h5WW+cWOHIFSI2czDIzPoEhjeXXbijz5VM8GzPnVeqIOU6DBIl1+Q5rd/rVCVqR8G/axGNH3Pvxq/J5wuK4ePSRHSqxXp5UwjNBcj2+awK+LSN4NYvZScO1rEJpzE++uahjXWthXJe0c1yan/6/X5n/PMIk+nHCbg/AuegBvy1wXXDzerLlBZ88qgGmCXKQ+KhaN73+R0KWnPfYBPK4DV88e16aJAuZrmK0UBduFJ7wEQTL6fk/RO6LLSgHdmPl8/FmbpNgFr3nNrUhKWM92wnakxu0R4c2I5U+Cx1L2igs1udYnAIM4G/umcFgwiYt72DYHjxl4w8H5VNuD1PFy1D9jZNNEQLag0d3me6fmohcvzqcWuSeYVABFpW5nSVHgVngVqiv0Zj/IlE8iGUSKAZsKGf1nmkI6V9nxjFdt9rPt2P/nyhZpWgnRfH79bXWSmEjjjxUqCKRiCeuMcDQmPc/YYpt6eDmYfGH641n+VcOy+5m41YVDb8/GVjsDlxb3wjJq5XkSLwoGy0xZXWKKokVWqtPTBd76CaOZdvjY4pTWXdi8Zs5KnpCxG9V6UKv9N7F7C1i92O8Rz/3ohD6HmxE3llN6XzzA8FpsP4zRJdbT+gGrrhzzpLAE0fXDF6cmaumxg6VfqkGgYd4iMo0PBjUoar6VNEGZYsMrIWcvpv5vr8BjoR3vSgBBsYN4KcMn8aJ4M9iMbPxJtKEtwZzzs4rPnOMLGQ2K9C8+OOy3Oj2Gzs9YJq5uaIey43WKFLPTrVygoaIQ52IL7KJAXwa/4OjGEBpSC8RLGQJ18IyUN6rolxEq5EmsOiulAp/8ECCg5mWpFRdtXdcZakPhm+jeuL0KFt/YGjnQRvM9e0lWAFN2o1JXencC8HZPWYmG4+/XAb8c8+KYYLn7zwE+5Pl5y2PCJtWjchwaRodK0Wj5vFzmGnU5QK03gZ0DgoUO2M+ebugKSi4V5Fr9MJ3YBWWgq8kuhIwNhuhA=="}],"SecpkMessages":[{"Message":{"Version":0,"To":"t1wh2fhzvb5rcfoleedkupov442qp4hw34kzm52ki","From":"t1hw4amnow4gsgk2ottjdpdverfwhaznyrslsmoni","Nonce":102027,"Value":"50000000000000000000","GasPrice":"0","GasLimit":10000,"Method":0,"Params":""},"Signature":{"Type":1,"Data":"LVyd5LVrTQrBjd1yWT+5W/hpCCbSgGcAq2+dh4P6ktpPi9dqbSeDodHaj12kljJj3t8UEJNR1PdJDtJLWRy32gA="}},{"Message":{"Version":0,"To":"t1wh2fhzvb5rcfoleedkupov442qp4hw34kzm52ki","From":"t1hw4amnow4gsgk2ottjdpdverfwhaznyrslsmoni","Nonce":102028,"Value":"50000000000000000000","GasPrice":"0","GasLimit":10000,"Method":0,"Params":""},"Signature":{"Type":1,"Data":"TQRG6OyFQWZkVFBXb7pEsEiRCgaMn65rH8DwE4cQNuk+iEdnlHz+0T4VMQc3F36QAkHMQo8MRLKkgfhZPc4IZwE="}}],"Cids":[{"/":"bafy2bzacebehdtyulmuwnli7kehcdzhf7e7gnfxwcdwpelm5q7r63vuelya7a"},{"/":"bafy2bzacedhss6fmspdhupkux454xucetejmi74c7spqm32ddukf6sxiievey"},{"/":"bafy2bzaceah32k363smx3d7ppkkmaho4fonkxhyonpf3wyw4724op652dm4wo"},{"/":"bafy2bzaceafw3xgfjx5phdntsqggwssko3gnt4rwhbftjjzmictodk6qpds6w"},{"/":"bafy2bzacedexamhqmmifkcjfqbfpxo5b3ppjh2b6mdts5o7zvdoygecmdnc2w"},{"/":"bafy2bzaceacuo3phoa45wbpsge7s6f4wxqhfhi2w6koe2pkjmrf2itdj6jl6g"},{"/":"bafy2bzacebomwtsdtlto53hcybq6uqubsxe6q3plqeeualji2dsk3sgy3krfi"},{"/":"bafy2bzacedujiw7264z6iaevm7jmsqkeng4bhajgjdy7mux7zmf47ko2umice"},{"/":"bafy2bzaced7ar6fr266lgpxirdk27kirszxkoeyehvz57xlu5iz4u7gyfgcgq"},{"/":"bafy2bzacea2ryrhfaahz3dtrdccxo5kebhqftx24it3ddssvmuj4n3cgq4oq4"},{"/":"bafy2bzacea37yi33ld7vwpp7q5qactzh6qkcdbrjzi4gdbpvzfxkrgvqkyry4"},{"/":"bafy2bzaceblvbrugqma6gqdb53pwtudd7x65krvecezdb5tndrcsg4s3chudm"},{"/":"bafy2bzaceawpa3nwq6o2xsa6emfs6lb4q6qx2ntxoftbuwu5frxpelgl3qkxu"},{"/":"bafy2bzaceagsy2tap4bgkrwr2kvf2arxqek65wq3g23np4p4npd566xl27xxm"},{"/":"bafy2bzacecmcn4vnkqh3jvlobz6tfmowvkwdxajivyuycuyjr5rjbhgtnhzxc"},{"/":"bafy2bzacedrkd3q6g2jm4zsx3lhocxw3c76ivxv6o7jwnnjbjkus52cr4v67y"},{"/":"bafy2bzacebi6dc7tnfnvniokztzukiydkh2vjtntuxzffzr5h2d2mafc353tm"},{"/":"bafy2bzaceanztjdyehtygknzvsq5hlnebzinvxrdbv3u6tlg7ldhuzhuqbri2"},{"/":"bafy2bzacebxpq7qtous62tixy4qxwdhejvzh2qrblayq43lkitlhbfyo6ugas"},{"/":"bafy2bzacebmgwsmhibuz7hrueg2eiqyxneypuatz4ajhh6pfhc5w5prgrclqk"}]},"id":1}
func (wm *WalletManager) GetTransactionReceipt(txCid string) (int64, int64, error) {
	callMsg := map[string]interface{}{
		"/" : txCid,
	}

	params := []interface{}{ callMsg, nil}

	result, err := wm.WalletClient.Call("Filecoin.StateGetReceipt", params)
	if err != nil {
		return -1, -1, err
	}

	exitCode := int64(-1)
	gasUsed := int64(-1)

	hasCode := gjson.Get(result.Raw, "ExitCode").Exists()
	if hasCode {
		exitCode = gjson.Get(result.Raw, "ExitCode").Int()
		gasUsed = gjson.Get(result.Raw, "GasUsed").Int()
	}

	//wm.Log.Std.Info("transaction get receipt result, hash : %v, exitCode : %d, result string : ", txCid, exitCode, result.Str )

	return exitCode, gasUsed, nil

	//return OK_ExitCode, 0, nil
}

func (wm *WalletManager) SendRawTransaction( message *filecoinTransaction.Message, signature, accessToken string) (string, error){
	//js, _ := json.Marshal(message)
	//fmt.Println("js : ", string(js))

	sigData, _ := hex.DecodeString(signature)
	sig := crypto.Signature{
		Type: 1,
		Data: sigData,
	}

	callMsg := map[string]interface{}{
		"message" :  message,
		"signature" : sig,
	}

	params := []interface{}{ callMsg }

	result, err := wm.WalletClient.CallWithToken(accessToken, "Filecoin.MpoolPush", params)

	if err != nil {
		return "", err
	}

	txHash := gjson.Get(result.Raw, "/").String()
	return txHash, nil
}

func (wm *WalletManager) GetEstimateGasPremium(from string, gasLimit *big.Int) (*big.Int, error) {
	blockCids := make([]interface{}, 0)

	params := []interface{}{
		0,
		from,
		gasLimit,
		blockCids,
	}
	result, err := wm.WalletClient.Call("Filecoin.GasEstimateGasPremium", params)
	if err != nil {
		return big.NewInt(0), err
	}
	gasPremium, _ := big.NewInt(0).SetString( result.Str, 10)
	return gasPremium, nil
}

func (wm *WalletManager) GetEstimateGasLimit(msg interface{}) (*big.Int, error) {
	blockCids := make([]interface{}, 0)

	params := []interface{}{
		msg,
		blockCids,
	}
	result, err := wm.WalletClient.Call("Filecoin.GasEstimateGasLimit", params)
	if err != nil {
		return big.NewInt(0), err
	}
	gasLimit, _ := big.NewInt(0).SetString( result.Raw, 10)
	return gasLimit, nil
}

func (wm *WalletManager) GetEstimateFeeCap(msg interface{}) (*big.Int, error) {
	blockCids := make([]interface{}, 0)

	params := []interface{}{
		msg,
		0,
		blockCids,
	}
	result, err := wm.WalletClient.Call("Filecoin.GasEstimateFeeCap", params)
	if err != nil {
		return big.NewInt(0), err
	}
	gasFeeCap, _ := big.NewInt(0).SetString( result.Str, 10)
	return gasFeeCap, nil
}

func (wm *WalletManager) GetMpoolPending() (error) {
	blockCids := make([]interface{}, 0)

	params := []interface{}{
		blockCids,
	}
	result, err := wm.WalletClient.Call("Filecoin.MpoolPending", params)
	if err != nil {
		return err
	}

	for _, signedMessage := range result.Array() {
		message := gjson.Get( signedMessage.Raw, "Message")
		method := gjson.Get( message.Raw, "Method").Int()
		if method==0 {
			fmt.Println( message.String() )
		}
	}

	return nil
}

func (wm *WalletManager) GetTransactionFeeEstimated(from string, to string, value *big.Int, nonce uint64) (*txFeeInfo, error) {
	var (
		gasLimit *big.Int
		gasPrice *big.Int
	)
	gasLimit = wm.Config.FixGasLimit
	gasPrice = wm.Config.FixGasPrice

	msg := map[string]interface{}{
		"to" :  to,
		"From" : from,
		"value" : value.String(),
		//"gasPremium" : gasPremium.String(),
		"nonce" : nonce,
		//"gasLimit" : gasLimit,
		"method" : builtin.MethodSend,
	}

	//----------直接获取----------
	sendSpec := map[string]interface{}{
		"MaxFee" : "0",
	}

	blockCids := make([]interface{}, 0)

	params := []interface{}{
		msg,
		sendSpec,
		blockCids,
	}
	msgInfoJson, err := wm.WalletClient.Call("Filecoin.GasEstimateMessageGas", params)
	if err != nil {
		return nil, err
	}

	gasLimitStr := gjson.Get(msgInfoJson.Raw, "GasLimit").String()
	gasLimit, _ = big.NewInt(0).SetString(gasLimitStr, 10)
	gasLimit = gasLimit.Add( gasLimit, wm.Config.GasLimitAdd )

	gasPremiumStr := gjson.Get(msgInfoJson.Raw, "GasPremium").String()
	gasPremium, _ := big.NewInt(0).SetString(gasPremiumStr, 10)
	gasPremium = gasPremium.Add( gasPremium, wm.Config.GasPremiumAdd )

	gasFeeCapStr := gjson.Get(msgInfoJson.Raw, "GasFeeCap").String()
	gasFeeCap, _ := big.NewInt(0).SetString(gasFeeCapStr, 10)
	gasFeeCap = gasFeeCap.Add( gasFeeCap, wm.Config.GasFeeCapAdd )

	//----------分步获取----------
	//gasLimit, err := wm.GetEstimateGasLimit(msg)
	//if err != nil {
	//	return nil, err
	//}
	//gasLimit = gasLimit.Add( gasLimit, wm.Config.GasLimitAdd )
	//
	//gasPremium, err := wm.GetEstimateGasPremium(from, gasLimit )
	//if err != nil {
	//	return nil, err
	//}
	//gasPremium = gasPremium.Add( gasPremium, wm.Config.GasPremiumAdd )
	//
	//msg["gasPremium"] = gasPremium.String()
	//msg["gasLimit"] = gasLimit
	//
	//gasFeeCap, err := wm.GetEstimateFeeCap(msg)
	//if err != nil {
	//	return nil, err
	//}
	//gasFeeCap = gasFeeCap.Add( gasFeeCap, wm.Config.GasFeeCapAdd )

	mpoolNonce, err := wm.GetMpoolGetNonce( from )
	if err != nil {
		return nil, err
	}
	if mpoolNonce>nonce { //如果现在要打出的交易，nonce小于内存池中的nonce，加大premium50万
		gasPremium = gasPremium.Add( gasPremium, big.NewInt(5000000) )
	}

	feeInfo := &txFeeInfo{
		GasLimit: gasLimit,
		GasPrice: gasPrice,
		GasPremium: gasPremium,
		GasFeeCap: gasFeeCap,
		//		Fee:      fee,
	}

	feeInfo.CalcFee()
	return feeInfo, nil
}

// GetMpoolGetNonce
// {"jsonrpc":"2.0","result":1236,"id":1}
func (wm *WalletManager) GetMpoolGetNonce(address string) (uint64, error) {
	params := []interface{}{
		address,
	}
	result, err := wm.WalletClient.Call("Filecoin.MpoolGetNonce", params)
	if err != nil {
		return 0, err
	}

	nonce := result.Uint()
	if nonce < 0 {
		return 0, errors.New(" wrong nonce ")
	}
	return nonce, nil
}

func GetBlockFromTipSet(tipSet *TipSet) (OwBlock, error){
	block := OwBlock{}
	//block.Hash = string( encodeKey( tipSet.BlkCids ) )
	block.Hash = hex.EncodeToString( encodeKey( tipSet.BlkCids ) )

	parentCids := make([]string, 0)
	parentCidMap := make(map[string]string) //防止重复用的map
	for _, innerBlock := range tipSet.Blks{
		for _, parentCid := range innerBlock.ParentHashs{
			_, ok := parentCidMap[parentCid]
			if !ok { //找不到的时候补充
				parentCids = append(parentCids, parentCid)
				parentCidMap[ parentCid ] = parentCid
			}
		}
		block.Timestamp = innerBlock.Timestamp
	}

	//block.PrevBlockHash = string( encodeKey(parentCids) )
	block.PrevBlockHash = hex.EncodeToString( encodeKey( parentCids ) )
	block.Height = tipSet.Height
	block.TipSet = tipSet

	return block, nil
}

func encodeKey(cids []string) []byte {
	buffer := new(bytes.Buffer)
	for _, c := range cids {
		// bytes.Buffer.Write() err is documented to be always nil.
		_, _ = buffer.Write( []byte(c) )
	}



	newHash := owcrypt.Hash(buffer.Bytes(), 0, owcrypt.HASH_ALG_SHA256)
	return newHash
	//return buffer.Bytes()
}

func CustomAddressEncode(address string) string {
	return address
}
func CustomAddressDecode(address string) string {
	return address
}

// GetAddressNonce
func (wm *WalletManager) GetAddressNonce(wrapper openwallet.WalletDAI, address string, nonce_onchain uint64) uint64 {
	var (
		key           = wm.Symbol() + "-nonce"
		nonce         uint64
		nonce_db      interface{}
	)

	//获取db记录的nonce并确认nonce值
	nonce_db, _ = wrapper.GetAddressExtParam(address, key)

	//判断nonce_db是否为空,为空则说明当前nonce是0
	if nonce_db == nil {
		nonce = 0
	} else {
		//nonce = common.NewString(nonce_db).UInt64()
		nonceStr := common.NewString(nonce_db)
		nonceStrArr := strings.Split(string(nonceStr), "_")

		if len(nonceStrArr)>1 {
			saveTime := common.NewString(nonceStrArr[1]).UInt64()
			now := uint64(time.Now().Unix())
			diff, _ := math.SafeSub(now, saveTime)

			if diff > wm.Config.NonceDiff { //当前时间减去保存时间，超过1小时，就不算了
				nonce = 0
			} else {
				nonce = common.NewString(nonceStrArr[0]).UInt64()
			}
		}else{
			nonce = common.NewString(nonce_db).UInt64()
		}
	}

	wm.Log.Info(address, " get nonce : ", nonce, ", nonce_onchain : ", nonce_onchain)

	//如果本地nonce_db > 链上nonce,采用本地nonce,否则采用链上nonce
	if nonce > nonce_onchain {
		//wm.Log.Debugf("%s nonce_db=%v > nonce_chain=%v,Use nonce_db...", address, nonce_db, nonce_onchain)
	} else {
		nonce = nonce_onchain
		//wm.Log.Debugf("%s nonce_db=%v <= nonce_chain=%v,Use nonce_chain...", address, nonce_db, nonce_onchain)
	}

	return nonce
}

// UpdateAddressNonce
func (wm *WalletManager) UpdateAddressNonce(wrapper openwallet.WalletDAI, address string, nonce uint64) {
	key := wm.Symbol() + "-nonce"

	nonceStr := strconv.FormatUint(nonce, 10) + "_" + strconv.FormatUint( uint64(time.Now().Unix()), 10)
	wm.Log.Info(address, " set nonce ", nonceStr)

	err := wrapper.SetAddressExtParam(address, key, nonceStr)
	if err != nil {
		wm.Log.Errorf("WalletDAI SetAddressExtParam failed, err: %v", err)
	}
}