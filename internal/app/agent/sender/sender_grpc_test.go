package sender

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	grpcapi "github.com/Hobrus/hobrusmetrics.git/internal/app/grpcapi"
	"github.com/Hobrus/hobrusmetrics.git/internal/app/server/grpcserver"
	"github.com/Hobrus/hobrusmetrics.git/internal/app/server/repository"
	"github.com/Hobrus/hobrusmetrics.git/internal/app/server/service"
)

// start real TCP gRPC server on ephemeral port for integration-like test
func startTestGRPCServer(t *testing.T) (addr string, stop func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	s := grpc.NewServer()
	ms := &service.MetricsService{Storage: repository.NewMemStorage()}
	grpcapi.RegisterMetricsServiceServer(s, grpcserver.NewMetricsServer(ms))

	go func() { _ = s.Serve(ln) }()
	return ln.Addr().String(), func() { s.GracefulStop(); _ = ln.Close() }
}

func TestGRPCSender_SendBatchGRPC(t *testing.T) {
	addr, stop := startTestGRPCServer(t)
	defer stop()

	g := &GRPCSender{Address: addr, Key: ""}
	defer g.Close()

	metrics := map[string]interface{}{
		"c1": int64(5),
		"g1": float64(1.23),
	}
	g.SendBatchGRPC(metrics)

	// small wait to ensure RPC processed
	time.Sleep(50 * time.Millisecond)

	// verify through client that values exist
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()
	client := grpcapi.NewMetricsServiceClient(conn)

    val, err := client.GetValue(context.Background(), &grpcapi.GetValueRequest{Id: func() *string { s := "c1"; return &s }(), Type: func() *grpcapi.MetricType { v := grpcapi.MetricType_COUNTER; return &v }()})
	require.NoError(t, err)
	require.Equal(t, int64(5), val.Metric.GetDelta())
}
