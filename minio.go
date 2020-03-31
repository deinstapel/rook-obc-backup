package main

import (
  "fmt"
  "os/exec"
  "os"
  log "github.com/sirupsen/logrus"
)

var mcliBinary string
var remoteBucketSuffix string
var remoteBucketPrefix string
func init() {
  bin, err := exec.LookPath("mc")
  if err != nil {
    bin, err = exec.LookPath("mcli")
    if err != nil {
      log.Fatal("Failed to find minio client binary")
      os.Exit(1)
    }
  }
  mcliBinary = bin
}

func addMinioHost(name, url, accesskey, secretkey string) error {
  cmd := exec.Command(mcliBinary, "config", "host", "add", name, url, accesskey, secretkey, "--api=s3v4")
  cmd.Stdout = os.Stdout
  cmd.Stderr = os.Stderr
  return cmd.Run()
}

func setupMinioRemote() error {
  url, uok := os.LookupEnv("REMOTE_S3_HOST")
  accesskey, aok := os.LookupEnv("REMOTE_S3_ACCESS_KEY")
  secretkey, sok := os.LookupEnv("REMOTE_S3_SECRET_KEY")
  suffix, pok := os.LookupEnv("REMOTE_S3_BUCKET_SUFFIX")
  prefix, pfok := os.LookupEnv("REMOTE_S3_BUCKET_PREFIX")
  if !uok || !aok || !sok || !pok || !pfok {
    return fmt.Errorf("Invalid environment: url: %v, access: %v, secret: %v, suffix: %v, prefix: %v", uok, aok, sok, pok, pfok)
  }
  remoteBucketSuffix = suffix
  remoteBucketPrefix = prefix
  return addMinioHost("remote", url, accesskey, secretkey)
}

func mirrorBucket(sourceHost, sourceBucket, targetBucket string) error {
  cmd := exec.Command(
    mcliBinary,
    "mirror",
    "-q",
    "--overwrite",
    "--remove",
    fmt.Sprintf("%s/%s", sourceHost, sourceBucket),
    fmt.Sprintf("remote/%s-%s-%s", remoteBucketPrefix, targetBucket, remoteBucketSuffix),
  )
  cmd.Stdout = os.Stdout
  cmd.Stderr = os.Stderr
  return cmd.Run()
}
