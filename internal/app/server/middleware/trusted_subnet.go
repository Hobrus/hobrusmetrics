package middleware

import (
	"net"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// TrustedSubnetMiddleware проверяет, что X-Real-IP клиента входит в доверенную подсеть CIDR
// Проверка применяется только к запросам отправки метрик: POST /update..., POST /updates...
// При пустом cidr проверка отключена.
func TrustedSubnetMiddleware(cidr string) gin.HandlerFunc {
	var ipnet *net.IPNet
	if cidr != "" {
		if _, n, err := net.ParseCIDR(strings.TrimSpace(cidr)); err == nil {
			ipnet = n
		}
	}

	return func(c *gin.Context) {
		// Если проверка отключена — пропускаем
		if ipnet == nil {
			c.Next()
			return
		}

		// Применяем только к POST /update* и POST /updates*
		if c.Request.Method != http.MethodPost {
			c.Next()
			return
		}
		path := c.Request.URL.Path
		if !(strings.HasPrefix(path, "/update") || strings.HasPrefix(path, "/updates")) {
			c.Next()
			return
		}

		ipStr := strings.TrimSpace(c.Request.Header.Get("X-Real-IP"))
		if ipStr == "" {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		ip := net.ParseIP(ipStr)
		if ip == nil {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		if !ipnet.Contains(ip) {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		c.Next()
	}
}
