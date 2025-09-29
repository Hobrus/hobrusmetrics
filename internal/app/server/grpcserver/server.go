package grpcserver

import (
    "context"
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "errors"
    "fmt"
    "net"
    "strconv"
    "strings"
    "time"

    "google.golang.org/grpc"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/credentials"
    _ "google.golang.org/grpc/encoding/gzip"
    "google.golang.org/grpc/metadata"
    "google.golang.org/grpc/status"
    "google.golang.org/protobuf/proto"

    grpcapi "github.com/Hobrus/hobrusmetrics.git/internal/app/grpcapi"
    servercfg "github.com/Hobrus/hobrusmetrics.git/internal/app/server/config"
    "github.com/Hobrus/hobrusmetrics.git/internal/app/server/middleware"
    "github.com/Hobrus/hobrusmetrics.git/internal/app/server/service"
)

// computeHMAC calculates HMAC-SHA256 hex over data with key
func computeHMAC(data []byte, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

// hmacInterceptor verifies optional HashSHA256 metadata for requests using proto.Marshal(req)
func hmacInterceptor(key string) grpc.UnaryServerInterceptor {
	if key == "" || key == "none" {
		return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
			return handler(ctx, req)
		}
	}
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if req != nil {
			md, _ := metadata.FromIncomingContext(ctx)
			vals := md.Get("hashsha256")
			if len(vals) > 0 {
				b, err := proto.Marshal(req.(proto.Message))
				if err != nil {
					return nil, status.Errorf(codes.InvalidArgument, "invalid request")
				}
				if computeHMAC(b, key) != strings.TrimSpace(vals[0]) {
					return nil, status.Errorf(codes.InvalidArgument, "invalid signature")
				}
			}
		}
		return handler(ctx, req)
	}
}

type MetricsServer struct {
	grpcapi.UnimplementedMetricsServiceServer
	svc *service.MetricsService
}

func NewMetricsServer(svc *service.MetricsService) *MetricsServer {
	return &MetricsServer{svc: svc}
}

func (s *MetricsServer) UpdateMetric(ctx context.Context, req *grpcapi.UpdateMetricRequest) (*grpcapi.UpdateMetricResponse, error) {
	if req == nil || req.Metric == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}
	m := req.Metric
	var mtype string
	var value string
    switch m.GetType() {
	case grpcapi.MetricType_GAUGE:
		mtype = service.GaugeMetric
		value = fmt.Sprintf("%g", m.GetValue())
	case grpcapi.MetricType_COUNTER:
		mtype = service.CounterMetric
		value = fmt.Sprintf("%d", m.GetDelta())
	default:
		return nil, status.Errorf(codes.InvalidArgument, "unsupported type")
	}
    if err := s.svc.UpdateMetric(mtype, m.GetId(), value); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	// read back actual value
    updated, err := s.svc.GetMetricValue(mtype, m.GetId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch updated value")
	}
    out := &grpcapi.Metric{Id: proto.String(m.GetId())}
    t := m.GetType()
    out.Type = &t
    switch m.GetType() {
	case grpcapi.MetricType_GAUGE:
		// parse to float64
		fv, err := strconv.ParseFloat(strings.TrimSpace(updated), 64)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to parse updated gauge")
		}
		out.MetricValue = &grpcapi.Metric_Value{Value: fv}
	case grpcapi.MetricType_COUNTER:
		iv, err := strconv.ParseInt(strings.TrimSpace(updated), 10, 64)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to parse updated counter")
		}
		out.MetricValue = &grpcapi.Metric_Delta{Delta: iv}
	}
	return &grpcapi.UpdateMetricResponse{Metric: out}, nil
}

