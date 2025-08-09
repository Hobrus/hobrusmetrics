package middleware

import (
    "bytes"
    "compress/gzip"
    "io"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/gin-gonic/gin"
)

func TestGzipMiddleware_CompressResponseAndDecompressRequest(t *testing.T) {
    gin.SetMode(gin.TestMode)
    router := gin.New()
    router.Use(GzipMiddleware())

    router.POST("/echo.json", func(c *gin.Context) {
        data, _ := io.ReadAll(c.Request.Body)
        // Return same payload; middleware should compress if client accepts it
        c.Header("Content-Type", "application/json")
        c.String(http.StatusOK, string(data))
    })

    // Prepare gzipped request body
    var buf bytes.Buffer
    gz := gzip.NewWriter(&buf)
    _, _ = gz.Write([]byte("{\"a\":1}"))
    _ = gz.Close()

    req, _ := http.NewRequest(http.MethodPost, "/echo.json", bytes.NewReader(buf.Bytes()))
    req.Header.Set("Content-Encoding", "gzip")
    req.Header.Set("Accept-Encoding", "gzip")
    req.Header.Set("Content-Type", "application/json")

    rr := httptest.NewRecorder()
    router.ServeHTTP(rr, req)

    if rr.Code != http.StatusOK {
        t.Fatalf("status=%d", rr.Code)
    }
    if rr.Header().Get("Content-Encoding") != "gzip" {
        t.Fatalf("expected gzip response")
    }

    // Decompress response to verify body
    gr, err := gzip.NewReader(bytes.NewReader(rr.Body.Bytes()))
    if err != nil {
        t.Fatalf("gzip reader: %v", err)
    }
    out, _ := io.ReadAll(gr)
    _ = gr.Close()
    if string(out) != "{\"a\":1}" {
        t.Fatalf("unexpected body: %s", out)
    }
}


