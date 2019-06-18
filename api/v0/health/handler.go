package health

import (
	"context"

	"github.com/oasislabs/developer-gateway/rpc"
	"github.com/oasislabs/developer-gateway/stats"
)

type StatsProvider interface {
	Stats() stats.Group
}

type Deps struct {
	StatsProvider StatsProvider
}

type HealthHandler struct {
	stats StatsProvider
}

func NewHealthHandler(deps *Deps) HealthHandler {
	return HealthHandler{stats: deps.StatsProvider}
}

func (h HealthHandler) GetHealth(ctx context.Context, v interface{}) (interface{}, error) {
	_ = v.(*GetHealthRequest)
	return &GetHealthResponse{
		Health:  stats.Healthy,
		Metrics: h.stats.Stats(),
	}, nil
}

func BindHandler(deps *Deps, binder rpc.HandlerBinder) {
	handler := NewHealthHandler(deps)

	binder.Bind("GET", "/v0/api/health", rpc.HandlerFunc(handler.GetHealth),
		rpc.EntityFactoryFunc(func() interface{} { return &GetHealthRequest{} }))
}
