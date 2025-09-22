package grpcserver

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/require"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"

	grpcapi "github.com/Hobrus/hobrusmetrics.git/internal/app/grpcapi"
	"github.com/Hobrus/hobrusmetrics.git/internal/app/server/repository"
	"github.com/Hobrus/hobrusmetrics.git/internal/app/server/service"
)

const bufSize = 1024 * 1024

func newBufConnServer(t *testing.T, key string) (*grpc.ClientConn, *grpc.Server, func()) {
	t.Helper()
	lis := bufconn.Listen(bufSize)

	ts, err := trustedSubnetInterceptor("")
	require.NoError(t, err)
	s := grpc.NewServer(grpc.ChainUnaryInterceptor(ts, hmacInterceptor(key)))

	// in-memory storage and service
	ms := &service.MetricsService{Storage: repository.NewMemStorage()}
	grpcapi.RegisterMetricsServiceServer(s, NewMetricsServer(ms))

	go func() { _ = s.Serve(lis) }()

	dialer := func(context.Context, string) (net.Conn, error) { return lis.Dial() }
	conn, err := grpc.DialContext(context.Background(), "bufnet",
		grpc.WithContextDialer(dialer),
		grpc.WithInsecure(),
	)
	require.NoError(t, err)

	cleanup := func() {
		_ = conn.Close()
		s.GracefulStop()
		_ = lis.Close()
	}
	return conn, s, cleanup
}

func TestGRPCServer_UpdateAndGet(t *testing.T) {
	key := "testkey"
	conn, _, cleanup := newBufConnServer(t, key)
	defer cleanup()

	client := grpcapi.NewMetricsServiceClient(conn)

	// Update gauge
	ur := &grpcapi.UpdateMetricRequest{Metric: &grpcapi.Metric{Id: "g1", Type: grpcapi.MetricType_GAUGE, MetricValue: &grpcapi.Metric_Value{Value: 1.5}}}
	b, err := proto.Marshal(ur)
	require.NoError(t, err)
	md := metadata.Pairs("hashsha256", computeHMAC(b, key))
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	resp, err := client.UpdateMetric(ctx, ur)
	require.NoError(t, err)
	require.NotNil(t, resp)
	m := resp.Metric
	require.Equal(t, grpcapi.MetricType_GAUGE, m.Type)

	// Get value for gauge
	gr := &grpcapi.GetValueRequest{Id: "g1", Type: grpcapi.MetricType_GAUGE}
	b2, err := proto.Marshal(gr)
	require.NoError(t, err)
	ctx2 := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("hashsha256", computeHMAC(b2, key)))
	gv, err := client.GetValue(ctx2, gr)
	require.NoError(t, err)
	require.Equal(t, "g1", gv.Metric.Id)
	_, ok := gv.Metric.MetricValue.(*grpcapi.Metric_Value)
	require.True(t, ok)

	// Batch update: counter + gauge
	br := &grpcapi.BatchUpdateRequest{Metrics: []*grpcapi.Metric{
		{Id: "c1", Type: grpcapi.MetricType_COUNTER, MetricValue: &grpcapi.Metric_Delta{Delta: 10}},
		{Id: "g2", Type: grpcapi.MetricType_GAUGE, MetricValue: &grpcapi.Metric_Value{Value: 2.25}},
	}}
	b3, err := proto.Marshal(br)
	require.NoError(t, err)
	ctx3 := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("hashsha256", computeHMAC(b3, key)))
	bresp, err := client.BatchUpdate(ctx3, br)
	require.NoError(t, err)
	require.Len(t, bresp.Metrics, 2)
}

func TestGRPCServer_HMACRejected(t *testing.T) {
	key := "testkey"
	conn, _, cleanup := newBufConnServer(t, key)
	defer cleanup()
	client := grpcapi.NewMetricsServiceClient(conn)

	// wrong signature
	ur := &grpcapi.UpdateMetricRequest{Metric: &grpcapi.Metric{Id: "bad", Type: grpcapi.MetricType_COUNTER, MetricValue: &grpcapi.Metric_Delta{Delta: 1}}}
	b, err := proto.Marshal(ur)
	require.NoError(t, err)
	md := metadata.Pairs("hashsha256", computeHMAC(b, "wrong"))
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	_, err = client.UpdateMetric(ctx, ur)
	require.Error(t, err)
}
