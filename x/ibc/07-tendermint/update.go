package tendermint

import (
	"errors"
	"time"

	lite "github.com/tendermint/tendermint/lite2"
	tmtypes "github.com/tendermint/tendermint/types"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	clientexported "github.com/cosmos/cosmos-sdk/x/ibc/02-client/exported"
	clienttypes "github.com/cosmos/cosmos-sdk/x/ibc/02-client/types"
	"github.com/cosmos/cosmos-sdk/x/ibc/07-tendermint/types"
	commitmenttypes "github.com/cosmos/cosmos-sdk/x/ibc/23-commitment/types"
)

// CheckValidityAndUpdateState checks if the provided header is valid and updates
// the consensus state if appropriate. It returns an error if:
// - the client or header provided are not parseable to tendermint types
// - the header is invalid
// - header height is lower than the latest client height
// - header valset commit verification fails
//
// Tendermint client validity checking uses the bisection algorithm described
// in the [Tendermint spec](https://github.com/tendermint/spec/blob/master/spec/consensus/light-client.md).
func CheckValidityAndUpdateState(
	clientState clientexported.ClientState, header clientexported.Header,
	currentTimestamp time.Time,
) (clientexported.ClientState, clientexported.ConsensusState, error) {
	tmClientState, ok := clientState.(types.ClientState)
	if !ok {
		return nil, nil, sdkerrors.Wrap(
			clienttypes.ErrInvalidClientType, "light client is not from Tendermint",
		)
	}

	tmHeader, ok := header.(types.Header)
	if !ok {
		return nil, nil, sdkerrors.Wrap(
			clienttypes.ErrInvalidHeader, "header is not from Tendermint",
		)
	}

	if err := checkValidity(tmClientState, tmHeader, currentTimestamp); err != nil {
		return nil, nil, err
	}

	tmClientState, consensusState := update(tmClientState, tmHeader)
	return tmClientState, consensusState, nil
}

// checkValidity checks if the Tendermint header is valid.
//
// CONTRACT: assumes header.Height > consensusState.Height
func checkValidity(
	clientState types.ClientState, header types.Header, currentTimestamp time.Time,
) error {
	var (
		tmLastSignedHeader, tmSignedHeader *tmtypes.SignedHeader
		tmLastValidatorSet, tmValidatorSet *tmtypes.ValidatorSet
	)

	// assert trusting period has not yet passed
	if currentTimestamp.Sub(clientState.GetLatestTimestamp()) >= clientState.TrustingPeriod {
		return errors.New("trusting period since last client timestamp already passed")
	}

	// assert header timestamp is not past the trusting period
	if header.SignedHeader.Header.GetTime().Sub(clientState.GetLatestTimestamp()) >= clientState.TrustingPeriod {
		return sdkerrors.Wrap(
			clienttypes.ErrInvalidHeader,
			"header blocktime is outside trusting period from last client timestamp",
		)
	}

	// assert header timestamp is past latest clientstate timestamp
	if header.SignedHeader.Header.GetTime().Unix() <= clientState.GetLatestTimestamp().Unix() {
		return sdkerrors.Wrapf(
			clienttypes.ErrInvalidHeader,
			"header blocktime ≤ latest client state block time (%s ≤ %s)",
			header.SignedHeader.Header.GetTime().String(), clientState.GetLatestTimestamp().String(),
		)
	}

	// assert header height is newer than any we know
	if header.GetHeight() <= clientState.GetLatestHeight() {
		return sdkerrors.Wrapf(
			clienttypes.ErrInvalidHeader,
			"header height ≤ latest client state height (%d ≤ %d)", header.GetHeight(), clientState.GetLatestHeight(),
		)
	}

	if err := tmLastSignedHeader.FromProto(clientState.LastHeader.SignedHeader); err != nil {
		return err
	}

	if err := tmLastValidatorSet.FromProto(*clientState.LastHeader.ValidatorSet); err != nil {
		return err
	}

	if err := tmSignedHeader.FromProto(header.SignedHeader); err != nil {
		return err
	}

	if err := tmValidatorSet.FromProto(*header.ValidatorSet); err != nil {
		return err
	}

	// Verify next header with the last header's validatorset as trusted validatorset
	maxClockDrift := 10 * time.Second
	err := lite.Verify(
		clientState.GetChainID(),
		tmLastSignedHeader,
		tmLastValidatorSet,
		tmSignedHeader,
		tmValidatorSet,
		clientState.TrustingPeriod,
		currentTimestamp,
		maxClockDrift,
		lite.DefaultTrustLevel)
	if err != nil {
		return sdkerrors.Wrap(clienttypes.ErrInvalidHeader, err.Error())
	}
	return nil
}

// update the consensus state from a new header
func update(clientState types.ClientState, header types.Header) (types.ClientState, types.ConsensusState) {
	clientState.LastHeader = header
	consensusState := types.ConsensusState{
		Height:       uint64(header.SignedHeader.Header.GetHeight()),
		Timestamp:    header.SignedHeader.Header.GetTime(),
		Root:         commitmenttypes.NewMerkleRoot(header.SignedHeader.Header.GetAppHash()),
		ValidatorSet: header.ValidatorSet,
	}

	return clientState, consensusState
}