func (s *MetricsServer) BatchUpdate(ctx context.Context, req *grpcapi.BatchUpdateRequest) (*grpcapi.BatchUpdateResponse, error) {
	if req == nil || len(req.Metrics) == 0 {
		return &grpcapi.BatchUpdateResponse{}, nil
	}
	// convert to middleware.MetricsJSON slice for service batch API
	batch := make([]serviceMetricJSON, 0, len(req.Metrics))
    for _, m := range req.Metrics {
		if m == nil {
			continue
		}
        switch m.GetType() {
		case grpcapi.MetricType_GAUGE:
            val := m.GetValue()
            batch = append(batch, serviceMetricJSON{ID: m.GetId(), MType: service.GaugeMetric, FValue: &val})
		case grpcapi.MetricType_COUNTER:
            d := m.GetDelta()
            batch = append(batch, serviceMetricJSON{ID: m.GetId(), MType: service.CounterMetric, IValue: &d})
		}
	}
	updated, err := s.updateBatchThroughService(batch)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	// map back to grpc
    resp := &grpcapi.BatchUpdateResponse{Metrics: make([]*grpcapi.Metric, 0, len(updated))}
    for _, m := range updated {
        out := &grpcapi.Metric{Id: proto.String(m.ID)}
        if m.MType == service.CounterMetric && m.IValue != nil {
            tt := grpcapi.MetricType_COUNTER
            out.Type = &tt
            out.MetricValue = &grpcapi.Metric_Delta{Delta: *m.IValue}
        } else if m.MType == service.GaugeMetric && m.FValue != nil {
            tt := grpcapi.MetricType_GAUGE
            out.Type = &tt
            out.MetricValue = &grpcapi.Metric_Value{Value: *m.FValue}
        }
        resp.Metrics = append(resp.Metrics, out)
    }
	return resp, nil
}

func (s *MetricsServer) GetValue(ctx context.Context, req *grpcapi.GetValueRequest) (*grpcapi.GetValueResponse, error) {
    if req == nil || req.GetId() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "invalid request")
	}
	var mtype string
    switch req.GetType() {
	case grpcapi.MetricType_GAUGE:
		mtype = service.GaugeMetric
	case grpcapi.MetricType_COUNTER:
		mtype = service.CounterMetric
	default:
		return nil, status.Errorf(codes.InvalidArgument, "unsupported type")
	}
    val, err := s.svc.GetMetricValue(mtype, req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "metric not found")
	}
    out := &grpcapi.Metric{Id: proto.String(req.GetId())}
    switch req.GetType() {
	case grpcapi.MetricType_GAUGE:
		fv, err := strconv.ParseFloat(strings.TrimSpace(val), 64)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to parse gauge")
		}
        tt := grpcapi.MetricType_GAUGE
        out.Type = &tt
		out.MetricValue = &grpcapi.Metric_Value{Value: fv}
	case grpcapi.MetricType_COUNTER:
		iv, err := strconv.ParseInt(strings.TrimSpace(val), 10, 64)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to parse counter")
		}
        tt := grpcapi.MetricType_COUNTER
        out.Type = &tt
		out.MetricValue = &grpcapi.Metric_Delta{Delta: iv}
	}
	return &grpcapi.GetValueResponse{Metric: out}, nil
}

func (s *MetricsServer) GetAllMetrics(ctx context.Context, _ *grpcapi.GetAllMetricsRequest) (*grpcapi.GetAllMetricsResponse, error) {
	all := s.svc.GetAllMetrics()
	// Parse to counters from strings (best-effort)
	gauges := map[string]string{}
	for k, v := range all {
		gauges[k] = v
	}
	// For counters try parse as int64; if not parseable, skip
	counters := map[string]int64{}
	for k, v := range all {
		var iv int64
		if parsed, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64); err == nil {
			iv = parsed
			counters[k] = iv
		}
	}
	return &grpcapi.GetAllMetricsResponse{Gauges: gauges, Counters: counters}, nil
}

// serviceMetricJSON is a minimal DTO to call service.UpdateMetricsBatch
type serviceMetricJSON struct {
	ID     string
	MType  string
	IValue *int64   // for counter
	FValue *float64 // for gauge
}

