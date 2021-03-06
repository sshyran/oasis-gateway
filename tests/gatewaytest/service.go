package gatewaytest

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"reflect"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/oasislabs/oasis-gateway/auth"
	authcore "github.com/oasislabs/oasis-gateway/auth/core"
	"github.com/oasislabs/oasis-gateway/backend"
	backendcore "github.com/oasislabs/oasis-gateway/backend/core"
	"github.com/oasislabs/oasis-gateway/backend/eth"
	"github.com/oasislabs/oasis-gateway/callback/callbacktest"
	"github.com/oasislabs/oasis-gateway/eth/ethtest"
	"github.com/oasislabs/oasis-gateway/gateway"
	"github.com/oasislabs/oasis-gateway/mqueue"
	"github.com/oasislabs/oasis-gateway/rpc"
	"github.com/oasislabs/oasis-gateway/tx"
	"github.com/stretchr/testify/mock"
)

func NewPublicRouter(config *gateway.Config, provider *Provider) *rpc.HttpRouter {
	request := provider.MustGet(reflect.TypeOf(&backendcore.RequestManager{})).(*backendcore.RequestManager)
	authenticator := provider.MustGet(reflect.TypeOf((*authcore.Auth)(nil)).Elem()).(authcore.Auth)

	return gateway.NewPublicRouter(config, &gateway.ServiceGroup{
		Request:       request,
		Authenticator: authenticator,
	})
}

func NewServices(ctx context.Context, config *gateway.Config) (*Provider, error) {
	provider := Provider{}

	// start by adding the mocks
	ethclient := &ethtest.MockClient{}
	provider.MustAdd(ethclient)
	callbackclient := &callbacktest.MockClient{}
	provider.MustAdd(callbackclient)

	callbacktest.ImplementMock(callbackclient)
	ethclient.On("BalanceAt", mock.Anything, mock.Anything, mock.Anything).
		Return(big.NewInt(1), nil)

	ethclient.On("NonceAt", mock.Anything, mock.Anything).Return(uint64(0), nil)

	mqueue, err := mqueue.NewMailbox(ctx, mqueue.Services{Logger: gateway.RootLogger}, &config.MailboxConfig)
	if err != nil {
		return nil, err
	}
	provider.MustAdd(mqueue)

	var privateKeys []*ecdsa.PrivateKey
	for _, key := range config.BackendConfig.BackendConfig.(*backend.EthereumConfig).WalletConfig.PrivateKeys {
		privateKey, err := crypto.HexToECDSA(key)
		if err != nil {
			return nil, fmt.Errorf("failed to read private key with error %s", err.Error())
		}
		privateKeys = append(privateKeys, privateKey)
	}

	executor, err := tx.NewExecutor(ctx, &tx.ExecutorServices{
		Logger:    gateway.RootLogger,
		Client:    ethclient,
		Callbacks: callbackclient,
	}, &tx.ExecutorProps{
		PrivateKeys: privateKeys,
	})
	if err != nil {
		return nil, err
	}
	provider.MustAdd(executor)

	backendclient, err := backend.NewEthClientWithDeps(ctx, &eth.ClientDeps{
		Logger:   gateway.RootLogger,
		Client:   ethclient,
		Executor: executor,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize eth client with error %s", err.Error())
	}
	provider.MustAdd(backendclient)

	request, err := backend.NewRequestManagerWithDeps(ctx, &backend.Deps{
		Logger: gateway.RootLogger,
		MQueue: mqueue,
		Client: backendclient,
	})
	if err != nil {
		return nil, err
	}
	provider.MustAdd(request)

	authenticator, err := auth.NewAuth(&config.AuthConfig)
	if err != nil {
		return nil, err
	}
	provider.MustAdd(authenticator)

	return &provider, nil
}
