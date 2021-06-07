package kubernetes

import (
	"github.com/ericchiang/k8s"
  metav1 "github.com/ericchiang/k8s/apis/meta/v1"
)

type ObjectBucketClaimSpec struct {
  BucketName string `json:"bucketName"`
  StorageClassName string `json:"storageClassName"`
}

// ObjectBucketClaim is a struct defining a CRD which is used to get
// claims for S3 buckets.
type ObjectBucketClaim struct {
  Kind       string             `json:"kind"`
  APIVersion string             `json:"apiVersion"`
  Metadata *metav1.ObjectMeta    `json:"metadata"`
  Spec     ObjectBucketClaimSpec `json:"spec"`
}

// GetMetadata gets the object metadata.
func(obc *ObjectBucketClaim) GetMetadata() *metav1.ObjectMeta {
  return obc.Metadata
}

// ObjectBucketClaimList is the corresponding list struct to
// ObjectBucketClaim
type ObjectBucketClaimList struct {
  Metadata *metav1.ListMeta `json:"metadata"`
  Items []ObjectBucketClaim `json:"items"`
}


// GetMetadata gets the object metadata.
func(e *ObjectBucketClaimList) GetMetadata() *metav1.ListMeta {
  return e.Metadata
}

func init() {
  k8s.Register("objectbucket.io", "v1alpha1", "objectbucketclaims", true, &ObjectBucketClaim{})
  k8s.RegisterList("objectbucket.io", "v1alpha1", "objectbucketclaims", true, &ObjectBucketClaimList{})
}
