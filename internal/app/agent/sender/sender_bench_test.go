package sender

import (
	"bytes"
	"compress/gzip"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"testing"
)

func BenchmarkCompressData(b *testing.B) {
	payload := bytes.Repeat([]byte(`{"k":"v"}`), 1024)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		if _, err := gz.Write(payload); err != nil {
			b.Fatal(err)
		}
		if err := gz.Close(); err != nil {
			b.Fatal(err)
		}
		_ = buf.Len()
	}
}

func BenchmarkComputeHMAC(b *testing.B) {
	key := []byte("secret-key")
	data := bytes.Repeat([]byte("payload"), 1024)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h := hmac.New(sha256.New, key)
		h.Write(data)
		_ = base64.StdEncoding.EncodeToString(h.Sum(nil))
	}
}
