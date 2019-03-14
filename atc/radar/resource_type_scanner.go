package radar

import (
	"context"
	"time"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagerctx"
	"github.com/concourse/concourse/atc"
	"github.com/concourse/concourse/atc/creds"
	"github.com/concourse/concourse/atc/db"
	"github.com/concourse/concourse/atc/resource"
	"github.com/concourse/concourse/atc/worker"
)

type resourceTypeScanner struct {
	clock           clock.Clock
	resourceFactory resource.ResourceFactory
	defaultInterval time.Duration
	dbPipeline      db.Pipeline
	externalURL     string
	variables       creds.Variables

	conn db.Conn
}

func NewResourceTypeScanner(
	conn db.Conn,
	clock clock.Clock,
	resourceFactory resource.ResourceFactory,
	defaultInterval time.Duration,
	dbPipeline db.Pipeline,
	externalURL string,
	variables creds.Variables,
) Scanner {
	return &resourceTypeScanner{
		conn:            conn,
		clock:           clock,
		resourceFactory: resourceFactory,
		defaultInterval: defaultInterval,
		dbPipeline:      dbPipeline,
		externalURL:     externalURL,
		variables:       variables,
	}
}

func (scanner *resourceTypeScanner) Run(logger lager.Logger, resourceTypeName string) (time.Duration, error) {
	return scanner.scan(logger.Session("tick"), resourceTypeName, nil, false, false)
}

func (scanner *resourceTypeScanner) ScanFromVersion(logger lager.Logger, resourceTypeName string, fromVersion map[atc.Space]atc.Version) error {
	_, err := scanner.scan(logger, resourceTypeName, fromVersion, true, true)
	return err
}

func (scanner *resourceTypeScanner) Scan(logger lager.Logger, resourceTypeName string) error {
	_, err := scanner.scan(logger, resourceTypeName, nil, true, false)
	return err
}

