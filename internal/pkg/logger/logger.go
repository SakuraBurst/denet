package logger

import (
	"io"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type zapLogger struct {
	*zap.Logger
}

func (l *zapLogger) Close() error {
	return l.Sync()
}

var Logger *zapLogger

func InitLogger() (io.Closer, error) {
	var logWriter = zapcore.AddSync(os.Stdout)
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	logCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		logWriter,
		zap.InfoLevel,
	)
	Logger = &zapLogger{
		Logger: zap.New(logCore, zap.AddCallerSkip(1), zap.AddCaller(), zap.AddStacktrace(zap.ErrorLevel)),
	}
	return Logger, nil
}
