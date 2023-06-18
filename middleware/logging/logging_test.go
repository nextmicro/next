package logging

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	std "log"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/go-volo/logger"
)

var _ transport.Transporter = (*Transport)(nil)

type Transport struct {
	kind      transport.Kind
	endpoint  string
	operation string
}

func (tr *Transport) Kind() transport.Kind {
	return tr.kind
}

func (tr *Transport) Endpoint() string {
	return tr.endpoint
}

func (tr *Transport) Operation() string {
	return tr.operation
}

func (tr *Transport) RequestHeader() transport.Header {
	return nil
}

func (tr *Transport) ReplyHeader() transport.Header {
	return nil
}

type mockLogger struct {
	buf *bytes.Buffer
}

func (l *mockLogger) Init(options ...logger.Option) error {
	//TODO implement me
	panic("implement me")
}

func (l *mockLogger) Options() logger.Options {
	//TODO implement me
	panic("implement me")
}

func (l *mockLogger) SetLevel(lv logger.Level) {
	//TODO implement me
	panic("implement me")
}

func (l *mockLogger) WithContext(ctx context.Context) logger.Logger {
	return l
}

func (l *mockLogger) WithFields(fields map[string]interface{}) logger.Logger {
	for k, v := range fields {
		_, _ = fmt.Fprintf(l.buf, " %s=%v", k, v)
	}
	return l
}

func (l *mockLogger) WithCallDepth(callDepth int) logger.Logger {
	//TODO implement me
	panic("implement me")
}

func (l *mockLogger) Debug(args ...interface{}) {
	//TODO implement me
	panic("implement me")
}

func (l *mockLogger) Info(args ...interface{}) {
	fmt.Fprintf(l.buf, "%v", args)
	std.Output(1, l.buf.String())
}

func (l *mockLogger) Warn(args ...interface{}) {
	//TODO implement me
	panic("implement me")
}

func (l *mockLogger) Error(args ...interface{}) {
	fmt.Fprintf(l.buf, "%v", args)
	std.Output(1, l.buf.String())
}

func (l *mockLogger) Fatal(args ...interface{}) {
	//TODO implement me
	panic("implement me")
}

func (l *mockLogger) Debugf(template string, args ...interface{}) {
	//TODO implement me
	panic("implement me")
}

func (l *mockLogger) Infof(template string, args ...interface{}) {
	//TODO implement me
	panic("implement me")
}

func (l *mockLogger) Warnf(template string, args ...interface{}) {
	fmt.Fprintf(l.buf, "%v", fmt.Sprintf(template, args...))
	std.Output(1, l.buf.String())
}

func (l *mockLogger) Errorf(template string, args ...interface{}) {
	fmt.Fprintf(l.buf, "%v", fmt.Sprintf(template, args...))
	std.Output(1, l.buf.String())
}

func (l *mockLogger) Fatalf(template string, args ...interface{}) {
	//TODO implement me
	panic("implement me")
}

func (l *mockLogger) Debugw(msg string, keysAndValues ...interface{}) {
	//TODO implement me
	panic("implement me")
}

func (l *mockLogger) Infow(msg string, keysAndValues ...interface{}) {
	//TODO implement me
	panic("implement me")
}

func (l *mockLogger) Warnw(msg string, keysAndValues ...interface{}) {
	//TODO implement me
	panic("implement me")
}

func (l *mockLogger) Errorw(msg string, keysAndValues ...interface{}) {
	//TODO implement me
	panic("implement me")
}

func (l *mockLogger) Fatalw(msg string, keysAndValues ...interface{}) {
	//TODO implement me
	panic("implement me")
}

func (l *mockLogger) String() string {
	//TODO implement me
	panic("implement me")
}

func (l *mockLogger) Sync() error {
	//TODO implement me
	panic("implement me")
}

func TestHTTP(t *testing.T) {
	err := errors.New("reply.error")
	bf := bytes.NewBuffer(nil)
	_log := &mockLogger{
		buf: bf,
	}

	tests := []struct {
		name string
		slow time.Duration
		kind func(opts ...Option) middleware.Middleware
		err  error
		ctx  context.Context
	}{
		{
			"http-server@fail",
			100 * time.Millisecond,
			Server,
			err,
			func() context.Context {
				return transport.NewServerContext(context.Background(), &Transport{kind: transport.KindHTTP, endpoint: "endpoint", operation: "/package.service/method"})
			}(),
		},
		{
			"http-server@succ",
			300 * time.Millisecond,
			Server,
			nil,
			func() context.Context {
				return transport.NewServerContext(context.Background(), &Transport{kind: transport.KindHTTP, endpoint: "endpoint", operation: "/package.service/method"})
			}(),
		},
		{
			"http-client@succ",
			500 * time.Millisecond,
			Client,
			nil,
			func() context.Context {
				return transport.NewClientContext(context.Background(), &Transport{kind: transport.KindHTTP, endpoint: "endpoint", operation: "/package.service/method"})
			}(),
		},
		{
			"http-client@fail",
			100 * time.Millisecond,
			Client,
			err,
			func() context.Context {
				return transport.NewClientContext(context.Background(), &Transport{kind: transport.KindHTTP, endpoint: "endpoint", operation: "/package.service/method"})
			}(),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bf.Reset()
			next := func(ctx context.Context, req interface{}) (interface{}, error) {
				time.Sleep(test.slow)
				return "reply", test.err
			}
			next = test.kind(WithLogger(_log))(next)
			v, e := next(test.ctx, "req.args")
			t.Logf("[%s]reply: %v, error: %v", test.name, v, e)
			t.Logf("[%s]log: %s", test.name, bf.String())
		})
	}
}

type (
	dummy struct {
		field string
	}
	dummyStringer struct {
		field string
	}
	dummyStringerRedacter struct {
		field string
	}
)

func (d *dummyStringer) String() string {
	return "my value"
}

func (d *dummyStringerRedacter) String() string {
	return "my value"
}

func (d *dummyStringerRedacter) Redact() string {
	return "my value redacted"
}

func TestExtractArgs(t *testing.T) {
	tests := []struct {
		name     string
		req      interface{}
		expected string
	}{
		{
			name:     "dummyStringer",
			req:      &dummyStringer{field: ""},
			expected: "my value",
		}, {
			name:     "dummy",
			req:      &dummy{field: "value"},
			expected: "&{field:value}",
		}, {
			name:     "dummyStringerRedacter",
			req:      &dummyStringerRedacter{field: ""},
			expected: "my value redacted",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if value := extractArgs(test.req); value != test.expected {
				t.Errorf(`The stringified %s structure must be equal to "%s", %v given`, test.name, test.expected, value)
			}
		})
	}
}

func TestExtractError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantLevel  log.Level
		wantErrStr string
	}{
		{
			"no error", nil, log.LevelInfo, "",
		},
		{
			"error", errors.New("test error"), log.LevelError, "test error",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			errStr := extractError(test.err)
			if errStr != test.wantErrStr {
				t.Errorf("want: %s, got: %s", test.wantErrStr, errStr)
			}
		})
	}
}
