package sync

import (
	"context"
	"errors"
	"time"

	"github.com/deinstapel/rook-obc-backup/kubernetes"
	"github.com/larrabee/s3sync/pipeline"
	"github.com/larrabee/s3sync/pipeline/collection"
	"github.com/larrabee/s3sync/storage"
	"github.com/larrabee/s3sync/storage/s3stream"
	log "github.com/sirupsen/logrus"
)

const KEYS_PER_REQ = 1000

func PrepareSyncGroup(
	ctx context.Context,
	source *kubernetes.BucketDetails,
	sourceEndpoint string,
	target *kubernetes.BucketDetails,
	targetEndpoint string,
	numWorkers int,
) (pipeline.Group, error) {

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
		0,
		0,
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
		0,
		0,
	)

	sourceStorage.WithContext(ctx)

	syncGroup.SetSource(sourceStorage)
	syncGroup.SetTarget(targetStorage)

	syncGroup.AddPipeStep(pipeline.Step{
		Name:     "ListSource",
		Fn:       collection.ListSourceStorage,
		ChanSize: KEYS_PER_REQ,
	})

	syncGroup.AddPipeStep(pipeline.Step{
		Name: "FilterObjectsModified",
		Fn:   collection.FilterObjectsModified,
	})

	syncGroup.AddPipeStep(pipeline.Step{
		Name:       "LoadObjData",
		Fn:         collection.LoadObjectData,
		AddWorkers: uint(numWorkers),
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

	return syncGroup, nil
}

func printLiveStats(ctx context.Context, name string, syncGroup *pipeline.Group) {
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
					"InputObjSpeed":  float64(val.Stats.Input) / dur,
					"OutputObjSpeed": float64(val.Stats.Output) / dur,
				}).Info("Current Group")
			}
			time.Sleep(60 * time.Second)
		}
	}
}

func printFinalStats(name string, syncGroup *pipeline.Group) {
	dur := time.Since(syncGroup.StartTime).Seconds()
	for _, val := range syncGroup.GetStepsInfo() {
		log.WithFields(log.Fields{
			"name":           name,
			"stepNum":        val.Num,
			"stepName":       val.Name,
			"InputObj":       val.Stats.Input,
			"OutputObj":      val.Stats.Output,
			"ErrorObj":       val.Stats.Error,
			"InputObjSpeed":  float64(val.Stats.Input) / dur,
			"OutputObjSpeed": float64(val.Stats.Output) / dur,
		}).Info("Pipeline step finished")
	}
	log.WithFields(log.Fields{
		"durationSec": time.Since(syncGroup.StartTime).Seconds(),
	}).Infof("Duration: %s", time.Since(syncGroup.StartTime).String())

}

func RunSyncGroup(ctx context.Context, name string, syncGroup pipeline.Group) error {

	syncGroup.Run()

	go printLiveStats(ctx, name, &syncGroup)

WaitLoop:
	for {
		select {

		case err := <-syncGroup.ErrChan():
			if err == nil {
				break WaitLoop
			}

			var confErr *pipeline.StepConfigurationError
			if errors.As(err, &confErr) {
				log.Errorf("Pipeline configuration error: %s, terminating", confErr)
				return err
			}

			if storage.IsErrNotExist(err) {
				var objErr *pipeline.ObjectError
				if errors.As(err, &objErr) {
					log.Warnf("Skip missing object: %s", *objErr.Object.Key)
				} else {
					log.Warnf("Skip missing object, err: %s", err)
				}
				continue WaitLoop
			} else if storage.IsErrPermission(err) {
				var objErr *pipeline.ObjectError
				if errors.As(err, &objErr) {
					log.Warnf("Skip permission denied object: %s", *objErr.Object.Key)
				} else {
					log.Warnf("Skip permission denied object, err: %s", err)
				}
				continue WaitLoop
			}

			return err
		}
	}

	printFinalStats(name, &syncGroup)
	return nil
}
