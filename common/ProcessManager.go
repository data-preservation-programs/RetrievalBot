package common

import (
	"context"
	logging "github.com/ipfs/go-log/v2"
	"github.com/pkg/errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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

func MustSetEnv(key, value string) {
	err := os.Setenv(key, value)
	if err != nil {
		panic(err)
	}
}

func NewProcessManager() (*ProcessManager, error) {
	logger := logging.Logger("process-manager")
	// Setup all modules
	concurrency := make(map[string]int)
	modules := strings.Split(GetRequiredEnv("MODULES"), ",")
	for _, module := range modules {
		path, err := exec.LookPath(module)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to find module %s", module)
		}

		moduleName := strings.ToUpper(strings.Split(filepath.Base(path), ".")[0])
		logger.Infof("Found module %s at %s. Looking for CONCURRENCY_%s now.", module, path, moduleName)

		concurrencyString := GetRequiredEnv("CONCURRENCY_" + moduleName)
		concurrencyNumber, err := strconv.Atoi(concurrencyString)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse concurrency number for module %s", module)
		}

		concurrency[path] = concurrencyNumber
	}

	// Check public IP address
	ipInfo, err := GetPublicIPInfo()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get public IP info")
	}

	logger.With("ipinfo", ipInfo).Infof("Public IP info retrieved")

	loc := strings.Split(ipInfo.Loc, ",")
	//nolint:gomnd
	if len(loc) != 2 {
		return nil, errors.Errorf("invalid location info: %s", ipInfo.Loc)
	}

	if _, err := strconv.ParseFloat(loc[0], 32); err != nil {
		return nil, errors.Errorf("invalid latitude: %s", loc[0])
	}

	if _, err := strconv.ParseFloat(loc[1], 32); err != nil {
		return nil, errors.Errorf("invalid longitude: %s", loc[1])
	}

	//nolint:gomnd
	org := strings.SplitN(ipInfo.Org, " ", 2)
	//nolint:gomnd
	if len(org) != 2 {
		return nil, errors.Errorf("invalid org info: %s", ipInfo.Org)
	}

	MustSetEnv("_PUBLIC_IP", ipInfo.IP)
	MustSetEnv("_CITY", ipInfo.City)
	MustSetEnv("_REGION", ipInfo.Region)
	MustSetEnv("_COUNTRY", ipInfo.Country)
	MustSetEnv("_CONTINENT", "")
	MustSetEnv("_ASN", org[0])
	MustSetEnv("_ORG", org[1])
	MustSetEnv("_LATITUDE", loc[0])
	MustSetEnv("_LONGITUDE", loc[1])

	errorIntervalString := os.Getenv("ERROR_INTERVAL_SECOND")
	if errorIntervalString == "" {
		errorIntervalString = "5"
	}

	errorIntervalNumber, err := strconv.Atoi(errorIntervalString)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse error interval")
	}

	errorInterval := time.Duration(errorIntervalNumber) * time.Second

	return &ProcessManager{
		concurrency,
		errorInterval,
	}, nil
}
