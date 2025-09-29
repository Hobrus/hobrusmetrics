package middleware

import (
    "net"
    "net/http"
    "strings"

    "github.com/gin-gonic/gin"
    "google.golang.org/grpc"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/metadata"
    "google.golang.org/grpc/peer"
    "google.golang.org/grpc/status"
)

// TrustedSubnetValidator инкапсулирует проверку IP на вхождение в доверенную подсеть.
type TrustedSubnetValidator struct {
    ipnet *net.IPNet
}

// NewTrustedSubnetValidator создаёт валидатор. Пустая строка cidr отключает проверку.
// Возвращает ошибку, если CIDR некорректен.
func NewTrustedSubnetValidator(cidr string) (*TrustedSubnetValidator, error) {
    cidr = strings.TrimSpace(cidr)
    if cidr == "" {
        return &TrustedSubnetValidator{ipnet: nil}, nil
    }
    _, ipnet, err := net.ParseCIDR(cidr)
    if err != nil {
        return nil, err
    }
    return &TrustedSubnetValidator{ipnet: ipnet}, nil
}

// newValidatorLenient создаёт валидатор без ошибок: при неверном CIDR проверка будет отключена.
func newValidatorLenient(cidr string) *TrustedSubnetValidator {
    cidr = strings.TrimSpace(cidr)
    if cidr == "" {
        return &TrustedSubnetValidator{ipnet: nil}
    }
    if _, ipnet, err := net.ParseCIDR(cidr); err == nil {
        return &TrustedSubnetValidator{ipnet: ipnet}
    }
    return &TrustedSubnetValidator{ipnet: nil}
}

// Enabled возвращает true, если проверка включена.
func (v *TrustedSubnetValidator) Enabled() bool {
    return v != nil && v.ipnet != nil
}

// IsAllowedIP возвращает true, если ipStr валиден и входит в доверенную подсеть.
// Если валидатор отключён — всегда true.
func (v *TrustedSubnetValidator) IsAllowedIP(ipStr string) bool {
    if !v.Enabled() {
        return true
    }
    ip := net.ParseIP(strings.TrimSpace(ipStr))
    if ip == nil {
        return false
    }
    return v.ipnet.Contains(ip)
}

// shouldCheckHTTP определяет, нужно ли применять проверку к HTTP-запросу.
func (v *TrustedSubnetValidator) shouldCheckHTTP(method, path string) bool {
    if !v.Enabled() {
        return false
    }
    if method != http.MethodPost {
        return false
    }
    return strings.HasPrefix(path, "/update") || strings.HasPrefix(path, "/updates")
}

// shouldCheckGRPC определяет, нужно ли применять проверку к gRPC-методу.
func (v *TrustedSubnetValidator) shouldCheckGRPC(fullMethod string) bool {
    if !v.Enabled() {
        return false
    }
    return strings.HasSuffix(fullMethod, "/UpdateMetric") || strings.HasSuffix(fullMethod, "/BatchUpdate")
}

// TrustedSubnetMiddleware — HTTP middleware, использующее общий валидатор.
// Семантика сохранена: требуется заголовок X-Real-IP, при пустом/невалидном — 403.
// При неверном CIDR проверка просто отключается (как и раньше).
func TrustedSubnetMiddleware(cidr string) gin.HandlerFunc {
    v := newValidatorLenient(cidr)
    return func(c *gin.Context) {
        if !v.shouldCheckHTTP(c.Request.Method, c.Request.URL.Path) {
            c.Next()
            return
        }

        ipStr := strings.TrimSpace(c.Request.Header.Get("X-Real-IP"))
        if !v.IsAllowedIP(ipStr) {
            c.AbortWithStatus(http.StatusForbidden)
            return
        }
        c.Next()
    }
}

// TrustedSubnetUnaryInterceptor — gRPC интерсептор, использующий общий валидатор.
// В отличие от HTTP-версии, при неверном CIDR возвращает ошибку на этапе инициализации.
// IP берётся из метаданных x-real-ip; если отсутствует — из peer.Addr.
func TrustedSubnetUnaryInterceptor(cidr string) (grpc.UnaryServerInterceptor, error) {
    v, err := NewTrustedSubnetValidator(cidr)
    if err != nil {
        return nil, err
    }
    return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
        if !v.shouldCheckGRPC(info.FullMethod) {
            return handler(ctx, req)
        }

        var ipStr string
        if md, ok := metadata.FromIncomingContext(ctx); ok {
            vals := md.Get("x-real-ip")
            if len(vals) > 0 {
                ipStr = strings.TrimSpace(vals[0])
            }
        }
        if ipStr == "" {
            if p, ok := peer.FromContext(ctx); ok && p.Addr != nil {
                host, _, _ := net.SplitHostPort(p.Addr.String())
                ipStr = host
            }
        }
        if !v.IsAllowedIP(ipStr) {
            return nil, status.Errorf(codes.PermissionDenied, "forbidden")
        }
        return handler(ctx, req)
    }, nil
}
