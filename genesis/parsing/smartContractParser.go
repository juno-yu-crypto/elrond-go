package parsing

import (
	"encoding/hex"
	"fmt"
	"os"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/genesis"
	"github.com/ElrondNetwork/elrond-go/genesis/data"
	"github.com/ElrondNetwork/elrond-go/sharding"
)

// smartContractParser hold data for initial smart contracts
type smartContractParser struct {
	initialSmartContracts []*data.InitialSmartContract
	pubkeyConverter       state.PubkeyConverter
	checkForFileHandler   func(filename string) error
}

// NewSmartContractsParser creates a new decoded smart contracts genesis structure from json config file
func NewSmartContractsParser(
	genesisFilePath string,
	pubkeyConverter state.PubkeyConverter,
) (*smartContractParser, error) {

	if check.IfNil(pubkeyConverter) {
		return nil, genesis.ErrNilPubkeyConverter
	}

	initialSmartContracts := make([]*data.InitialSmartContract, 0)
	err := core.LoadJsonFile(&initialSmartContracts, genesisFilePath)
	if err != nil {
		return nil, err
	}

	scp := &smartContractParser{
		initialSmartContracts: initialSmartContracts,
		pubkeyConverter:       pubkeyConverter,
	}
	scp.checkForFileHandler = scp.checkForFile

	err = scp.process()
	if err != nil {
		return nil, err
	}

	return scp, nil
}

func (scp *smartContractParser) process() error {
	for _, initialSmartContract := range scp.initialSmartContracts {
		err := scp.parseElement(initialSmartContract)
		if err != nil {
			return err
		}

		err = scp.checkForFileHandler(initialSmartContract.Filename)
		if err != nil {
			return err
		}
	}

	err := scp.checkForDuplicates()
	if err != nil {
		return err
	}

	return nil
}

func (scp *smartContractParser) parseElement(initialSmartContract *data.InitialSmartContract) error {
	if len(initialSmartContract.Owner) == 0 {
		return genesis.ErrEmptyOwnerAddress
	}
	ownerBytes, err := scp.pubkeyConverter.Decode(initialSmartContract.Owner)
	if err != nil {
		return fmt.Errorf("%w for `%s`",
			genesis.ErrInvalidOwnerAddress, initialSmartContract.Owner)
	}

	initialSmartContract.SetOwnerBytes(ownerBytes)

	if len(initialSmartContract.VmType) == 0 {
		return fmt.Errorf("%w for  %s",
			genesis.ErrEmptyVmType, initialSmartContract.Owner)
	}

	_, err = hex.DecodeString(initialSmartContract.VmType)
	if err != nil {
		return fmt.Errorf("%w for provided %s, error: %s",
			genesis.ErrInvalidVmType, initialSmartContract.VmType, err.Error())
	}

	return nil
}

func (scp *smartContractParser) checkForFile(filename string) error {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return fmt.Errorf("%w for the file %s", err, filename)
	}

	if info.IsDir() {
		return fmt.Errorf("%w for the file %s", genesis.ErrFilenameIsDirectory, filename)
	}

	return nil
}

func (scp *smartContractParser) checkForDuplicates() error {
	for idx1 := 0; idx1 < len(scp.initialSmartContracts); idx1++ {
		ib1 := scp.initialSmartContracts[idx1]
		for idx2 := idx1 + 1; idx2 < len(scp.initialSmartContracts); idx2++ {
			ib2 := scp.initialSmartContracts[idx2]
			if ib1.Owner == ib2.Owner {
				return fmt.Errorf("%w found for '%s'",
					genesis.ErrDuplicateOwnerAddress,
					ib1.Owner)
			}
		}
	}

	return nil
}

// InitialSmartContracts return the initial smart contracts contained by this parser
func (scp *smartContractParser) InitialSmartContracts() []genesis.InitialSmartContractHandler {
	smartContracts := make([]genesis.InitialSmartContractHandler, len(scp.initialSmartContracts))

	for idx, isc := range scp.initialSmartContracts {
		smartContracts[idx] = isc
	}

	return smartContracts
}

// InitialSmartContractsSplitOnOwnersShards returns the initial smart contracts split by the owner shards
func (scp *smartContractParser) InitialSmartContractsSplitOnOwnersShards(
	shardCoordinator sharding.Coordinator,
) (map[uint32][]genesis.InitialSmartContractHandler, error) {

	if check.IfNil(shardCoordinator) {
		return nil, genesis.ErrNilShardCoordinator
	}

	var smartContracts = make(map[uint32][]genesis.InitialSmartContractHandler)
	for _, isc := range scp.initialSmartContracts {
		ownerAddress, err := scp.pubkeyConverter.CreateAddressFromBytes(isc.OwnerBytes())
		if err != nil {
			return nil, err
		}
		shardID := shardCoordinator.ComputeId(ownerAddress)

		smartContracts[shardID] = append(smartContracts[shardID], isc)
	}

	return smartContracts, nil
}

// IsInterfaceNil returns if underlying object is true
func (scp *smartContractParser) IsInterfaceNil() bool {
	return scp == nil
}
