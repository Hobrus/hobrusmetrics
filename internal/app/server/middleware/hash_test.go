package middleware

import (
    "bytes"
    "compress/gzip"
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "io"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/gin-gonic/gin"
)

func TestHashRequestMiddleware_GzipAndSignature(t *testing.T) {
    gin.SetMode(gin.TestMode)
    router := gin.New()

    key := "k"
    router.Use(HashRequestMiddleware(key))
    router.POST("/u", func(c *gin.Context) {
        data, _ := io.ReadAll(c.Request.Body)
        // after middleware, body should be decompressed JSON
        if string(data) != "{}" {
            c.Status(http.StatusBadRequest)
            return
        }
        c.Status(http.StatusOK)
    })

    // Prepare gzipped body
    var buf bytes.Buffer
    gz := gzip.NewWriter(&buf)
    _, _ = gz.Write([]byte("{}"))
    _ = gz.Close()

    // Compute signature over compressed bytes (original wire data)
    h := hmac.New(sha256.New, []byte(key))
    h.Write(buf.Bytes())
    sig := hex.EncodeToString(h.Sum(nil))

    req, _ := http.NewRequest(http.MethodPost, "/u", bytes.NewReader(buf.Bytes()))
    req.Header.Set("Content-Encoding", "gzip")
    req.Header.Set("HashSHA256", sig)

    rr := httptest.NewRecorder()
    router.ServeHTTP(rr, req)
    if rr.Code != http.StatusOK {
        t.Fatalf("status=%d", rr.Code)
    }
}


