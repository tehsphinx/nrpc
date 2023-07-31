package nrpc

import (
	"context"

	"google.golang.org/grpc/stats"
)

var _ stats.Handler = (*noopStatsHandler)(nil)

type noopStatsHandler struct{}

func (n noopStatsHandler) TagRPC(ctx context.Context, _ *stats.RPCTagInfo) context.Context {
	return ctx
}

func (n noopStatsHandler) HandleRPC(context.Context, stats.RPCStats) {}

func (n noopStatsHandler) TagConn(ctx context.Context, _ *stats.ConnTagInfo) context.Context {
	return ctx
}

func (n noopStatsHandler) HandleConn(context.Context, stats.ConnStats) {}
