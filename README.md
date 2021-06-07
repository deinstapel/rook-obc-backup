# rook-obc-backup

This project automates backups / replication of (rook) ObjectBucketClaims onto other rook Object Bucket Claim targets.
It is designed to be run as Kubernetes Cron Job.

## How it works

This tool fetches all OBC objects from the Kubernetes API, creates target OBCs on the target cluster and mirrors all files from the source cluster to the target cluster. It creates all OBCs at the target cluster within `TARGET_BUCKET_NAMESPACE` and uses the following naming scheme:
`${TARGET_BUCKET_PREFIX}-${OBC_NAMESPACE}-${OBC_NAME}`.

## Configuration

This program is configured via environment variables.

| Variable | Meaning | Example |
| -------- | ------- | ------- |
| SOURCE_S3_URL | Public reachable endpoint for downloading the S3 files | https://source.example.com
| SOURCE_KUBECONFIG | Path to kubeconfig for source cluster, empty for in-cluster-auth. | source_kubeconfig
| TARGET_S3_URL | Public reachable endpoint for uploading the S3 files | https://target.example.com
| TARGET_KUBECONFIG | Path to kubeconfig for target cluster, empty for in-cluster-auth. | target_kubeconfig
| TARGET_BUCKET_PREFIX | Prefix used for target buckets | prefix
| TARGET_BUCKET_NAMESPACE | Namespace for Rook OBC resources | rook-ceph
| TARGET_STORAGE_CLASS_NAME | Storage Class Name for Rook OBC resources | spinning-rust
