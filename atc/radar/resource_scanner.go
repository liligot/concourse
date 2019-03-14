package radar

import (
	"context"
	"errors"
	"fmt"
	"time"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagerctx"
	"github.com/concourse/concourse/atc"
	"github.com/concourse/concourse/atc/creds"
	"github.com/concourse/concourse/atc/db"
	"github.com/concourse/concourse/atc/metric"
	"github.com/concourse/concourse/atc/resource"
	"github.com/concourse/concourse/atc/worker"
)

var GlobalResourceCheckTimeout time.Duration

type resourceScanner struct {
	clock           clock.Clock
	resourceFactory resource.ResourceFactory
	defaultInterval time.Duration
	dbPipeline      db.Pipeline
	externalURL     string
	variables       creds.Variables

	conn db.Conn
}

func NewResourceScanner(
	conn db.Conn,
	clock clock.Clock,
	resourceFactory resource.ResourceFactory,
	defaultInterval time.Duration,
	dbPipeline db.Pipeline,
	externalURL string,
	variables creds.Variables,
) Scanner {
	return &resourceScanner{
		conn:            conn,
		clock:           clock,
		resourceFactory: resourceFactory,
		defaultInterval: defaultInterval,
		dbPipeline:      dbPipeline,
		externalURL:     externalURL,
		variables:       variables,
	}
}

var ErrFailedToAcquireLock = errors.New("failed to acquire lock")
var ErrResourceTypeNotFound = errors.New("resource type not found")
var ErrResourceTypeCheckError = errors.New("resource type failed to check")

func (scanner *resourceScanner) Run(logger lager.Logger, resourceName string) (time.Duration, error) {
	interval, err := scanner.scan(logger.Session("tick"), resourceName, nil, false, false)

	err = swallowErrResourceScriptFailed(err)

	return interval, err
}

func (scanner *resourceScanner) ScanFromVersion(logger lager.Logger, resourceName string, fromVersion map[atc.Space]atc.Version) error {
	_, err := scanner.scan(logger, resourceName, fromVersion, true, true)

	return err
}

func (scanner *resourceScanner) Scan(logger lager.Logger, resourceName string) error {
	_, err := scanner.scan(logger, resourceName, nil, true, false)

	err = swallowErrResourceScriptFailed(err)

	return err
}

