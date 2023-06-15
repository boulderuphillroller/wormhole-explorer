package domain

import (
	"encoding/base32"
	"encoding/hex"
	"fmt"

	"github.com/mr-tron/base58"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

// GetSupportedChainIDs returns a map of all supported chain IDs to their respective names.
func GetSupportedChainIDs() map[vaa.ChainID]string {
	chainIDs := vaa.GetAllNetworkIDs()
	supportedChaindIDs := make(map[vaa.ChainID]string, len(chainIDs))
	for _, chainID := range chainIDs {
		supportedChaindIDs[chainID] = chainID.String()
	}
	return supportedChaindIDs
}

// EncodeTrxHashByChainID encodes the transaction hash by chain id with different encoding methods.
func EncodeTrxHashByChainID(chainID vaa.ChainID, txHash []byte) (string, error) {
	switch chainID {
	case vaa.ChainIDSolana:
		return base58.Encode(txHash), nil
	case vaa.ChainIDEthereum:
		return hex.EncodeToString(txHash), nil
	case vaa.ChainIDTerra:
		return hex.EncodeToString(txHash), nil
	case vaa.ChainIDBSC:
		return hex.EncodeToString(txHash), nil
	case vaa.ChainIDPolygon:
		return hex.EncodeToString(txHash), nil
	case vaa.ChainIDAvalanche:
		return hex.EncodeToString(txHash), nil
	case vaa.ChainIDOasis:
		return hex.EncodeToString(txHash), nil
	case vaa.ChainIDAlgorand:
		return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(txHash), nil
	case vaa.ChainIDAurora:
		return hex.EncodeToString(txHash), nil
	case vaa.ChainIDFantom:
		return hex.EncodeToString(txHash), nil
	case vaa.ChainIDKarura:
		return hex.EncodeToString(txHash), nil
	case vaa.ChainIDAcala:
		return hex.EncodeToString(txHash), nil
	case vaa.ChainIDKlaytn:
		return hex.EncodeToString(txHash), nil
	case vaa.ChainIDCelo:
		return hex.EncodeToString(txHash), nil
	case vaa.ChainIDNear:
		return base58.Encode(txHash), nil
	case vaa.ChainIDMoonbeam:
		return hex.EncodeToString(txHash), nil
	case vaa.ChainIDNeon:
		return hex.EncodeToString(txHash), nil
	case vaa.ChainIDTerra2:
		return hex.EncodeToString(txHash), nil
	case vaa.ChainIDInjective:
		return hex.EncodeToString(txHash), nil
	case vaa.ChainIDSui:
		return base58.Encode(txHash), nil
	case vaa.ChainIDAptos:
		return hex.EncodeToString(txHash), nil
	case vaa.ChainIDArbitrum:
		return hex.EncodeToString(txHash), nil
	case vaa.ChainIDOptimism:
		return hex.EncodeToString(txHash), nil
	case vaa.ChainIDXpla:
		return hex.EncodeToString(txHash), nil
	case vaa.ChainIDBtc:
		//TODO: check if this is correct
		return hex.EncodeToString(txHash), nil
	case vaa.ChainIDBase:
		//TODO: check if this is correct
		return hex.EncodeToString(txHash), nil
	case vaa.ChainIDSei:
		return hex.EncodeToString(txHash), nil
	case vaa.ChainIDWormchain:
		//TODO: check if this is correct
		return hex.EncodeToString(txHash), nil
	case vaa.ChainIDSepolia:
		return hex.EncodeToString(txHash), nil
	default:
		return hex.EncodeToString(txHash), fmt.Errorf("unknown chain id: %d", chainID)
	}
}
