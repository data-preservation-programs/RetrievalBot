package process

import (
	"context"
	"github.com/data-preservation-programs/RetrievalBot/pkg/env"
	"github.com/data-preservation-programs/RetrievalBot/pkg/resolver"
	"github.com/google/uuid"
	logging "github.com/ipfs/go-log/v2"
	"github.com/pkg/errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type ProcessManager struct {
	concurrency   map[string]int
	errorInterval time.Duration
}

func (p ProcessManager) Run(ctx context.Context) {
	for module, concurrency := range p.concurrency {
		for i := 0; i < concurrency; i++ {
			module := module
			logger := logging.Logger("process-manager").With("module", module)
			go func() {
				for {
					cmd := exec.CommandContext(ctx, module)
					cmd.Env = os.Environ()
					label := "correlation_id=" + uuid.New().String()
					if os.Getenv("GOLOG_LOG_LABELS") != "" {
						label = label + os.Getenv("GOLOG_LOG_LABELS") + "," + label
					}
					cmd.Env = append(cmd.Env, "GOLOG_LOG_LABELS="+label)
					cmd.Stdout = os.Stdout
					cmd.Stderr = os.Stderr
					logger.Debug("Spawning new process")
					err := cmd.Run()
					if errors.Is(err, context.Canceled) {
						logger.With("err", err).Infof("Process %s canceled", module)
						return
					}
					if err != nil {
						logger.With("err", err).
							Errorf("Process %s failed. Waiting for %f Seconds", module, p.errorInterval.Seconds())
						time.Sleep(p.errorInterval)
					}
				}
			}()
		}
	}

	<-ctx.Done()
}

func NewProcessManager() (*ProcessManager, error) {
	logger := logging.Logger("process-manager")
	// Setup all worker
	concurrency := make(map[string]int)
	modules := strings.Split(env.GetRequiredString(env.ProcessModules), ",")
	for _, module := range modules {
		path, err := exec.LookPath(module)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to find module %s", module)
		}

		moduleName := strings.ToUpper(strings.Split(filepath.Base(path), ".")[0])
		logger.Infof("Found module %s at %s. Looking for CONCURRENCY_%s now.", module, path, moduleName)

		concurrencyNumber := env.GetInt(env.Key("CONCURRENCY_"+moduleName), 1)
		concurrency[path] = concurrencyNumber
	}

	// Check public IP address
	ipInfo, err := resolver.GetPublicIPInfo(context.TODO(), "", "")
	if err != nil {
		return nil, errors.Wrap(err, "failed to get public IP info")
	}

	logger.With("ipinfo", ipInfo).Infof("Public IP info retrieved")

	env.MustSet(env.PublicIP, ipInfo.IP)
	env.MustSet(env.City, ipInfo.City)
	env.MustSet(env.Region, ipInfo.Region)
	env.MustSet(env.Country, ipInfo.Country)
	env.MustSet(env.Continent, ipInfo.Continent)
	env.MustSet(env.ASN, ipInfo.ASN)
	env.MustSet(env.ISP, ipInfo.ISP)
	env.MustSetAny(env.Latitude, ipInfo.Latitude)
	env.MustSetAny(env.Longitude, ipInfo.Longitude)
	errorInterval := env.GetDuration(env.ProcessErrorInterval, 5*time.Second)

	return &ProcessManager{
		concurrency,
		errorInterval,
	}, nil
}