func (scanner *resourceScanner) scan(logger lager.Logger, resourceName string, fromVersion map[atc.Space]atc.Version, mustComplete bool, saveGiven bool) (time.Duration, error) {
	lockLogger := logger.Session("lock", lager.Data{
		"resource": resourceName,
	})

	savedResource, found, err := scanner.dbPipeline.Resource(resourceName)
	if err != nil {
		return 0, err
	}

	if !found {
		logger.Debug("resource-not-found")
		return 0, db.ResourceNotFoundError{Name: resourceName}
	}

	timeout, err := scanner.parseResourceCheckTimeoutOrDefault(savedResource.CheckTimeout())
	if err != nil {
		scanner.setResourceCheckError(logger, savedResource, err)
		logger.Error("failed-to-read-check-timeout", err)
		return 0, err
	}

	interval, err := scanner.checkInterval(savedResource.CheckEvery())
	if err != nil {
		scanner.setResourceCheckError(logger, savedResource, err)
		logger.Error("failed-to-read-check-interval", err)
		return 0, err
	}

	resourceTypes, err := scanner.dbPipeline.ResourceTypes()
	if err != nil {
		logger.Error("failed-to-get-resource-types", err)
		return 0, err
	}

	for _, parentType := range resourceTypes {
		if parentType.Name() != savedResource.Type() {
			continue
		}

		for {
			version, err := parentType.Version()
			if err != nil {
				return 0, err
			}

			if version != nil {
				break
			}

			// XXX: CHECK SETUP ERROR TOO
			if parentType.CheckError() != nil {
				scanner.setResourceCheckError(logger, savedResource, parentType.CheckError())
				logger.Error("resource-type-failed-to-check", err, lager.Data{"resource-type": parentType.Name()})
				return 0, ErrResourceTypeCheckError
			} else {
				logger.Debug("waiting-on-resource-type-version", lager.Data{"resource-type": parentType.Name()})
				scanner.clock.Sleep(10 * time.Second)
			}
		}
	}

	resourceTypes, err = scanner.dbPipeline.ResourceTypes()
	if err != nil {
		logger.Error("failed-to-get-resource-types", err)
		return 0, err
	}

	vrts, err := resourceTypes.Deserialize()
	if err != nil {
		logger.Error("failed-to-deserialize-resource-types", err)
		return 0, err
	}

	versionedResourceTypes := creds.NewVersionedResourceTypes(
		scanner.variables,
		vrts,
	)

	source, err := creds.NewSource(scanner.variables, savedResource.Source()).Evaluate()
	if err != nil {
		logger.Error("failed-to-evaluate-resource-source", err)
		scanner.setResourceCheckError(logger, savedResource, err)
		return 0, err
	}

	resourceConfigScope, err := savedResource.SetResourceConfig(
		logger,
		source,
		versionedResourceTypes,
	)
	if err != nil {
		logger.Error("failed-to-set-resource-config-id-on-resource", err)
		scanner.setResourceCheckError(logger, savedResource, err)
		return 0, err
	}

	// Clear out check error on the resource
	scanner.setResourceCheckError(logger, savedResource, nil)

	// XXX figure out how to pin versions
	// currentVersion := savedResource.CurrentPinnedVersion()
	// if currentVersion != nil {
	// 	// XXX: Need to grab space from the pinned version?
	// 	_, found, err := resourceConfig.FindVersion(currentVersion, atc.Space(""))

	// 	if err != nil {
	// 		logger.Error("failed-to-find-pinned-version-on-resource", err, lager.Data{"pinned-version": currentVersion})
	// chkErr := resourceConfigScope.SetCheckError(err)
	// 		if chkErr != nil {
	// 			logger.Error("failed-to-set-check-error-on-resource-config", chkErr)
	// 		}
	// 		return 0, err
	// 	}
	// 	if found {
	// 		logger.Info("skipping-check-because-pinned-version-found", lager.Data{"pinned-version": currentVersion})
	// 		return interval, nil
	// 	}

	// 	fromVersion = currentVersion
	// }

	for {
		lock, acquired, err := resourceConfigScope.AcquireResourceCheckingLock(
			logger,
			interval,
		)
		if err != nil {
			lockLogger.Error("failed-to-get-lock", err, lager.Data{
				"resource_name":   resourceName,
				"resource_config": resourceConfigScope.ResourceConfig().ID(),
			})
			return interval, ErrFailedToAcquireLock
		}

		if !acquired {
			lockLogger.Debug("did-not-get-lock")
			scanner.clock.Sleep(time.Second)
			continue
		}

		defer lock.Release()

		updated, err := resourceConfigScope.UpdateLastChecked(interval, mustComplete)
		if err != nil {
			return interval, err
		}

		if !updated {
			logger.Debug("interval-not-reached", lager.Data{
				"interval": interval,
			})
			return interval, ErrFailedToAcquireLock
		}

		break
	}

	latestVersions, err := resourceConfigScope.LatestVersions()
	if err != nil {
		logger.Error("failed-to-get-current-version", err)
		return interval, err
	}

	latestFromVersions := make(map[atc.Space]atc.Version)
	for _, resourceVersion := range latestVersions {
		latestFromVersions[resourceVersion.Space()] = atc.Version(resourceVersion.Version())
	}

	for space, version := range fromVersion {
		latestFromVersions[space] = version
	}

	return interval, scanner.check(
		logger,
		savedResource,
		resourceConfigScope,
		latestFromVersions,
		versionedResourceTypes,
		source,
		saveGiven,
		timeout,
	)
}

