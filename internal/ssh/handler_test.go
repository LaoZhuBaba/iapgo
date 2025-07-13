package ssh

import (
	"io"
	"log/slog"
	"os"
	"reflect"
	"testing"
)

type testReadWriteCloser struct {
	data     []byte
	readErr  error
	writeErr error
}

func (t *testReadWriteCloser) Read(p []byte) (int, error) {
	for i := range t.data {
		p[i] = t.data[i]
	}
	return len(t.data), t.readErr
}
func (t *testReadWriteCloser) Write(p []byte) (int, error) {
	return len(t.data), t.writeErr
}
func (t *testReadWriteCloser) Close() error {
	return nil
}

var testData1 = []byte("test data")

func TestHandler_Handle(t *testing.T) {
	var logLevel slog.LevelVar
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: &logLevel,
	}))
	slog.SetDefault(logger)
	logLevel.Set(slog.LevelDebug)

	type fields struct {
		logger     *slog.Logger
		localConn  io.ReadWriteCloser
		tunnelConn io.ReadWriteCloser
	}
	tests := []struct {
		name   string
		fields fields
		want   error
		want1  error
	}{
		{
			name: "no_errors",
			fields: fields{
				logger: logger,
				localConn: &testReadWriteCloser{
					data:     []byte("test data"),
					readErr:  io.EOF,
					writeErr: nil,
				},
				tunnelConn: &testReadWriteCloser{
					data:     []byte("test data"),
					readErr:  io.EOF,
					writeErr: nil,
				},
			},
			want:  nil,
			want1: nil,
		},
		{
			name: "local_conn_error",
			fields: fields{
				logger: logger,
				localConn: &testReadWriteCloser{
					data:     []byte("test data"),
					readErr:  io.ErrUnexpectedEOF,
					writeErr: nil,
				},
				tunnelConn: &testReadWriteCloser{
					data:     []byte("test data"),
					readErr:  io.EOF,
					writeErr: nil,
				},
			},
			want:  io.ErrUnexpectedEOF,
			want1: nil,
		},
		{
			name: "tunnel_conn_error",
			fields: fields{
				logger: logger,
				localConn: &testReadWriteCloser{
					data:     []byte("test data"),
					readErr:  io.EOF,
					writeErr: nil,
				},
				tunnelConn: &testReadWriteCloser{
					data:     []byte("test data"),
					readErr:  io.ErrClosedPipe,
					writeErr: nil,
				},
			},
			want:  nil,
			want1: io.ErrClosedPipe,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Handler{
				logger:     tt.fields.logger,
				localConn:  tt.fields.localConn,
				tunnelConn: tt.fields.tunnelConn,
			}
			got, got1 := h.Handle()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Handle() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("Handle() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
