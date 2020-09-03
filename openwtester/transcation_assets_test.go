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

package openwtester

import (
	"testing"
	"time"

	"github.com/blocktree/openwallet/v2/log"
	"github.com/blocktree/openwallet/v2/openw"
	"github.com/blocktree/openwallet/v2/openwallet"
)

func testGetAssetsAccountBalance(tm *openw.WalletManager, walletID, accountID string) {
	balance, err := tm.GetAssetsAccountBalance(testApp, walletID, accountID)
	if err != nil {
		log.Error("GetAssetsAccountBalance failed, unexpected error:", err)
		return
	}
	log.Info("balance:", balance)
}

func testGetAssetsAccountTokenBalance(tm *openw.WalletManager, walletID, accountID string, contract openwallet.SmartContract) {
	balance, err := tm.GetAssetsAccountTokenBalance(testApp, walletID, accountID, contract)
	if err != nil {
		log.Error("GetAssetsAccountTokenBalance failed, unexpected error:", err)
		return
	}
	log.Info("token balance:", balance.Balance)
}

func testCreateTransactionStep(tm *openw.WalletManager, walletID, accountID, to, amount, feeRate string, contract *openwallet.SmartContract) (*openwallet.RawTransaction, error) {

	//err := tm.RefreshAssetsAccountBalance(testApp, accountID)
	//if err != nil {
	//	log.Error("RefreshAssetsAccountBalance failed, unexpected error:", err)
	//	return nil, err
	//}

	rawTx, err := tm.CreateTransaction(testApp, walletID, accountID, amount, to, feeRate, "test", contract)

	if err != nil {
		log.Error("CreateTransaction failed, unexpected error:", err)
		return nil, err
	}

	return rawTx, nil
}

func testCreateSummaryTransactionStep(
	tm *openw.WalletManager,
	walletID, accountID, summaryAddress, minTransfer, retainedBalance, feeRate string,
	start, limit int,
	contract *openwallet.SmartContract) ([]*openwallet.RawTransactionWithError, error) {

	rawTxArray, err := tm.CreateSummaryRawTransactionWithError(testApp, walletID, accountID, summaryAddress, minTransfer,
		retainedBalance, feeRate, start, limit, contract, nil)

	if err != nil {
		log.Error("CreateSummaryTransaction failed, unexpected error:", err)
		return nil, err
	}

	return rawTxArray, nil
}

func testSignTransactionStep(tm *openw.WalletManager, rawTx *openwallet.RawTransaction) (*openwallet.RawTransaction, error) {

	_, err := tm.SignTransaction(testApp, rawTx.Account.WalletID, rawTx.Account.AccountID, "12345678", rawTx)
	if err != nil {
		log.Error("SignTransaction failed, unexpected error:", err)
		return nil, err
	}

	log.Infof("rawTx: %+v", rawTx)
	return rawTx, nil
}

func testVerifyTransactionStep(tm *openw.WalletManager, rawTx *openwallet.RawTransaction) (*openwallet.RawTransaction, error) {

	//log.Info("rawTx.Signatures:", rawTx.Signatures)

	_, err := tm.VerifyTransaction(testApp, rawTx.Account.WalletID, rawTx.Account.AccountID, rawTx)
	if err != nil {
		log.Error("VerifyTransaction failed, unexpected error:", err)
		return nil, err
	}

	log.Infof("rawTx: %+v", rawTx)
	return rawTx, nil
}

func testSubmitTransactionStep(tm *openw.WalletManager, rawTx *openwallet.RawTransaction) (*openwallet.RawTransaction, error) {

	tx, err := tm.SubmitTransaction(testApp, rawTx.Account.WalletID, rawTx.Account.AccountID, rawTx)
	if err != nil {
		log.Error("SubmitTransaction failed, unexpected error:", err)
		return nil, err
	}

	log.Std.Info("tx: %+v", tx)
	log.Info("wxID:", tx.WxID)
	log.Info("txID:", rawTx.TxID)

	return rawTx, nil
}

func ClearAddressNonce(tm *openw.WalletManager, walletID string, accountID string) error{
	wrapper, err := tm.NewWalletWrapper(testApp, "")
	if err != nil {
		return err
	}

	list, err := tm.GetAddressList(testApp, walletID, accountID, 0, -1, false)
	if err != nil {
		log.Error("unexpected error:", err)
		return err
	}
	for i, w := range list {
		log.Info("address[", i, "] :", w.Address)

		key := "TESTFIL-nonce"
		wrapper.SetAddressExtParam(w.Address, key, 0)
	}
	log.Info("address count:", len(list))

	tm.CloseDB(testApp)

	return nil
}

