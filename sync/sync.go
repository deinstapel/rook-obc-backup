package sync

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/deinstapel/rook-obc-backup/kubernetes"
	"github.com/larrabee/s3sync/pipeline"
	"github.com/larrabee/s3sync/pipeline/collection"
	"github.com/larrabee/s3sync/storage"
	"github.com/larrabee/s3sync/storage/s3stream"
	log "github.com/sirupsen/logrus"
)

const KEYS_PER_REQ = 1000

type SyncGroup struct {
	pipeline.Group
	retryChan chan *storage.Object
}

func PrepareSyncGroup(
	ctx context.Context,
	source *kubernetes.BucketDetails,
	sourceEndpoint string,
	target *kubernetes.BucketDetails,
	targetEndpoint string,
	numWorkers int,
	retryFiles []*storage.Object,
) (*SyncGroup, error) {

	syncGroup := pipeline.NewGroup()

	sourceStorage := s3stream.NewS3StreamStorage(
		false,
		source.AccessKeyId,
		source.SecretAccessKey,
		"",
		source.BucketRegion,
		sourceEndpoint,
		source.BucketName,
		"",
		KEYS_PER_REQ,
		3,
		1*time.Second,
	)

	targetStorage := s3stream.NewS3StreamStorage(
		false,
		target.AccessKeyId,
		target.SecretAccessKey,
		"",
		target.BucketRegion,
		targetEndpoint,
		target.BucketName,
		"",
		KEYS_PER_REQ,
		3,
		1*time.Second,
	)

	retryChan := make(chan *storage.Object, 32)

	sourceStorage.WithContext(ctx)

	syncGroup.SetSource(sourceStorage)
	syncGroup.SetTarget(targetStorage)

	if retryFiles == nil {
		syncGroup.AddPipeStep(pipeline.Step{
			Name:     "ListSource",
			Fn:       collection.ListSourceStorage,
			ChanSize: KEYS_PER_REQ,
		})
	} else {
		syncGroup.AddPipeStep(pipeline.Step{
			Name:     "ListRetryFiles",
			Fn:       ListRetryFiles(retryFiles),
			ChanSize: KEYS_PER_REQ,
		})
	}

	syncGroup.AddPipeStep(pipeline.Step{
		Name:       "FilterObjectsModified",
		Fn:         AdvancedObjectFilter,
		AddWorkers: uint(numWorkers),
	})

	syncGroup.AddPipeStep(pipeline.Step{
		Name:       "LoadObjData",
		Fn:         collection.LoadObjectData,
		AddWorkers: uint(numWorkers),
	})

	syncGroup.AddPipeStep(pipeline.Step{
		Name: "AnnotateETag",
		Fn:   AnnotateETag,
	})

	syncGroup.AddPipeStep(pipeline.Step{
		Name:       "UploadObj",
		Fn:         collection.UploadObjectData,
		AddWorkers: uint(numWorkers),
	})

	syncGroup.AddPipeStep(pipeline.Step{
		Name: "Terminator",
		Fn:   collection.Terminator,
	})

	return &SyncGroup{
		Group:     syncGroup,
		retryChan: retryChan,
	}, nil
}

func printLiveStats(ctx context.Context, name string, syncGroup *SyncGroup) {
	sleeptime := 60 * time.Second
	if stimeStr, ok := os.LookupEnv("STATS_INTERVAL"); ok {
		if stime, err := time.ParseDuration(stimeStr); err == nil {
			sleeptime = stime
		}
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
			dur := time.Since(syncGroup.StartTime).Seconds()
			for _, val := range syncGroup.GetStepsInfo() {
				log.WithFields(log.Fields{
					"name":           name,
					"stepNum":        val.Num,
					"stepName":       val.Name,
					"InputObj":       val.Stats.Input,
					"OutputObj":      val.Stats.Output,
					"ErrorObj":       val.Stats.Error,
					"InputObjSpeed":  float64(val.Stats.Input.Load()) / dur,
					"OutputObjSpeed": float64(val.Stats.Output.Load()) / dur,
				}).Info("Current Group")
			}
			time.Sleep(sleeptime)
		}
	}
}

func printFinalStats(name string, syncGroup *SyncGroup) {
	dur := time.Since(syncGroup.StartTime).Seconds()
	for _, val := range syncGroup.GetStepsInfo() {
		log.WithFields(log.Fields{
			"name":           name,
			"stepNum":        val.Num,
			"stepName":       val.Name,
			"InputObj":       val.Stats.Input,
			"OutputObj":      val.Stats.Output,
			"ErrorObj":       val.Stats.Error,
			"InputObjSpeed":  float64(val.Stats.Input.Load()) / dur,
			"OutputObjSpeed": float64(val.Stats.Output.Load()) / dur,
		}).Info("Pipeline step finished")
	}
	log.WithFields(log.Fields{
		"durationSec": time.Since(syncGroup.StartTime).Seconds(),
	}).Infof("Duration: %s", time.Since(syncGroup.StartTime).String())

}

func RunSyncGroup(ctx context.Context, name string, syncGroup *SyncGroup) ([]*storage.Object, error) {
	retryFiles := []*storage.Object{}

	syncGroup.Run()

	go printLiveStats(ctx, name, syncGroup)

	var lastErr error

	for err := range syncGroup.ErrChan() {
		if err == nil {
			break
		}

		var confErr *pipeline.StepConfigurationError
		if errors.As(err, &confErr) {
			log.Errorf("Pipeline configuration error: %s, terminating", confErr)
			return nil, err
		}
		var objectErr *pipeline.ObjectError
		if errors.As(err, &objectErr) {
			log.Errorf("Failed downloading object: %s, retrying", *objectErr.Object.Key)
			retryFiles = append(retryFiles, objectErr.Object)
			continue
		}

		if storage.IsErrNotExist(err) {
			var objErr *pipeline.ObjectError
			if errors.As(err, &objErr) {
				log.Warnf("Skip missing object: %s", *objErr.Object.Key)
			} else {
				log.Warnf("Skip missing object, err: %s", err)
			}
			continue
		} else if storage.IsErrPermission(err) {
			var objErr *pipeline.ObjectError
			if errors.As(err, &objErr) {
				log.Warnf("Skip permission denied object: %s", *objErr.Object.Key)
			} else {
				log.Warnf("Skip permission denied object, err: %s", err)
			}
			continue
		}

		lastErr = err
		log.Warnf("Skip error: %v", err)
	}

	printFinalStats(name, syncGroup)
	return retryFiles, lastErr
}
