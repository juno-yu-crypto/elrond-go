package interceptors

import (
	"sync"

	logger "github.com/ElrondNetwork/elrond-go-logger"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data/batch"
	"github.com/ElrondNetwork/elrond-go/debug/resolver"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/p2p"
	"github.com/ElrondNetwork/elrond-go/process"
)

var log = logger.GetOrCreate("process/interceptors")

// MultiDataInterceptor is used for intercepting packed multi data
type MultiDataInterceptor struct {
	topic                      string
	marshalizer                marshal.Marshalizer
	factory                    process.InterceptedDataFactory
	processor                  process.InterceptorProcessor
	throttler                  process.InterceptorThrottler
	whiteListRequest           process.WhiteListHandler
	antifloodHandler           process.P2PAntifloodHandler
	mutInterceptedDebugHandler sync.RWMutex
	interceptedDebugHandler    process.InterceptedDebugger
}

// NewMultiDataInterceptor hooks a new interceptor for packed multi data
func NewMultiDataInterceptor(
	topic string,
	marshalizer marshal.Marshalizer,
	factory process.InterceptedDataFactory,
	processor process.InterceptorProcessor,
	throttler process.InterceptorThrottler,
	antifloodHandler process.P2PAntifloodHandler,
	whiteListRequest process.WhiteListHandler,
) (*MultiDataInterceptor, error) {
	if len(topic) == 0 {
		return nil, process.ErrEmptyTopic
	}
	if check.IfNil(marshalizer) {
		return nil, process.ErrNilMarshalizer
	}
	if check.IfNil(factory) {
		return nil, process.ErrNilInterceptedDataFactory
	}
	if check.IfNil(processor) {
		return nil, process.ErrNilInterceptedDataProcessor
	}
	if check.IfNil(throttler) {
		return nil, process.ErrNilInterceptorThrottler
	}
	if check.IfNil(antifloodHandler) {
		return nil, process.ErrNilAntifloodHandler
	}
	if check.IfNil(whiteListRequest) {
		return nil, process.ErrNilWhiteListHandler
	}

	multiDataIntercept := &MultiDataInterceptor{
		topic:            topic,
		marshalizer:      marshalizer,
		factory:          factory,
		processor:        processor,
		throttler:        throttler,
		whiteListRequest: whiteListRequest,
		antifloodHandler: antifloodHandler,
	}
	multiDataIntercept.interceptedDebugHandler = resolver.NewDisabledInterceptorResolver()

	return multiDataIntercept, nil
}

// ProcessReceivedMessage is the callback func from the p2p.Messenger and will be called each time a new message was received
// (for the topic this validator was registered to)
func (mdi *MultiDataInterceptor) ProcessReceivedMessage(message p2p.MessageP2P, fromConnectedPeer core.PeerID) error {
	err := preProcessMesage(mdi.throttler, mdi.antifloodHandler, message, fromConnectedPeer, mdi.topic)
	if err != nil {
		return err
	}

	b := batch.Batch{}
	err = mdi.marshalizer.Unmarshal(&b, message.Data())
	if err != nil {
		mdi.throttler.EndProcessing()

		//this situation is so severe that we need to black list de peers
		reason := "unmarshalable data got on topic " + mdi.topic
		mdi.antifloodHandler.BlacklistPeer(message.Peer(), reason, core.InvalidMessageBlacklistDuration)
		mdi.antifloodHandler.BlacklistPeer(fromConnectedPeer, reason, core.InvalidMessageBlacklistDuration)

		return err
	}

	multiDataBuff := b.Data
	lenMultiData := len(multiDataBuff)
	if lenMultiData == 0 {
		mdi.throttler.EndProcessing()
		return process.ErrNoDataInMessage
	}

	err = mdi.antifloodHandler.CanProcessMessagesOnTopic(
		fromConnectedPeer,
		mdi.topic,
		uint32(lenMultiData),
		uint64(len(message.Data())),
		message.SeqNo(),
	)
	if err != nil {
		return err
	}

	lastErrEncountered := error(nil)

	for _, dataBuff := range multiDataBuff {
		var interceptedData process.InterceptedData
		interceptedData, err = mdi.interceptedData(dataBuff, message.Peer(), fromConnectedPeer)
		if err != nil {
			lastErrEncountered = err
			continue
		}

		isForCurrentShard := interceptedData.IsForCurrentShard()
		isWhiteListed := mdi.whiteListRequest.IsWhiteListed(interceptedData)
		shouldProcess := isForCurrentShard || isWhiteListed
		if !shouldProcess {
			log.Trace("intercepted data should not be processed",
				"pid", p2p.MessageOriginatorPid(message),
				"seq no", p2p.MessageOriginatorSeq(message),
				"topics", message.Topics(),
				"hash", interceptedData.Hash(),
				"is for this shard", isForCurrentShard,
				"is white listed", isWhiteListed,
			)
			continue
		}

		err = processInterceptedData(
			mdi.processor,
			mdi.interceptedDebugHandler,
			interceptedData,
			mdi.topic,
			message,
		)
		if err != nil {
			lastErrEncountered = err
		}
	}

	mdi.throttler.EndProcessing()

	return lastErrEncountered
}

func (mdi *MultiDataInterceptor) interceptedData(dataBuff []byte, originator core.PeerID, fromConnectedPeer core.PeerID) (process.InterceptedData, error) {
	interceptedData, err := mdi.factory.Create(dataBuff)
	if err != nil {
		//this situation is so severe that we need to black list de peers
		reason := "can not create object from received bytes, topic " + mdi.topic
		mdi.antifloodHandler.BlacklistPeer(originator, reason, core.InvalidMessageBlacklistDuration)
		mdi.antifloodHandler.BlacklistPeer(fromConnectedPeer, reason, core.InvalidMessageBlacklistDuration)

		return nil, err
	}

	receivedDebugInterceptedData(mdi.interceptedDebugHandler, interceptedData, mdi.topic)

	err = interceptedData.CheckValidity()
	if err != nil {
		processDebugInterceptedData(mdi.interceptedDebugHandler, interceptedData, mdi.topic, err)
		return nil, err
	}

	return interceptedData, nil
}

// SetInterceptedDebugHandler will set a new intercepted debug handler
func (mdi *MultiDataInterceptor) SetInterceptedDebugHandler(handler process.InterceptedDebugger) error {
	if check.IfNil(handler) {
		return process.ErrNilDebugger
	}

	mdi.mutInterceptedDebugHandler.Lock()
	mdi.interceptedDebugHandler = handler
	mdi.mutInterceptedDebugHandler.Unlock()

	return nil
}

// IsInterfaceNil returns true if there is no value under the interface
func (mdi *MultiDataInterceptor) IsInterfaceNil() bool {
	return mdi == nil
}
