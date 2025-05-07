package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func InitLogger() (*zap.Logger, error) {
	var logWriter = zapcore.AddSync(os.Stdout)
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	logCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		logWriter,
		zap.InfoLevel,
	)
	return zap.New(logCore, zap.AddCallerSkip(1), zap.AddCaller(), zap.AddStacktrace(zap.ErrorLevel)), nil
}
