/*
Copyright 2025 The olive Authors

This program is offered under a commercial and under the AGPL license.
For AGPL licensing, see below.

AGPL licensing:
This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package server

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/utils"
)

type gormLogger struct {
	level logger.LogLevel
	lg    *zap.SugaredLogger

	SlowThreshold time.Duration

	IgnoreRecordNotFoundError bool

	traceStr, traceErrStr, traceWarnStr string
}

func (glog *gormLogger) LogMode(level logger.LogLevel) logger.Interface {
	glog.level = level
	return glog
}

func (glog gormLogger) Info(ctx context.Context, format string, i ...interface{}) {
	if glog.level >= logger.Info {
		glog.lg.Infof(format, i...)
	}
}

func (glog gormLogger) Warn(ctx context.Context, format string, i ...interface{}) {
	if glog.level >= logger.Warn {
		glog.lg.Warnf(format, i...)
	}
}

func (glog gormLogger) Error(ctx context.Context, format string, i ...interface{}) {
	if glog.level >= logger.Error {
		glog.lg.Errorf(format, i...)
	}
}

func (glog gormLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if glog.level <= logger.Silent {
		return
	}

	elapsed := time.Since(begin)
	switch {
	case err != nil && glog.level >= logger.Error && (!errors.Is(err, gorm.ErrRecordNotFound) || !glog.IgnoreRecordNotFoundError):
		sql, rows := fc()
		if rows == -1 {
			glog.lg.Errorf(glog.traceErrStr, utils.FileWithLineNum(), err, float64(elapsed.Nanoseconds())/1e6, "-", sql)
		} else {
			glog.lg.Errorf(glog.traceErrStr, utils.FileWithLineNum(), err, float64(elapsed.Nanoseconds())/1e6, rows, sql)
		}
	case elapsed > glog.SlowThreshold && glog.SlowThreshold != 0 && glog.level >= logger.Warn:
		sql, rows := fc()
		slowLog := fmt.Sprintf("SLOW SQL >= %v", glog.SlowThreshold)
		if rows == -1 {
			glog.lg.Logf(zap.WarnLevel, glog.traceWarnStr, utils.FileWithLineNum(), slowLog, float64(elapsed.Nanoseconds())/1e6, "-", sql)
		} else {
			glog.lg.Logf(zap.WarnLevel, glog.traceWarnStr, utils.FileWithLineNum(), slowLog, float64(elapsed.Nanoseconds())/1e6, rows, sql)
		}
	case glog.level == logger.Info:
		sql, rows := fc()
		if rows == -1 {
			glog.lg.Infof(glog.traceStr, utils.FileWithLineNum(), float64(elapsed.Nanoseconds())/1e6, "-", sql)
		} else {
			glog.lg.Infof(glog.traceStr, utils.FileWithLineNum(), float64(elapsed.Nanoseconds())/1e6, rows, sql)
		}
	}
}

func openLocalDB(lg *zap.Logger, dataDir string) (*gorm.DB, error) {
	var (
		traceStr     = "%s\n[%.3fms] [rows:%v] %s"
		traceWarnStr = "%s %s\n[%.3fms] [rows:%v] %s"
		traceErrStr  = "%s %s\n[%.3fms] [rows:%v] %s"
	)

	dbLogger := &gormLogger{
		level: logger.Warn,
		lg:    lg.Sugar(),

		SlowThreshold: time.Millisecond * 200,

		IgnoreRecordNotFoundError: true,

		traceErrStr:  traceErrStr,
		traceWarnStr: traceWarnStr,
		traceStr:     traceStr,
	}

	dbPath := filepath.Join(dataDir, "olive.db")
	dbPath += "?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)"
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: dbLogger,
	})
	if err != nil {
		return nil, err
	}
	// 设置数据库连接池参数
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return db, nil
}