func (scanner *resourceScanner) check(
	logger lager.Logger,
	savedResource db.Resource,
	resourceConfigScope db.ResourceConfigScope,
	fromVersion map[atc.Space]atc.Version,
	resourceTypes creds.VersionedResourceTypes,
	source atc.Source,
	saveGiven bool,
	timeout time.Duration,
) error {
	pipelinePaused, err := scanner.dbPipeline.CheckPaused()
	if err != nil {
		logger.Error("failed-to-check-if-pipeline-paused", err)
		return err
	}

	if pipelinePaused {
		logger.Debug("pipeline-paused")
		return nil
	}

	found, err := scanner.dbPipeline.Reload()
	if err != nil {
		logger.Error("failed-to-reload-scannerdb", err)
		return err
	}
	if !found {
		logger.Info("pipeline-removed")
		return errPipelineRemoved
	}

	metadata := resource.TrackerMetadata{
		ResourceName: savedResource.Name(),
		PipelineName: savedResource.PipelineName(),
		ExternalURL:  scanner.externalURL,
	}

	containerSpec := worker.ContainerSpec{
		ImageSpec: worker.ImageSpec{
			ResourceType: savedResource.Type(),
		},
		Tags:   savedResource.Tags(),
		TeamID: scanner.dbPipeline.TeamID(),
		Env:    metadata.Env(),
	}

	workerSpec := worker.WorkerSpec{
		ResourceType:  savedResource.Type(),
		Tags:          savedResource.Tags(),
		ResourceTypes: resourceTypes,
		TeamID:        scanner.dbPipeline.TeamID(),
	}

	res, err := scanner.resourceFactory.NewResource(
		context.Background(),
		logger,
		db.NewResourceConfigCheckSessionContainerOwner(resourceConfigScope.ResourceConfig(), ContainerExpiries),
		db.ContainerMetadata{
			Type: db.ContainerTypeCheck,
		},
		containerSpec,
		workerSpec,
		resourceTypes,
		worker.NoopImageFetchingDelegate{},
	)

	if err != nil {
		logger.Error("failed-to-initialize-new-container", err)
		chkErr := resourceConfigScope.SetCheckError(err)
		if chkErr != nil {
			logger.Error("failed-to-set-check-error-on-resource-config", chkErr)
		}
		return err
	}

	logger.Debug("checking", lager.Data{
		"from": fromVersion,
	})

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	tx, err := scanner.conn.Begin()
	if err != nil {
		return err
	}

	defer tx.Rollback()

	spaces := make(map[atc.Space]atc.Version)
	checkHandler := NewCheckEventHandler(logger, tx, resourceConfigScope, spaces)
	err = res.Check(lagerctx.NewContext(ctx, logger), checkHandler, source, fromVersion)
	if err == context.DeadlineExceeded {
		err = fmt.Errorf("Timed out after %v while checking for new versions - perhaps increase your resource check timeout?", timeout)
	}

	resourceConfigScope.SetCheckError(err)
	metric.ResourceCheck{
		PipelineName: scanner.dbPipeline.Name(),
		ResourceName: savedResource.Name(),
		TeamName:     scanner.dbPipeline.TeamName(),
		Success:      err == nil,
	}.Emit(logger)

	if err != nil {
		if rErr, ok := err.(atc.ErrResourceScriptFailed); ok {
			logger.Info("check-failed", lager.Data{"exit-status": rErr.ExitStatus})
			return rErr
		}

		logger.Error("failed-to-check", err)
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func swallowErrResourceScriptFailed(err error) error {
	if _, ok := err.(atc.ErrResourceScriptFailed); ok {
		return nil
	}
	return err
}

func (scanner *resourceScanner) parseResourceCheckTimeoutOrDefault(checkTimeout string) (time.Duration, error) {
	interval := GlobalResourceCheckTimeout
	if checkTimeout != "" {
		configuredInterval, err := time.ParseDuration(checkTimeout)
		if err != nil {
			return 0, err
		}

		interval = configuredInterval
	}

	return interval, nil
}

func (scanner *resourceScanner) checkInterval(checkEvery string) (time.Duration, error) {
	interval := scanner.defaultInterval
	if checkEvery != "" {
		configuredInterval, err := time.ParseDuration(checkEvery)
		if err != nil {
			return 0, err
		}

		interval = configuredInterval
	}

	return interval, nil
}

func (scanner *resourceScanner) setResourceCheckError(logger lager.Logger, savedResource db.Resource, err error) {
	setErr := savedResource.SetCheckSetupError(err)
	if setErr != nil {
		logger.Error("failed-to-set-check-error", err)
	}
}

var errPipelineRemoved = errors.New("pipeline removed")
