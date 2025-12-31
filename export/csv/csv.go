package csv

import (
	"bufio"
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/94peter/vulpes/storage"
)

type CSVMarshaler interface {
	// MarshalCSV returns header (optional) and multiple rows.
	// If headers is nil, library will not auto-generate header.
	MarshalCSV() (headers []string, rows [][]string, err error)
}

// Options configures the Writer behavior.
type Config struct {
	Delimiter rune // field delimiter, default ','
	UseBOM    bool // write UTF-8 BOM at start
	UseCRLF   bool // use CRLF line endings
}

var defaultOption = &Config{
	Delimiter: ',',
	UseBOM:    true,
	UseCRLF:   true,
}

// Writer is the main CSV generator.
type Writer struct {
	w      *csv.Writer
	bufw   *bufio.Writer
	opts   *Config
	closed bool
}

// New creates a new Writer writing to provided io.Writer.
func new(out io.Writer, opts ...Option) *Writer {
	for _, opt := range opts {
		opt(defaultOption)
	}

	bufw := bufio.NewWriter(out)
	cw := csv.NewWriter(bufw)
	cw.Comma = defaultOption.Delimiter
	cw.UseCRLF = defaultOption.UseCRLF
	return &Writer{w: cw, bufw: bufw, opts: defaultOption}
}

func (cw *Writer) writeBOM() error {
	if !cw.opts.UseBOM {
		return nil
	}
	if cw.closed {
		return errors.New("writer closed")
	}
	_, err := cw.bufw.Write([]byte{0xEF, 0xBB, 0xBF})
	return err
}

// WriteHeader writes header fields as a CSV row.
func (cw *Writer) writeHeader(fields []string) error {
	if cw.closed {
		return errors.New("writer closed")
	}
	// ensure BOM is written if configured — best-effort
	if cw.opts.UseBOM {
		_ = cw.writeBOM()
	}
	return cw.w.Write(fields)
}

// WriteRecord writes a single row (slice of string fields).
func (cw *Writer) writeRecord(fields []string) error {
	if cw.closed {
		return errors.New("writer closed")
	}
	return cw.w.Write(fields)
}

// Flush flushes internal buffers and the underlying writer.
func (cw *Writer) flush() {
	cw.w.Flush()
	_ = cw.bufw.Flush()
}

// Close flushes and marks writer closed. It does not close underlying io.Writer.
func (cw *Writer) close() error {
	if cw.closed {
		return errors.New("already closed")
	}
	cw.flush()
	cw.closed = true
	return nil
}

// Write writes a single CSVMarshaler object to the provided io.Writer.
// It supports multiple rows and optional header from the object.
func Write(out io.Writer, data CSVMarshaler, opts ...Option) error {
	w := new(out, opts...)
	defer w.close()

	headers, rows, err := data.MarshalCSV()
	if err != nil {
		return err
	}

	if headers != nil {
		if err := w.writeHeader(headers); err != nil {
			return err
		}
	}

	for _, row := range rows {
		if err := w.writeRecord(row); err != nil {
			return err
		}
	}

	w.flush()
	return nil
}

func Upload(
	ctx context.Context, storage storage.Storage, key string,
	data CSVMarshaler, opts ...Option,
) (string, error) {
	buf := &bytes.Buffer{}
	err := Write(buf, data, opts...)
	if err != nil {
		return "", fmt.Errorf("failed to write csv: %w", err)
	}
	err = storage.Upload(ctx, key, buf, "text/csv")
	if err != nil {
		return "", fmt.Errorf("faile to update to storage: %w", err)
	}
	return storage.SignedDownloadUrl(ctx, key, 20*time.Minute)
}
