package main

import (
  "context"
  "io/ioutil"
  "os"
  "fmt"
  log "github.com/sirupsen/logrus"
  "github.com/ericchiang/k8s"
  "github.com/ghodss/yaml"
  corev1 "github.com/ericchiang/k8s/apis/core/v1"
)
var eg SignalledErrGroup
var client *k8s.Client

func init() {
  log.SetFormatter(&log.TextFormatter{})
  log.SetOutput(os.Stderr)
  log.SetLevel(log.TraceLevel)
  eg = buildErrGroup(context.Background())
  c, err := makeClient()
  if err != nil {
    log.WithField("err", err).Fatal("Failed to create cluster client")
    os.Exit(1)
  }
  client = c
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

func makeClient() (*k8s.Client, error) {
	if kubeconfig := os.Getenv("KUBECONFIG"); kubeconfig != "" {
		return makeKubeconfigClient(kubeconfig)
	}
	return k8s.NewInClusterClient()
}

func main() {
  if err := setupMinioRemote(); err != nil {
    log.WithField("err", err).Fatal("Failed to setup remote s3 server")
    os.Exit(1)
  }
  eg.Go(backupOBCs)

  if err := eg.Wait(); err != nil {
    log.WithField("err", err).Fatal("Backup failed")
    os.Exit(1)
  }
}

func backupOBCs(ctx context.Context) error {
  log.Trace("start crd watch")
  crdList := &ObjectBucketClaimList{}
  if err := client.List(ctx, k8s.AllNamespaces, crdList); err != nil {
    log.WithField("err", err).Warning("Failed to list OBCs")
    return err
  }

  var failedBuckets []string

  for index := range crdList.Items {
    item := crdList.Items[index]
    scopedLog := log.WithFields(log.Fields{
      "claim": *item.Metadata.Name,
      "namespace": *item.Metadata.Namespace,
    })
    scopedLog.Trace("fetch creds");

    cm := &corev1.ConfigMap{}
    sec := &corev1.Secret{}
    if err := client.Get(ctx, *item.Metadata.Namespace, *item.Metadata.Name, cm); err != nil {
      failedBuckets = append(failedBuckets, *item.Metadata.Name)
      continue
    }
    if err := client.Get(ctx, *item.Metadata.Namespace, *item.Metadata.Name, sec); err != nil {
      scopedLog.WithField("err", err).Warning("secret get failed")
      failedBuckets = append(failedBuckets, *item.Metadata.Name)
      continue
    }
    minioHostName := fmt.Sprintf("%s-%s", *item.Metadata.Namespace, *item.Metadata.Name)
    cmData := cm.GetData()
    secData := sec.GetData()
    proto := "http"
    if cmData["BUCKET_SSL"] == "true" {
      proto = "https"
    }
    minioHostURL := fmt.Sprintf("%s://%s:%s", proto, cmData["BUCKET_HOST"], cmData["BUCKET_PORT"])
    if localURL, ok := os.LookupEnv("LOCAL_S3_URL"); ok {
      minioHostURL = localURL
    }
    if err := addMinioHost(minioHostName, minioHostURL, string(secData["AWS_ACCESS_KEY_ID"]), string(secData["AWS_SECRET_ACCESS_KEY"])); err != nil {
      scopedLog.WithField("err", err).Warning("Failed to set minio host")
      failedBuckets = append(failedBuckets, *item.Metadata.Name)
      continue
    }
    if err := mirrorBucket(minioHostName, cmData["BUCKET_NAME"], minioHostName); err != nil {
      scopedLog.WithField("err", err).Warning("Failed to mirror bucket")
      failedBuckets = append(failedBuckets, *item.Metadata.Name)
      continue

    }
  }
  if len(failedBuckets) > 0 {
    return fmt.Errorf("Backup for buckets %v failed", failedBuckets)
  }
  return nil
}
