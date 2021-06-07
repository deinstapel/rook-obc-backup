package env

import (
	"errors"
	log "github.com/sirupsen/logrus"
	"os"
)

type Environment struct {
	SOURCE_KUBECONFIG string
	SOURCE_S3_URL string
	TARGET_KUBECONFIG string
	TARGET_S3_URL string
	TARGET_BUCKET_PREFIX string
	TARGET_BUCKET_NAMESPACE string
	TARGET_STORAGE_CLASS_NAME string
}

func ReadEnv() Environment {

	SOURCE_KUBECONFIG, _ := os.LookupEnv("SOURCE_KUBECONFIG")
	SOURCE_S3_URL, hasSourceUrl := os.LookupEnv("SOURCE_S3_URL")
	TARGET_KUBECONFIG, _ := os.LookupEnv("TARGET_KUBECONFIG")
	TARGET_S3_URL, hasTargetUrl := os.LookupEnv("TARGET_S3_URL")
	TARGET_BUCKET_PREFIX, hasTargetBucketPrefix := os.LookupEnv("TARGET_BUCKET_PREFIX")
	TARGET_BUCKET_NAMESPACE, hasTargetBucketNamespace := os.LookupEnv("TARGET_BUCKET_NAMESPACE")
	TARGET_STORAGE_CLASS_NAME, hasTargetStorageClassName := os.LookupEnv("TARGET_STORAGE_CLASS_NAME")

	log.WithFields(log.Fields{
		"SOURCE_KUBECONFIG": SOURCE_KUBECONFIG,
		"SOURCE_S3_URL": SOURCE_S3_URL,
		"TARGET_KUBECONFIG": TARGET_KUBECONFIG,
		"TARGET_S3_URL": TARGET_S3_URL,
		"TARGET_BUCKET_PREFIX": TARGET_BUCKET_PREFIX,
		"TARGET_BUCKET_NAMESPACE": TARGET_BUCKET_NAMESPACE,
		"TARGET_STORAGE_CLASS_NAME": TARGET_STORAGE_CLASS_NAME,
	}).Info("Loaded Environment")

	if !hasSourceUrl ||
		!hasTargetUrl ||
		!hasTargetBucketPrefix ||
		!hasTargetBucketNamespace ||
		!hasTargetStorageClassName {
		panic(errors.New("Invalid Environment"))
	}


	e := Environment{
		SOURCE_KUBECONFIG: SOURCE_KUBECONFIG,
		SOURCE_S3_URL: SOURCE_S3_URL,
		TARGET_KUBECONFIG: TARGET_KUBECONFIG,
		TARGET_S3_URL: TARGET_S3_URL,
		TARGET_BUCKET_PREFIX: TARGET_BUCKET_PREFIX,
		TARGET_BUCKET_NAMESPACE: TARGET_BUCKET_NAMESPACE,
		TARGET_STORAGE_CLASS_NAME: TARGET_STORAGE_CLASS_NAME,
	}

	return e
}