/*
withdraw
wallet : Vza1Xa1BayaJF8UcMCm1Zad5zFtrVaLDvW
account : 67J2EMuyPEBUMbQ1BPwMdBcDLWXGAuVmz2RGpFatCZKK
1 address : t1jqjwa4vqv374gitefzyw6lst5lokuwdxf62fopi
*/
func TestTransfer(t *testing.T) {
	tm := testInitWalletManager()
	walletID := "Vza1Xa1BayaJF8UcMCm1Zad5zFtrVaLDvW"
	accountID := "67J2EMuyPEBUMbQ1BPwMdBcDLWXGAuVmz2RGpFatCZKK"
	to := "t1wh2fhzvb5rcfoleedkupov442qp4hw34kzm52ki"

	testGetAssetsAccountBalance(tm, walletID, accountID)

	rawTx, err := testCreateTransactionStep(tm, walletID, accountID, to, "1.6235", "", nil)
	if err != nil {
		return
	}

	log.Std.Info("rawTx: %+v", rawTx)

	_, err = testSignTransactionStep(tm, rawTx)
	if err != nil {
		return
	}

	_, err = testVerifyTransactionStep(tm, rawTx)
	if err != nil {
		return
	}

	_, err = testSubmitTransactionStep(tm, rawTx)
	if err != nil {
		return
	}
}

/*
withdraw
wallet : Vza1Xa1BayaJF8UcMCm1Zad5zFtrVaLDvW
account : 67J2EMuyPEBUMbQ1BPwMdBcDLWXGAuVmz2RGpFatCZKK
1 address : t1jqjwa4vqv374gitefzyw6lst5lokuwdxf62fopi
---------------
charge
wallet : W5Gfeiz9NB9HVGqMpNqWDZMm1XFAQv2TRT
account : CCcjBvcwvh2eiL5Bpt5i4y2McYwcfVrZntSa6bgRktUw
1 address : t14inb7dsbir2n25a6yg6dmu6q4tdc4tltokddtaa
2 address : t153wzdi5rb3tznwty5rzumil3e4zxc2sdoozbhui
3 address : t15chpd4bqtgwrlkdequlcxb4zssqcquejonbjtcy
4 address : t1cmtgyrmhh6at3ki2ucak3chedzg3s43aag2z3cq
5 address : t1dh5jwrbsle3asepu4fpnxlgyh6l66pmh3ypabhi
6 address : t1nyte6aj5r2lbyf5alciih743wl3ccueboiodj7a
7 address : t1qye5h3lydyzx4cuwjtr6oe3vqubo7zfjcqziwqa
*/
func TestBatchTransfer(t *testing.T) {
	tm := testInitWalletManager()
	walletID := "Vza1Xa1BayaJF8UcMCm1Zad5zFtrVaLDvW"
	accountID := "67J2EMuyPEBUMbQ1BPwMdBcDLWXGAuVmz2RGpFatCZKK"
	toArr := make([]string, 0)
	toArr = append(toArr, "t14inb7dsbir2n25a6yg6dmu6q4tdc4tltokddtaa")
	toArr = append(toArr, "t153wzdi5rb3tznwty5rzumil3e4zxc2sdoozbhui")
	toArr = append(toArr, "t15chpd4bqtgwrlkdequlcxb4zssqcquejonbjtcy")
	toArr = append(toArr, "t1cmtgyrmhh6at3ki2ucak3chedzg3s43aag2z3cq")
	toArr = append(toArr, "t1dh5jwrbsle3asepu4fpnxlgyh6l66pmh3ypabhi")
	toArr = append(toArr, "t1nyte6aj5r2lbyf5alciih743wl3ccueboiodj7a")

	for i := 0; i < len(toArr); i++{
		to := toArr[i]
		testGetAssetsAccountBalance(tm, walletID, accountID)

		rawTx, err := testCreateTransactionStep(tm, walletID, accountID, to, "1.23", "", nil)
		if err != nil {
			return
		}

		log.Std.Info("rawTx: %+v", rawTx)

		_, err = testSignTransactionStep(tm, rawTx)
		if err != nil {
			return
		}

		_, err = testVerifyTransactionStep(tm, rawTx)
		if err != nil {
			return
		}

		_, err = testSubmitTransactionStep(tm, rawTx)
		if err != nil {
			return
		}

		time.Sleep(time.Duration(5) * time.Second)
	}
}

func TestSummary(t *testing.T) {
	tm := testInitWalletManager()

	walletID := "W5Gfeiz9NB9HVGqMpNqWDZMm1XFAQv2TRT"
	accountID := "CCcjBvcwvh2eiL5Bpt5i4y2McYwcfVrZntSa6bgRktUw"
	summaryAddress := "t1jqjwa4vqv374gitefzyw6lst5lokuwdxf62fopi"

	ClearAddressNonce(tm, walletID, accountID)

	testGetAssetsAccountBalance(tm, walletID, accountID)

	rawTxArray, err := testCreateSummaryTransactionStep(tm, walletID, accountID,
		summaryAddress, "0.000000000000000282", "0.000000000000000282", "",
		0, 100, nil)
	if err != nil {
		log.Errorf("CreateSummaryTransaction failed, unexpected error: %v", err)
		return
	}

	//执行汇总交易
	for _, rawTxWithErr := range rawTxArray {

		if rawTxWithErr.Error != nil {
			log.Error(rawTxWithErr.Error.Error())
			continue
		}

		_, err = testSignTransactionStep(tm, rawTxWithErr.RawTx)
		if err != nil {
			return
		}

		_, err = testVerifyTransactionStep(tm, rawTxWithErr.RawTx)
		if err != nil {
			return
		}

		_, err = testSubmitTransactionStep(tm, rawTxWithErr.RawTx)
		if err != nil {
			return
		}
	}

}
