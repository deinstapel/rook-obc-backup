package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/deinstapel/rook-obc-backup/env"
	"github.com/deinstapel/rook-obc-backup/kubernetes"
	"github.com/deinstapel/rook-obc-backup/sync"
	"github.com/ericchiang/k8s"
	"github.com/ghodss/yaml"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func init() {
	log.SetFormatter(&log.TextFormatter{})
	log.SetOutput(os.Stderr)
	log.SetLevel(log.TraceLevel)
}

func makeKubeconfigClient(path string) (*k8s.Client, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	config := new(k8s.Config)
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, err
	}
	client, err := k8s.NewClient(config)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func makeClient(path string) (*k8s.Client, error) {
	if path != "" {
		return makeKubeconfigClient(path)
	}
	return k8s.NewInClusterClient()
}

func handleMainError(err error) {
	if err != nil {
		log.Error(err)
		panic(err)
	}
}

func main() {

	readEnv := env.ReadEnv()

	sourceClient, err := makeClient(readEnv.SOURCE_KUBECONFIG)
	handleMainError(err)
	targetClient, err := makeClient(readEnv.TARGET_KUBECONFIG)
	handleMainError(err)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		signalChannel := make(chan os.Signal, 1)
		signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)
		<-signalChannel
		cancel()
	}()

	if err := backupOBCs(ctx, sourceClient, targetClient, readEnv); err != nil {
		log.WithField("err", err).Fatal("Backup failed")
		os.Exit(1)
	}
	os.Exit(0)
}

func backupOBCs(ctx context.Context, sourceClient, targetClient *k8s.Client, env env.Environment) error {
	log.Trace("start crd watch")

	crdList := &kubernetes.ObjectBucketClaimList{}
	if err := sourceClient.List(ctx, k8s.AllNamespaces, crdList); err != nil {
		log.WithField("err", err).Warning("Failed to list OBCs")
		return err
	}

	var failedBuckets []string

	for index := range crdList.Items {
		item := crdList.Items[index]
		err := backupOBC(ctx, item, sourceClient, targetClient, env)

		if err != nil {
			log.WithFields(log.Fields{"name": *item.Metadata.Name, "namespace": *item.Metadata.Namespace}).Error(err)
			failedBuckets = append(failedBuckets, *item.Metadata.Name)
			continue
		}

	}

	if len(failedBuckets) > 0 {
		return fmt.Errorf("OBCs %v failed", failedBuckets)
	}
	return nil
}

func backupOBC(ctx context.Context, objectBucketClaim kubernetes.ObjectBucketClaim, sourceClient, targetClient *k8s.Client, env env.Environment) error {

	item := objectBucketClaim

	targetBucketName := env.TARGET_BUCKET_PREFIX + "-" + *item.Metadata.Namespace + "-" + *item.Metadata.Name

	scopedLog := log.WithFields(log.Fields{
		"sourceName":      *item.Metadata.Name,
		"sourceNamespace": *item.Metadata.Namespace,
		"targetName": targetBucketName,
		"targetNamespace": env.TARGET_BUCKET_NAMESPACE,
	})
	scopedLog.Info("Working on Element")

	childCtx, cancelChildCtx := context.WithCancel(ctx)
	defer cancelChildCtx()

	sourceDetails, err := kubernetes.GetDetails(
		sourceClient,
		childCtx,
		*item.Metadata.Namespace,
		*item.Metadata.Name,
	)
	if err != nil {
		log.Errorf("sourceDetails could not be obtained, err: %v", err)
		return errors.New("sourceDetails could not be obtained")
	}

	doesTargetNamespaceExist, err := kubernetes.DoesExist(
		targetClient,
		childCtx,
		env.TARGET_BUCKET_NAMESPACE,
		targetBucketName,
	)

	if err != nil {
		return errors.New("Target namespace could not be checked")
	}

	if !doesTargetNamespaceExist {
		err := kubernetes.CreateOBC(
			targetClient,
			childCtx,
			env.TARGET_BUCKET_NAMESPACE,
			targetBucketName,
			env.TARGET_STORAGE_CLASS_NAME,
		)

		if err != nil {
			log.Errorf("Target object bucket claim could not be created, err: %v", err)
			return errors.New("Target object bucket claim could not be created")
		}

		time.Sleep(10 * time.Second)
	}

	targetDetails, err := kubernetes.GetDetails(
		targetClient,
		childCtx,
		env.TARGET_BUCKET_NAMESPACE,
		targetBucketName,
	)

	if err != nil {
		return errors.New("targetDetails could not be obtained")
	}

	syncGroup, err := sync.PrepareSyncGroup(
		childCtx,
		sourceDetails,
		env.SOURCE_S3_URL,
		targetDetails,
		env.TARGET_S3_URL,
	)

	if err != nil {
		return errors.New("Sync Group could not be prepared")
	}

	err = sync.RunSyncGroup(
		childCtx,
		targetBucketName,
		syncGroup,
	)

	if err != nil {
		return err
	}

	return nil
}
