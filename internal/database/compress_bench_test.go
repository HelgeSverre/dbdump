package database

import (
	"bufio"
	"bytes"
	stdgzip "compress/gzip"
	"io"
	"testing"

	kpgzip "github.com/klauspost/compress/gzip"
)

// BenchmarkGzipImpl compares the stdlib gzip currently used by dbdump against the
// klauspost/compress gzip (already a dependency, drop-in API).
func BenchmarkGzipImpl(b *testing.B) {
	data := syntheticDump(16)
	b.Run("stdlib", func(b *testing.B) {
		b.SetBytes(int64(len(data)))
		for i := 0; i < b.N; i++ {
			w := stdgzip.NewWriter(io.Discard)
			if _, err := w.Write(data); err != nil {
				b.Fatal(err)
			}
			if err := w.Close(); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("klauspost", func(b *testing.B) {
		b.SetBytes(int64(len(data)))
		for i := 0; i < b.N; i++ {
			w := kpgzip.NewWriter(io.Discard)
			if _, err := w.Write(data); err != nil {
				b.Fatal(err)
			}
			if err := w.Close(); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// syntheticDump builds roughly sizeMB of repetitive INSERT lines, standing in for
// the mysqldump byte stream that dbdump pipes through its compression writer.
func syntheticDump(sizeMB int) []byte {
	line := []byte("INSERT INTO `users` VALUES (12345,'Alice Johnson','alice@example.com','2026-01-02 03:04:05',1,'active',0.00);\n")
	var b bytes.Buffer
	b.Grow(sizeMB * 1024 * 1024)
	for b.Len() < sizeMB*1024*1024 {
		b.Write(line)
	}
	return b.Bytes()
}

// BenchmarkCompress exercises the exact write path Dump uses: a 256K bufio writer
// over the format's compressed writer. This is where nearly all of dbdump's own
// CPU goes (everything else is subprocess/IO wait).
func BenchmarkCompress(b *testing.B) {
	data := syntheticDump(16)
	for _, format := range []string{CompressionNone, CompressionGzip, CompressionZstd} {
		b.Run(format, func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				w, err := newCompressedWriter(io.Discard, format)
				if err != nil {
					b.Fatal(err)
				}
				bw := bufio.NewWriterSize(w, 256*1024)
				if _, err := bw.Write(data); err != nil {
					b.Fatal(err)
				}
				if err := bw.Flush(); err != nil {
					b.Fatal(err)
				}
				if err := w.Close(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
