package locker

import (
	"context"

	"github.com/n-r-w/collector/internal/config"
	"github.com/n-r-w/collector/internal/repository/sql"
	"github.com/n-r-w/collector/internal/usecases/cleaner"
	"github.com/n-r-w/pgh/v2/px/db"
	"github.com/n-r-w/pgh/v2/px/db/conn"
	"github.com/n-r-w/pgh/v2/txmgr"
)

// Service implements sql repository.
type Service struct {
	txManager txmgr.ITransactionManager
	conn      func(ctx context.Context) conn.IConnection
}

var _ cleaner.ILocker = (*Service)(nil)

func New(
	cfg *config.Config,
	connectionGetter db.IConnectionGetter,
	txManager txmgr.ITransactionManager,
) (*Service, error) {
	return &Service{
		txManager: txManager,
		conn:      sql.GetConn(cfg, connectionGetter),
	}, nil
}
