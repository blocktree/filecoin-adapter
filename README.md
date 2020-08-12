# filecoin-adapter

本项目适配了openwallet.AssetsAdapter接口，给应用提供了底层的区块链协议支持。

## 项目依赖库

- [go-owcrypt](https://github.com/blocktree/go-owcrypt.git)
- [go-owcdrivers](https://github.com/blocktree/.git)

## 项目信息
- 官网 : https://filecoin.io/
- 区块浏览器 : https://filfox.info/zh/
- 全节点 : lotus( https://github.com/filecoin-project/lotus )
- 全节点rpc接口 : https://github.com/filecoin-project/lotus/blob/master/api/api_full.go
- rpc权限 : 生成token ./lotus auth create-token --perm admin
- 获取测试币水龙头 : https://faucet.testnet.filecoin.io/send?address=hahaha

## 如何测试

openwtester包下的测试用例已经集成了openwallet钱包体系，创建conf文件，新建FIL.ini文件，编辑如下内容：

```ini
#wallet api url
ServerAPI = "http://xxx.xxx.xxx.xxx:xxxxx/rpc/v0"

# is testnet
isTestNet = true

# fix gas limit
fixGasLimit = "1000000"

# fix gas price
fixGasPrice = "1"

# accessToken
accessToken = "xxxxx"

# symbol name
symbol = "TESTFIL"

# decimal
decimal = 18

# Cache data file directory, default = "", current directory: ./data
dataDir = ""
```
