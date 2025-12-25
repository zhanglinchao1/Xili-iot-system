package utils

import (
	"os"
	
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var logger *zap.Logger

// InitLogger 初始化日志系统
func InitLogger(level, format, output string) error {
	// 日志级别
	var zapLevel zapcore.Level
	switch level {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "info":
		zapLevel = zapcore.InfoLevel
	case "warn":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	default:
		zapLevel = zapcore.InfoLevel
	}
	
	// 编码器配置
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
	
	// 编码器类型
	var encoder zapcore.Encoder
	if format == "json" {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}
	
	// 输出配置
	var writeSyncer zapcore.WriteSyncer
	if output == "stdout" || output == "" {
		writeSyncer = zapcore.AddSync(os.Stdout)
	} else {
		// 如果需要文件输出，这里可以添加文件配置
		// file, _ := os.OpenFile(output, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		// writeSyncer = zapcore.AddSync(file)
		writeSyncer = zapcore.AddSync(os.Stdout)
	}
	
	core := zapcore.NewCore(encoder, writeSyncer, zapLevel)
	logger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	
	return nil
}

// GetLogger 获取全局logger
func GetLogger() *zap.Logger {
	if logger == nil {
		// 如果logger未初始化，创建一个默认的
		logger, _ = zap.NewProduction()
	}
	return logger
}

// 便捷方法
func Debug(msg string, fields ...zap.Field) {
	GetLogger().Debug(msg, fields...)
}

func Info(msg string, fields ...zap.Field) {
	GetLogger().Info(msg, fields...)
}

func Warn(msg string, fields ...zap.Field) {
	GetLogger().Warn(msg, fields...)
}

func Error(msg string, fields ...zap.Field) {
	GetLogger().Error(msg, fields...)
}

func Fatal(msg string, fields ...zap.Field) {
	GetLogger().Fatal(msg, fields...)
}

// Sync 刷新日志缓冲区
func Sync() error {
	if logger != nil {
		return logger.Sync()
	}
	return nil
}

