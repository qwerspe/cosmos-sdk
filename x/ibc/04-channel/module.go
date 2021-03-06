package channel

import (
	"github.com/gorilla/mux"
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/x/ibc/04-channel/client/cli"
	"github.com/cosmos/cosmos-sdk/x/ibc/04-channel/client/rest"
	"github.com/cosmos/cosmos-sdk/x/ibc/04-channel/types"
)

// Name returns the IBC connection ICS name
func Name() string {
	return types.SubModuleName
}

// GetTxCmd returns the root tx command for the IBC connections.
func GetTxCmd(clientCtx client.Context) *cobra.Command {
	return cli.NewTxCmd(clientCtx)
}

// GetQueryCmd returns no root query command for the IBC connections.
func GetQueryCmd(clientCtx client.Context) *cobra.Command {
	return cli.GetQueryCmd(clientCtx)
}

// RegisterRESTRoutes registers the REST routes for the IBC channel
func RegisterRESTRoutes(clientCtx client.Context, rtr *mux.Router) {
	rest.RegisterRoutes(clientCtx, rtr)
}
