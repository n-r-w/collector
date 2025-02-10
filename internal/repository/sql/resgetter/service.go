package resgetter

import (
	"context"

	"github.com/n-r-w/ammo-collector/internal/config"
	"github.com/n-r-w/ammo-collector/internal/repository/sql"
	"github.com/n-r-w/ammo-collector/internal/usecases/finalizer"
	"github.com/n-r-w/pgh/v2/px/db"
	"github.com/n-r-w/pgh/v2/px/db/conn"
	"github.com/n-r-w/pgh/v2/txmgr"
)

// Service implements sql repository.
type Service struct {
	cfg       *config.Config
	txManager txmgr.ITransactionManager
	conn      func(ctx context.Context) conn.IConnection
}

var (
	_ finalizer.IResultChanGetter        = (*Service)(nil)
	_ finalizer.ICollectionResultUpdater = (*Service)(nil)
)

func New(
	cfg *config.Config,
	connectionGetter db.IConnectionGetter,
	txManager txmgr.ITransactionManager,
) (*Service, error) {
	return &Service{
		cfg:       cfg,
		txManager: txManager,
		conn:      sql.GetConn(cfg, connectionGetter),
	}, nil
}
