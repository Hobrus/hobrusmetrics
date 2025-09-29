package sender

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"

	grpcapi "github.com/Hobrus/hobrusmetrics.git/internal/app/grpcapi"
)

// GRPCSender отправляет метрики по gRPC
type GRPCSender struct {
	Address string
	Key     string
	UseTLS  bool
	CAFile  string

	conn   *grpc.ClientConn
	client grpcapi.MetricsServiceClient
}

// compute computes HMAC-SHA256 hex
func compute(data []byte, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

func (g *GRPCSender) dial() error {
	if g.conn != nil {
		return nil
	}
	var opts []grpc.DialOption
	if g.UseTLS {
		// If CAFile is provided, load it; otherwise use system roots
		var creds credentials.TransportCredentials
		var err error
		if g.CAFile != "" {
			creds, err = credentials.NewClientTLSFromFile(g.CAFile, "")
			if err != nil {
				return err
			}
		} else {
			creds = credentials.NewTLS(nil)
		}
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}
	// enable gzip compression on client
	opts = append(opts, grpc.WithDefaultCallOptions(grpc.UseCompressor(gzip.Name)))
	// reasonable timeouts via context on calls
	conn, err := grpc.NewClient(g.Address, opts...)
	if err != nil {
		return err
	}
	g.conn = conn
	g.client = grpcapi.NewMetricsServiceClient(conn)
	return nil
}

func (g *GRPCSender) Close() {
	if g.conn != nil {
		_ = g.conn.Close()
	}
}

// SendBatchGRPC отправляет набор метрик одним RPC
func (g *GRPCSender) SendBatchGRPC(metrics map[string]interface{}) {
	if len(metrics) == 0 {
		return
	}
	if err := g.dial(); err != nil {
		return
	}

    req := &grpcapi.BatchUpdateRequest{Metrics: make([]*grpcapi.Metric, 0, len(metrics))}
	for name, val := range metrics {
		switch v := val.(type) {
		case int64:
			d := v
            tt := grpcapi.MetricType_COUNTER
            req.Metrics = append(req.Metrics, &grpcapi.Metric{Id: proto.String(name), Type: &tt, MetricValue: &grpcapi.Metric_Delta{Delta: d}})
		case float64:
			f := v
            tt := grpcapi.MetricType_GAUGE
            req.Metrics = append(req.Metrics, &grpcapi.Metric{Id: proto.String(name), Type: &tt, MetricValue: &grpcapi.Metric_Value{Value: f}})
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// build metadata with x-real-ip if available, and optional signature
	md := metadata.MD{}
	if ip := detectRealIP(); ip != "" {
		md.Set("x-real-ip", ip)
	}
	// compute signature using protobuf Marshal
	if g.Key != "" && g.Key != "none" {
		if b, err := proto.Marshal(req); err == nil {
			md.Set("hashsha256", compute(b, g.Key))
		}
	}
	ctx = metadata.NewOutgoingContext(ctx, md)

	_, _ = g.client.BatchUpdate(ctx, req)
}
