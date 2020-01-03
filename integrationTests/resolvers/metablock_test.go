package resolvers

import (
	"bytes"
	"testing"

	"github.com/ElrondNetwork/elrond-go/core"

	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/dataRetriever/dataPool"
	"github.com/ElrondNetwork/elrond-go/integrationTests"
	"github.com/ElrondNetwork/elrond-go/process/factory"
	"github.com/stretchr/testify/assert"
)

//------- Request resolve by hash

func TestRequestResolveMetaHeadersByHashRequestingShardResolvingShard(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	rm := newReceiverMonitor(t)
	shardId := uint32(0)
	nResolver, nRequester := createResolverRequester(shardId, shardId)
	headerNonce := uint64(0)
	header, hash := createMetaHeader(headerNonce, integrationTests.IntegrationTestsChainID)

	//add header with nonce 0 in pool
	_, _ = nResolver.ShardDataPool.MetaBlocks().HasOrAdd(hash, header)

	//setup header received event
	nRequester.ShardDataPool.MetaBlocks().RegisterHandler(
		func(key []byte) {
			if bytes.Equal(key, hash) {
				log.Info("received meta header", "hash", key)
				rm.done()
			}
		},
	)

	//request by hash should work
	resolver, err := nRequester.ResolverFinder.MetaChainResolver(factory.MetachainBlocksTopic)
	log.LogIfError(err)
	err = resolver.RequestDataFromHash(hash)
	log.LogIfError(err)

	rm.waitWithTimeout()
}

func TestRequestResolveMetaHeadersByHashRequestingMetaResolvingShard(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	rm := newReceiverMonitor(t)
	shardId := uint32(0)
	nResolver, nRequester := createResolverRequester(shardId, core.MetachainShardId)
	headerNonce := uint64(0)
	header, hash := createMetaHeader(headerNonce, integrationTests.IntegrationTestsChainID)

	//add header with nonce 0 in pool
	_, _ = nResolver.ShardDataPool.MetaBlocks().HasOrAdd(hash, header)

	//setup header received event
	nRequester.MetaDataPool.MetaBlocks().RegisterHandler(
		func(key []byte) {
			if bytes.Equal(key, hash) {
				log.Info("received meta header", "hash", key)
				rm.done()
			}
		},
	)

	//request by hash should work
	resolver, err := nRequester.ResolverFinder.MetaChainResolver(factory.MetachainBlocksTopic)
	log.LogIfError(err)
	err = resolver.RequestDataFromHash(hash)
	log.LogIfError(err)

	rm.waitWithTimeout()
}

func TestRequestResolveMetaHeadersByHashRequestingShardResolvingMeta(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	rm := newReceiverMonitor(t)
	shardId := uint32(0)
	nResolver, nRequester := createResolverRequester(core.MetachainShardId, shardId)
	headerNonce := uint64(0)
	header, hash := createMetaHeader(headerNonce, integrationTests.IntegrationTestsChainID)

	//add header with nonce 0 in pool
	_, _ = nResolver.MetaDataPool.MetaBlocks().HasOrAdd(hash, header)

	//setup header received event
	nRequester.ShardDataPool.MetaBlocks().RegisterHandler(
		func(key []byte) {
			if bytes.Equal(key, hash) {
				log.Info("received meta header", "hash", key)
				rm.done()
			}
		},
	)

	//request by hash should work
	resolver, err := nRequester.ResolverFinder.MetaChainResolver(factory.MetachainBlocksTopic)
	log.LogIfError(err)
	err = resolver.RequestDataFromHash(hash)
	log.LogIfError(err)

	rm.waitWithTimeout()
}

//------- Request resolve by nonce

