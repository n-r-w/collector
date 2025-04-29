//go:build wireinject
// +build wireinject

package di

import (
	"context"

	"github.com/google/wire"
	"github.com/n-r-w/collector/internal/config"
	"github.com/n-r-w/collector/internal/controller/consumer"
	grpchandlers "github.com/n-r-w/collector/internal/controller/handlers"
	"github.com/n-r-w/collector/internal/repository/s3"
	cleanerrepo "github.com/n-r-w/collector/internal/repository/sql/cleaner"
	colmanagerrepo "github.com/n-r-w/collector/internal/repository/sql/colmanager"
	lockerrepo "github.com/n-r-w/collector/internal/repository/sql/locker"
	reqprocessorrepo "github.com/n-r-w/collector/internal/repository/sql/reqprocessor"
	resgetterrepo "github.com/n-r-w/collector/internal/repository/sql/resgetter"
	"github.com/n-r-w/collector/internal/telemetry"
	"github.com/n-r-w/collector/internal/usecases/apiprocessor"
	"github.com/n-r-w/collector/internal/usecases/cache"
	"github.com/n-r-w/collector/internal/usecases/cleaner"
	"github.com/n-r-w/collector/internal/usecases/finalizer"
	"github.com/n-r-w/collector/internal/usecases/reqprocessor"
	"github.com/n-r-w/grpcsrv"
	"github.com/n-r-w/pgh/v2/px/db"
	"github.com/n-r-w/pgh/v2/txmgr"
)

// Container holds all the dependencies.
type Container struct {
	Config                     *config.Config
	Database                   *db.PxDB
	CleanerRepository          *cleanerrepo.Service
	ColManagerRepository       *colmanagerrepo.Service
	LockerRepository           *lockerrepo.Service
	RequestProcessorRepository *reqprocessorrepo.Service
	ResGetterRepository        *resgetterrepo.Service
	CacheService               *cache.Service
	FinalizerService           *finalizer.Service
	APIProcessorService        *apiprocessor.Service
	RequestProcessorService    *reqprocessor.Service
	CleanupService             *cleaner.Service
	GRPCServer                 *grpcsrv.Service
	GRPCHandlers               *grpchandlers.Service
	KafkaConsumer              *consumer.Service
	S3                         *s3.Service
}

// InitializeContainer initializes the dependency injection container.
func InitializeContainer(
	ctx context.Context,
	cfg *config.Config,
	metrics telemetry.IMetrics,
	dbOpts []db.Option,
	grpcOpts []grpcsrv.Option,
) (*Container, error) {
	wire.Build(
		wire.Struct(new(Container), "*"),
		sqlRepositorySet,
		s3Set,
		databaseSet,
		grpcServerSet,
		usecasesSet,
		kafkaConsumerSet,
	)
	return nil, nil
}

// sqlRepositorySet provides SQL repository and its interface bindings.
var sqlRepositorySet = wire.NewSet(
	resgetterrepo.New,
	wire.Bind(new(finalizer.IResultChanGetter), new(*resgetterrepo.Service)),
	wire.Bind(new(finalizer.ICollectionResultUpdater), new(*resgetterrepo.Service)),

	reqprocessorrepo.New,
	wire.Bind(new(reqprocessor.IRequestStorer), new(*reqprocessorrepo.Service)),

	lockerrepo.New,
	wire.Bind(new(finalizer.ILocker), new(*lockerrepo.Service)),
	wire.Bind(new(cleaner.ILocker), new(*lockerrepo.Service)),

	cleanerrepo.New,
	wire.Bind(new(cleaner.IDatabaseCleaner), new(*cleanerrepo.Service)),

	colmanagerrepo.New,
	wire.Bind(new(apiprocessor.ICollectionCreator), new(*colmanagerrepo.Service)),
	wire.Bind(new(finalizer.IStatusChanger), new(*colmanagerrepo.Service)),
	wire.Bind(new(apiprocessor.ICollectionReader), new(*colmanagerrepo.Service)),
	wire.Bind(new(apiprocessor.ICollectionUpdater), new(*colmanagerrepo.Service)),
	wire.Bind(new(finalizer.ICollectionReader), new(*colmanagerrepo.Service)),
	wire.Bind(new(cache.ICollectionReader), new(*colmanagerrepo.Service)),
	wire.Bind(new(cleaner.ICollectionReader), new(*colmanagerrepo.Service)),
)

// DatabaseSet is a Wire provider set that includes all database dependencies.
var databaseSet = wire.NewSet(
	db.New,
	wire.Bind(new(db.IConnectionGetter), new(*db.PxDB)),
	wire.Bind(new(txmgr.ITransactionBeginner), new(*db.PxDB)),
	wire.Bind(new(txmgr.ITransactionInformer), new(*db.PxDB)),

	txmgr.New,
	wire.Bind(new(txmgr.ITransactionManager), new(*txmgr.TransactionManager)),
)

// s3Set provides S3 storage and its interface bindings.
var s3Set = wire.NewSet(
	s3.New,
	wire.Bind(new(finalizer.IResultChanSaver), new(*s3.Service)),
	wire.Bind(new(apiprocessor.IResultGetter), new(*s3.Service)),
	wire.Bind(new(cleaner.IObjectStorageCleaner), new(*s3.Service)),
)

// grpcServerSet is a Wire provider set that includes all grpc dependencies.
var grpcServerSet = wire.NewSet(
	grpchandlers.New,
	provideGRPCInitializers,
	grpcsrv.New,
)

// kafkaConsumerSet is a Wire provider set that includes all Kafka consumer dependencies.
var kafkaConsumerSet = wire.NewSet(
	consumer.New,
)

// provideGRPCInitializers provides a Wire provider set that includes all gRPC initializers.
func provideGRPCInitializers(handlers *grpchandlers.Service) []grpcsrv.IGRPCInitializer {
	return []grpcsrv.IGRPCInitializer{handlers}
}

// usecasesSet is a Wire provider set that includes all usecases from this package.
var usecasesSet = wire.NewSet(
	cache.New,
	wire.Bind(new(reqprocessor.ICollectionCacher), new(*cache.Service)),

	finalizer.New,
	cleaner.New,

	reqprocessor.New,
	wire.Bind(new(consumer.IHandlers), new(*reqprocessor.Service)),

	apiprocessor.New,
	wire.Bind(new(grpchandlers.ICollectionManager), new(*apiprocessor.Service)),
	wire.Bind(new(grpchandlers.IResultGetter), new(*apiprocessor.Service)),
)
