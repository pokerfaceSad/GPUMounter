package log

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
)

var Logger *zap.SugaredLogger

func InitLogger(logFileDir string, logFileName string) {
	writerSyncer := getLogWriter(logFileDir, logFileName)
	encoder := getEncoder()
	core := zapcore.NewCore(encoder, writerSyncer, zapcore.DebugLevel)
	log := zap.New(core, zap.AddCaller())
	Logger = log.Sugar()
}

func getEncoder() zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	return zapcore.NewConsoleEncoder(encoderConfig)
}

func getLogWriter(logFileDir string, logFileName string) zapcore.WriteSyncer {
	os.MkdirAll(logFileDir, os.ModePerm)
	file, _ := os.Create(logFileDir + logFileName)
	return zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), zapcore.AddSync(file))
}
