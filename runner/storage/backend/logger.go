package backend

import (
	"go.uber.org/zap"
)

type Logger struct {
	lg *zap.Logger
}

func (l *Logger) Errorf(format string, args ...interface{}) {
	l.lg.Sugar().Errorf(format, args...)
}

func (l *Logger) Warningf(format string, args ...interface{}) {
	l.lg.Sugar().Warnf(format, args...)
}

func (l *Logger) Infof(format string, args ...interface{}) {
	l.lg.Sugar().Infof(format, args...)
}

func (l *Logger) Debugf(format string, args ...interface{}) {
	l.lg.Sugar().Debugf(format, args...)
}
