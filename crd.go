package main

import (
	"github.com/ericchiang/k8s"
  metav1 "github.com/ericchiang/k8s/apis/meta/v1"
)

// ObjectBucketClaim is a struct defining a CRD which is used to get
// claims for S3 buckets.
type ObjectBucketClaim struct {
  Metadata *metav1.ObjectMeta `json:"metadata"`
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
  k8s.Register("objectbucket.io", "v1alpha1", "objectbucketclaims", false, &ObjectBucketClaim{})
  k8s.RegisterList("objectbucket.io", "v1alpha1", "objectbucketclaims", false, &ObjectBucketClaimList{})
}
