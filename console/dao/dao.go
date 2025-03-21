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

package dao

import (
	"errors"
	"strings"
	"sync"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/olive-io/olive/api/types"
	"github.com/olive-io/olive/console/config"
	"github.com/olive-io/olive/console/model"
)

var (
	gdb  *gorm.DB
	once sync.Once
)

func Init(cfg *config.Config) error {
	dsnText := cfg.DSN

	driver, connStr, found := strings.Cut(dsnText, "://")
	if !found {
		return errors.New("bad dsn format")
	}

	var err error
	once.Do(func() {
		var dialector gorm.Dialector
		switch driver {
		case "mysql":
			dialector = mysql.Open(connStr)
		case "postgres":
			dialector = postgres.Open(connStr)
		case "sqlite3", "sqlite":
			dialector = sqlite.Open(connStr)
		default:
			err = errors.New("invalid database driver")
			return
		}

		mode := logger.Silent
		switch cfg.LogLevel {
		case "debug":
			mode = logger.Info
		case "warn":
			mode = logger.Warn
		case "error":
			mode = logger.Error
		default:
			mode = logger.Silent
		}
		gdb, err = gorm.Open(dialector, &gorm.Config{
			Logger: logger.Default.LogMode(mode),
		})
		if err != nil {
			return
		}

		if err = gdb.AutoMigrate(&model.Definition{}); err != nil {
			return
		}
		if err = gdb.AutoMigrate(&model.Process{}); err != nil {
			return
		}
		if err = gdb.AutoMigrate(&model.WatchRev{}); err != nil {
			return
		}

		if err = gdb.AutoMigrate(&types.Role{}); err != nil {
			return
		}
		if err = gdb.AutoMigrate(&types.User{}); err != nil {
			return
		}
	})

	return err
}

func GetDB() *gorm.DB {
	return gdb
}

func GetSession(cfg ...*gorm.Session) *gorm.DB {
	sc := &gorm.Session{}
	if cfg != nil && len(cfg) != 0 {
		sc = cfg[0]
	} else {
		sc = &gorm.Session{
			NewDB:       true,
			Initialized: true,
		}
	}
	return gdb.Session(sc)
}
