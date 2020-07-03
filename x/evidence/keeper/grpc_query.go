package keeper

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	"github.com/cosmos/cosmos-sdk/types/query"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/evidence/exported"
	"github.com/cosmos/cosmos-sdk/x/evidence/types"
	proto "github.com/gogo/protobuf/proto"
)

var _ types.QueryServer = Keeper{}

// Evidence implements the Query/Evidence gRPC method
func (k Keeper) Evidence(c context.Context, req *types.QueryEvidenceRequest) (*types.QueryEvidenceResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}

	if req.EvidenceHash == nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid hash")
	}

	ctx := sdk.UnwrapSDKContext(c)

	evidence, _ := k.GetEvidence(ctx, req.EvidenceHash)
	if evidence == nil {
		return nil, status.Errorf(codes.NotFound, "evidence %s not found", req.EvidenceHash)
	}

	evidenceAny, err := ConvertEvidence(evidence)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return &types.QueryEvidenceResponse{Evidence: evidenceAny}, nil
}

// AllEvidences implements the Query/AllEvidences gRPC method
func (k Keeper) AllEvidences(c context.Context, req *types.QueryAllEvidencesRequest) (*types.QueryAllEvidencesResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}
	ctx := sdk.UnwrapSDKContext(c)

	k.GetAllEvidence(ctx)

	var evidences []*codectypes.Any
	store := ctx.KVStore(k.storeKey)
	evidencesStore := prefix.NewStore(store, types.KeyPrefixEvidence)

	res, err := query.Paginate(evidencesStore, req.Req, func(key []byte, value []byte) error {
		result, err := k.UnmarshalEvidence(value)
		if err != nil {
			return err
		}
		evidenceAny, err := ConvertEvidence(result)
		if err != nil {
			return err
		}
		evidences = append(evidences, evidenceAny)
		return nil
	})

	if err != nil {
		return &types.QueryAllEvidencesResponse{}, err
	}

	return &types.QueryAllEvidencesResponse{Evidences: evidences, Res: res}, nil
}

// ConvertEvidence converts Evidence to Any type
func ConvertEvidence(evidence exported.Evidence) (*codectypes.Any, error) {
	msg, ok := evidence.(proto.Message)
	if !ok {
		return nil, fmt.Errorf("can't protomarshal %T", msg)
	}

	any, err := codectypes.NewAnyWithValue(msg)
	if err != nil {
		return nil, err
	}

	return any, nil
}
