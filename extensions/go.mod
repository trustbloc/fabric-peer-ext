module github.com/trustbloc/fabric-peer-ext/extensions

replace github.com/hyperledger/fabric => github.com/trustbloc/fabric-mod v0.0.0-20190508134351-4beae6ee306b

replace github.com/hyperledger/fabric/extensions => ./

require (
	github.com/hyperledger/fabric v0.0.0-20190313191403-aa14c142d8c7
	github.com/spf13/viper v0.0.0-20150908122457-1967d93db724
	github.com/stretchr/testify v1.3.0
)
