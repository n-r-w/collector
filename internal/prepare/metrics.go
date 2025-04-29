package prepare

import (
	"context"
	"fmt"
	"strings"

	"github.com/n-r-w/collector/internal/config"
	"github.com/n-r-w/collector/internal/repository/metrics/otelsrv"
	"github.com/n-r-w/collector/internal/repository/metrics/promsrv"
	"github.com/n-r-w/collector/internal/telemetry"
)

// InitMetrics initializes metrics.
func InitMetrics(ctx context.Context, cfg *config.Config) (telemetry.IMetrics, func(ctx context.Context) error, error) {
	if cfg.Server.MetricsType == "" {
		return nil, nil, nil
	}

	var (
		err                 error
		metricsRegisterFunc func(ctx context.Context) (func(ctx context.Context) error, error)
		metrics             telemetry.IMetrics
	)
	if strings.EqualFold(cfg.Server.MetricsType, "PROMETHEUS") {
		s := promsrv.New()
		metricsRegisterFunc = s.Register
		metrics = s
	} else if strings.EqualFold(cfg.Server.MetricsType, "OTEL") {
		var s *otelsrv.Service
		s, err = otelsrv.New(cfg)
		if err == nil {
			metricsRegisterFunc = s.Register
		}
		metrics = s
	} else {
		err = fmt.Errorf("invalid metrics type %s", cfg.Server.MetricsType)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create metrics service: %w", err)
	}
	metricsCleanupFunc, err := metricsRegisterFunc(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to register metrics: %w", err)
	}

	return metrics,
		func(ctx context.Context) error {
			if err := metricsCleanupFunc(ctx); err != nil {
				return fmt.Errorf("failed to cleanup metrics: %w", err)
			}
			return nil
		}, nil
}