func (scanner *resourceTypeScanner) scan(logger lager.Logger, resourceTypeName string, fromVersion map[atc.Space]atc.Version, mustComplete bool, saveGiven bool) (time.Duration, error) {
	lockLogger := logger.Session("lock", lager.Data{
		"resource-type": resourceTypeName,
	})

	savedResourceType, found, err := scanner.dbPipeline.ResourceType(resourceTypeName)
	if err != nil {
		logger.Error("failed-to-find-resource-type-in-db", err)
		return 0, err
	}

	if !found {
		return 0, db.ResourceTypeNotFoundError{Name: resourceTypeName}
	}

	interval, err := scanner.checkInterval(savedResourceType.CheckEvery())
	if err != nil {
		scanner.setCheckError(logger, savedResourceType, err)
		return 0, err
	}

	resourceTypes, err := scanner.dbPipeline.ResourceTypes()
	if err != nil {
		logger.Error("failed-to-get-resource-types", err)
		return 0, err
	}

	for _, parentType := range resourceTypes {
		if parentType.Name() == savedResourceType.Name() {
			continue
		}
		if parentType.Name() != savedResourceType.Type() {
			continue
		}

		version, err := parentType.Version()
		if err != nil {
			return 0, err
		}

		if version != nil {
			continue
		}

		if err = scanner.Scan(logger, parentType.Name()); err != nil {
			logger.Error("failed-to-scan-parent-resource-type-version", err)
			scanner.setCheckError(logger, savedResourceType, err)
			return 0, err
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

	source, err := creds.NewSource(scanner.variables, savedResourceType.Source()).Evaluate()
	if err != nil {
		logger.Error("failed-to-evaluate-resource-type-source", err)
		scanner.setCheckError(logger, savedResourceType, err)
		return 0, err
	}

	resourceConfigScope, err := savedResourceType.SetResourceConfig(
		logger,
		source,
		versionedResourceTypes.Without(savedResourceType.Name()),
	)
	if err != nil {
		logger.Error("failed-to-set-resource-config-id-on-resource-type", err)
		scanner.setCheckError(logger, savedResourceType, err)
		return 0, err
	}

	// Clear out the check error on the resource type
	scanner.setCheckError(logger, savedResourceType, err)

	reattempt := true
	for reattempt {
		reattempt = mustComplete
		lock, acquired, err := resourceConfigScope.AcquireResourceCheckingLock(
			logger,
			interval,
		)
		if err != nil {
			lockLogger.Error("failed-to-get-lock", err, lager.Data{
				"resource-type":      resourceTypeName,
				"resource-config-id": resourceConfigScope.ResourceConfig().ID(),
			})
			return interval, ErrFailedToAcquireLock
		}

		if !acquired {
			lockLogger.Debug("did-not-get-lock")
			if mustComplete {
				scanner.clock.Sleep(time.Second)
				continue
			} else {
				return interval, ErrFailedToAcquireLock
			}
		}

		defer lock.Release()

		updated, err := resourceConfigScope.UpdateLastChecked(interval, mustComplete)
		if err != nil {
			lockLogger.Error("failed-to-get-update-last-checked", err, lager.Data{
				"resource-type":      resourceTypeName,
				"resource-config-id": resourceConfigScope.ResourceConfig().ID(),
			})
			return interval, ErrFailedToAcquireLock
		}

		if !updated {
			lockLogger.Debug("did-not-update-last-checked")
			if mustComplete {
				scanner.clock.Sleep(time.Second)
				continue
			} else {
				return interval, ErrFailedToAcquireLock
			}
		}

		break
	}

	latestVersions, err := resourceConfigScope.LatestVersions()
	if err != nil {
		logger.Error("failed-to-get-current-version", err)
		return interval, err
	}

	latestFromVersions := make(map[atc.Space]atc.Version)
	for _, resourceConfigVersion := range latestVersions {
		latestFromVersions[resourceConfigVersion.Space()] = atc.Version(resourceConfigVersion.Version())
	}

	for space, version := range fromVersion {
		latestFromVersions[space] = version
	}

	return interval, scanner.check(
		logger,
		savedResourceType,
		resourceConfigScope,
		latestFromVersions,
		versionedResourceTypes,
		source,
		saveGiven,
	)
}

func (scanner *resourceTypeScanner) check(
	logger lager.Logger,
	savedResourceType db.ResourceType,
	resourceConfigScope db.ResourceConfigScope,
	fromVersion map[atc.Space]atc.Version,
	versionedResourceTypes creds.VersionedResourceTypes,
	source atc.Source,
	saveGiven bool,
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

	containerSpec := worker.ContainerSpec{
		ImageSpec: worker.ImageSpec{
			ResourceType: savedResourceType.Type(),
		},
		Tags:   savedResourceType.Tags(),
		TeamID: scanner.dbPipeline.TeamID(),
	}

	workerSpec := worker.WorkerSpec{
		ResourceType:  savedResourceType.Type(),
		Tags:          savedResourceType.Tags(),
		ResourceTypes: versionedResourceTypes.Without(savedResourceType.Name()),
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
		versionedResourceTypes.Without(savedResourceType.Name()),
		worker.NoopImageFetchingDelegate{},
	)
	if err != nil {
		chkErr := resourceConfigScope.SetCheckError(err)
		if chkErr != nil {
			logger.Error("failed-to-set-check-error-on-resource-config", chkErr)
		}
		logger.Error("failed-to-initialize-new-container", err)
		return err
	}

	tx, err := scanner.conn.Begin()
	if err != nil {
		return err
	}

	defer tx.Rollback()

	spaces := make(map[atc.Space]atc.Version)
	checkHandler := NewCheckEventHandler(logger, tx, resourceConfigScope, spaces)
	err = res.Check(lagerctx.NewContext(context.TODO(), logger), checkHandler, source, fromVersion)
	resourceConfigScope.SetCheckError(err)
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

func (scanner *resourceTypeScanner) checkInterval(checkEvery string) (time.Duration, error) {
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

func (scanner *resourceTypeScanner) setCheckError(logger lager.Logger, savedResourceType db.ResourceType, err error) {
	setErr := savedResourceType.SetCheckSetupError(err)
	if setErr != nil {
		logger.Error("failed-to-set-check-error", err)
	}
}
