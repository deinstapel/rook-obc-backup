package kubernetes

import (
	"context"
	"errors"
	"github.com/ericchiang/k8s"
	v1 "github.com/ericchiang/k8s/apis/core/v1"
	metav1 "github.com/ericchiang/k8s/apis/meta/v1"

	"net/http"
)

type BucketDetails struct {
	AccessKeyId string
	SecretAccessKey string

	BucketName string
	BucketRegion string
}

func DoesExist(client *k8s.Client, ctx context.Context, namespace string, name string) (bool, error) {

	var configMap v1.ConfigMap
	err := client.Get(ctx, namespace, name, &configMap)
	if err != nil {
		if apiErr, ok := err.(*k8s.APIError); ok {
			// Resource already exists. Carry on.
			if apiErr.Code == http.StatusNotFound {
				return false, nil
			}
		}
		return false, err
	}

	return true, nil

}

func GetDetails(client *k8s.Client, ctx context.Context, namespace string, name string) (*BucketDetails, error) {

	var configMap v1.ConfigMap
	err := client.Get(ctx, namespace, name, &configMap)
	if err != nil { return nil, err }

	var secret v1.Secret
	err = client.Get(ctx, namespace, name, &secret )
	if err != nil { return nil, err }

	configMapData := configMap.GetData()
	secretData := secret.GetData()

	accessKeyIdRaw, hasAccessKeyId := secretData["AWS_ACCESS_KEY_ID"]
	secretAccessKeyRaw, hasSecretAccessKey := secretData["AWS_SECRET_ACCESS_KEY"]
	bucketName, hasBucketName := configMapData["BUCKET_NAME"]
	bucketRegion, hasBucketRegion := configMapData["BUCKET_REGION"]

	accessKeyId := string(accessKeyIdRaw)
	secretAccessKey := string(secretAccessKeyRaw)

	if !hasAccessKeyId || !hasSecretAccessKey || !hasBucketName || !hasBucketRegion {
		return nil, errors.New("Invalid Config Map Or Secrets")
	}

	bD := BucketDetails{
		AccessKeyId:     accessKeyId,
		SecretAccessKey: secretAccessKey,
		BucketName:      bucketName,
		BucketRegion:    bucketRegion,
	}

	return &bD, nil

}

func CreateOBC(client *k8s.Client, ctx context.Context, namespace string, name string, storageClassName string) error {

	oBC := ObjectBucketClaim{
		Kind: "ObjectBucketClaim",
		APIVersion: "objectbucket.io/v1alpha1",
		Metadata: &metav1.ObjectMeta{
			Name: &name,
			Namespace: &namespace,
		},
		Spec: ObjectBucketClaimSpec{
			BucketName:       name,
			StorageClassName: storageClassName,
		},

	}

	return client.Create(ctx, &oBC)
}