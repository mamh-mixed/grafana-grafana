package db

import (
	"fmt"
	"strings"
	"time"

	"github.com/grafana/grafana/pkg/infra/db"
	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/grafana/grafana/pkg/services/sqlstore"
	"github.com/grafana/grafana/pkg/services/sqlstore/session"
	"github.com/grafana/grafana/pkg/setting"
	"github.com/grafana/grafana/pkg/util"
	"github.com/jmoiron/sqlx"
	"xorm.io/xorm"
)

func ProvideEntityDB(db db.DB, cfg *setting.Cfg) (EntityDB, error) {
	var engine *xorm.Engine
	var err error

	cfgSection := cfg.SectionWithEnvOverrides("entity_api")
	dbType := cfgSection.Key("db_type").MustString("")

	if dbType != "" {
		dbHost := cfgSection.Key("db_host").MustString("")
		dbName := cfgSection.Key("db_name").MustString("")
		dbUser := cfgSection.Key("db_user").MustString("")
		dbPass := cfgSection.Key("db_pass").MustString("")

		if dbType == "postgres" {
			dbSslMode := cfgSection.Key("db_sslmode").MustString("disable")

			addr, err := util.SplitHostPortDefault(dbHost, "127.0.0.1", "5432")
			if err != nil {
				return nil, fmt.Errorf("invalid host specifier '%s': %w", dbHost, err)
			}

			connectionString := fmt.Sprintf(
				"user=%s password=%s host=%s port=%s dbname=%s sslmode=%s", // sslcert=%s sslkey=%s sslrootcert=%s",
				dbUser, dbPass, addr.Host, addr.Port, dbName, dbSslMode, // ss.dbCfg.ClientCertPath, ss.dbCfg.ClientKeyPath, ss.dbCfg.CaCertPath
			)

			engine, err = xorm.NewEngine("postgres", connectionString)
			if err != nil {
				return nil, err
			}

			_, err = engine.Query("SET SESSION enable_experimental_alter_column_type_general=true")
			if err != nil {
				return nil, err
			}
		} else if dbType == "mysql" {
			protocol := "tcp"
			if strings.HasPrefix(dbHost, "/") {
				protocol = "unix"
			}

			connectionString := fmt.Sprintf("%s:%s@%s(%s)/%s?collation=utf8mb4_unicode_ci&allowNativePasswords=true&clientFoundRows=true",
				dbUser, dbPass, protocol, dbHost, dbName)

			engine, err = xorm.NewEngine("mysql", connectionString)
			if err != nil {
				return nil, err
			}

			engine.SetMaxOpenConns(0)
			engine.SetMaxIdleConns(2)
			engine.SetConnMaxLifetime(time.Second * time.Duration(14400))

			_, err = engine.Query("SELECT 1")
			if err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("invalid db type specified: %s", dbType)
		}

		// configure sql logging
		debugSQL := cfgSection.Key("log_queries").MustBool(false)
		if !debugSQL {
			engine.SetLogger(&xorm.DiscardLogger{})
		} else {
			// add stack to database calls to be able to see what repository initiated queries. Top 7 items from the stack as they are likely in the xorm library.
			engine.SetLogger(sqlstore.NewXormLogger(log.LvlInfo, log.WithSuffix(log.New("sqlstore.xorm"), log.CallerContextKey, log.StackCaller(log.DefaultCallerDepth))))
			engine.ShowSQL(true)
			engine.ShowExecTime(true)
		}
	} else {
		engine = db.GetEngine()
	}

	eDB := &entityDB{
		cfg:    cfg,
		engine: engine,
	}

	return eDB, nil
}

type EntityDB interface {
	GetSession() *session.SessionDB
	GetEngine() *xorm.Engine
	GetCfg() *setting.Cfg
}

type entityDB struct {
	engine *xorm.Engine
	cfg    *setting.Cfg
}

func (db entityDB) GetSession() *session.SessionDB {
	return session.GetSession(sqlx.NewDb(db.engine.DB().DB, db.engine.DriverName()))
}

func (db entityDB) GetEngine() *xorm.Engine {
	return db.engine
}

func (db entityDB) GetCfg() *setting.Cfg {
	return db.cfg
}
