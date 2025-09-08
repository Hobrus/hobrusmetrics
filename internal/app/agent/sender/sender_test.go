package sender

import (
    "bytes"
    "compress/gzip"
    "io"
    "net/http"
    "net/http/httptest"
    "testing"
)

// test server that echoes request body length
func TestSenderSendAndBatch(t *testing.T) {
    // mock server that returns 200 and consumes body
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // just drain body
        _, _ = io.Copy(io.Discard, r.Body)
        _ = r.Body.Close()
        w.WriteHeader(http.StatusOK)
    }))
    defer srv.Close()

    s := NewSender(srv.Listener.Addr().String(), "")

    // Single send
    s.Send(map[string]interface{}{
        "c": int64(1),
        "g": float64(2.5),
    })

    // Batch send
    s.SendBatch(map[string]interface{}{
        "c": int64(1),
        "g": float64(2.5),
    })
}

func TestCompressData(t *testing.T) {
    buf, err := compressData([]byte("hello"))
    if err != nil {
        t.Fatalf("compress: %v", err)
    }
    gr, err := gzip.NewReader(bytes.NewReader(buf.Bytes()))
    if err != nil {
        t.Fatalf("gzip reader: %v", err)
    }
    out, _ := io.ReadAll(gr)
    _ = gr.Close()
    if string(out) != "hello" {
        t.Fatalf("unexpected: %s", out)
    }
}