func TestRequestResolveMetaHeadersByNonceRequestingShardResolvingShard(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	rm := newReceiverMonitor(t)
	shardId := uint32(0)
	nResolver, nRequester := createResolverRequester(shardId, shardId)
	headerNonce := uint64(0)
	header, hash := createShardHeader(headerNonce, integrationTests.IntegrationTestsChainID)

	//add header with nonce 0 in pool
	_, _ = nResolver.ShardDataPool.MetaBlocks().HasOrAdd(hash, header)
	syncMap := &dataPool.ShardIdHashSyncMap{}
	syncMap.Store(core.MetachainShardId, hash)
	nResolver.ShardDataPool.HeadersNonces().Merge(headerNonce, syncMap)

	//setup header received event
	nRequester.ShardDataPool.MetaBlocks().RegisterHandler(
		func(key []byte) {
			if bytes.Equal(key, hash) {
				log.Info("received header", "hash", key)
				rm.done()
			}
		},
	)

	//request by hash should work
	resolver, err := nRequester.ResolverFinder.MetaChainResolver(factory.MetachainBlocksTopic)
	log.LogIfError(err)
	headerResolver, ok := resolver.(dataRetriever.HeaderResolver)
	assert.True(t, ok)
	err = headerResolver.RequestDataFromNonce(headerNonce)
	log.LogIfError(err)

	rm.waitWithTimeout()
}

func TestRequestResolveMetaHeadersByNonceRequestingMetaResolvingShard(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	rm := newReceiverMonitor(t)
	shardId := uint32(0)
	nResolver, nRequester := createResolverRequester(shardId, core.MetachainShardId)
	headerNonce := uint64(0)
	header, hash := createShardHeader(headerNonce, integrationTests.IntegrationTestsChainID)

	//add header with nonce 0 in pool
	_, _ = nResolver.ShardDataPool.MetaBlocks().HasOrAdd(hash, header)
	syncMap := &dataPool.ShardIdHashSyncMap{}
	syncMap.Store(core.MetachainShardId, hash)
	nResolver.ShardDataPool.HeadersNonces().Merge(headerNonce, syncMap)

	//setup header received event
	nRequester.MetaDataPool.MetaBlocks().RegisterHandler(
		func(key []byte) {
			if bytes.Equal(key, hash) {
				log.Info("received header", "hash", key)
				rm.done()
			}
		},
	)

	//request by hash should work
	resolver, err := nRequester.ResolverFinder.MetaChainResolver(factory.MetachainBlocksTopic)
	log.LogIfError(err)
	headerResolver, ok := resolver.(dataRetriever.HeaderResolver)
	assert.True(t, ok)
	err = headerResolver.RequestDataFromNonce(headerNonce)
	log.LogIfError(err)

	rm.waitWithTimeout()
}

func TestRequestResolveMetaHeadersByNonceRequestingShardResolvingMeta(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	rm := newReceiverMonitor(t)
	shardId := uint32(0)
	nResolver, nRequester := createResolverRequester(core.MetachainShardId, shardId)
	headerNonce := uint64(0)
	header, hash := createShardHeader(headerNonce, integrationTests.IntegrationTestsChainID)

	//add header with nonce 0 in pool
	_, _ = nResolver.MetaDataPool.MetaBlocks().HasOrAdd(hash, header)
	syncMap := &dataPool.ShardIdHashSyncMap{}
	syncMap.Store(core.MetachainShardId, hash)
	nResolver.MetaDataPool.HeadersNonces().Merge(headerNonce, syncMap)

	//setup header received event
	nRequester.ShardDataPool.MetaBlocks().RegisterHandler(
		func(key []byte) {
			if bytes.Equal(key, hash) {
				log.Info("received header", "hash", key)
				rm.done()
			}
		},
	)

	//request by hash should work
	resolver, err := nRequester.ResolverFinder.MetaChainResolver(factory.MetachainBlocksTopic)
	log.LogIfError(err)
	headerResolver, ok := resolver.(dataRetriever.HeaderResolver)
	assert.True(t, ok)
	err = headerResolver.RequestDataFromNonce(headerNonce)
	log.LogIfError(err)

	rm.waitWithTimeout()
}