// updateBatchThroughService converts to middleware.MetricsJSON and calls service
func (s *MetricsServer) updateBatchThroughService(batch []serviceMetricJSON) ([]serviceMetricJSON, error) {
	// local wrapper to avoid importing middleware package here
	type mj struct {
		ID    string
		MType string
		Delta *int64
		Value *float64
	}
	in := make([]mj, 0, len(batch))
	for _, b := range batch {
		in = append(in, mj{ID: b.ID, MType: b.MType, Delta: b.IValue, Value: b.FValue})
	}
	// call the service via its public method by reconstructing middleware.MetricsJSON using reflection is overkill;
	// Instead, use the existing service.Storage.UpdateMetricsBatch requires []middleware.MetricsJSON, which is in another package.
	// To keep dependency clean, we reimplement batch via per-metric UpdateMetric.
	// This is acceptable as service.UpdateMetricsBatch internally iterates too.
	for _, m := range in {
		switch strings.ToLower(m.MType) {
		case service.CounterMetric:
			if m.Delta == nil {
				return nil, errors.New("counter missing delta")
			}
			if err := s.svc.UpdateMetric(service.CounterMetric, m.ID, fmt.Sprintf("%d", *m.Delta)); err != nil {
				return nil, err
			}
		case service.GaugeMetric:
			if m.Value == nil {
				return nil, errors.New("gauge missing value")
			}
			if err := s.svc.UpdateMetric(service.GaugeMetric, m.ID, fmt.Sprintf("%g", *m.Value)); err != nil {
				return nil, err
			}
		default:
			return nil, errors.New("unsupported type")
		}
	}
	// Build result with current values
	out := make([]serviceMetricJSON, 0, len(in))
	for _, m := range in {
		val, err := s.svc.GetMetricValue(m.MType, m.ID)
		if err != nil {
			continue
		}
		if m.MType == service.CounterMetric {
			if iv, err := strconv.ParseInt(strings.TrimSpace(val), 10, 64); err == nil {
				ivCopy := iv
				out = append(out, serviceMetricJSON{ID: m.ID, MType: m.MType, IValue: &ivCopy})
			}
		} else {
			if fv, err := strconv.ParseFloat(strings.TrimSpace(val), 64); err == nil {
				fvCopy := fv
				out = append(out, serviceMetricJSON{ID: m.ID, MType: m.MType, FValue: &fvCopy})
			}
		}
	}
	return out, nil
}

// StartGRPCServer starts gRPC server according to cfg and returns the server and listener for graceful shutdown
func StartGRPCServer(cfg *servercfg.Config, svc *service.MetricsService) (*grpc.Server, net.Listener, error) {
	var opts []grpc.ServerOption
	// chain interceptors: trusted subnet then hmac
    ts, err := middleware.TrustedSubnetUnaryInterceptor(cfg.TrustedSubnet)
	if err != nil {
		return nil, nil, err
	}
	opts = append(opts, grpc.ChainUnaryInterceptor(ts, hmacInterceptor(cfg.Key)))

	if cfg.EnableGRPCTLS {
		if cfg.GRPCCertFile == "" || cfg.GRPCKeyFile == "" {
			return nil, nil, errors.New("grpc tls enabled but cert/key not provided")
		}
		creds, err := credentials.NewServerTLSFromFile(cfg.GRPCCertFile, cfg.GRPCKeyFile)
		if err != nil {
			return nil, nil, err
		}
		opts = append(opts, grpc.Creds(creds))
	}

	g := grpc.NewServer(opts...)
	grpcapi.RegisterMetricsServiceServer(g, NewMetricsServer(svc))

	ln, err := net.Listen("tcp", cfg.GRPCAddress)
	if err != nil {
		return nil, nil, err
	}
	// start in goroutine: we don't block here; caller should Serve in goroutine if needed
	go func() { _ = g.Serve(ln) }()
	// small delay to ensure server starts
	time.Sleep(50 * time.Millisecond)
	return g, ln, nil
}
