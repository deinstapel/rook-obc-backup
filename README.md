# rook-obc-backup

This project automates backups / replication of (rook) ObjectBucketClaims onto other S3 targets.
It is designed to be run as Kubernetes Cron Job.

## How it works

This tool fetches all OBC objects from the K8S API, reads the credentials to access them and mirrors them to a configurable target.
To get a quickstart, edit deploy.yaml and apply it.

Source buckets are picked up by fetching the configmap and secrets of existing OBCs. The target buckets will be created using the following naming scheme:

`${REMOTE_S3_BUCKET_PREFIX}-${OBC_NAMESPACE}-${OBC_NAME}-${REMOTE_S3_BUCKET_SUFFIX}`.

## Configuration

This program is configured via environment variables.

| Variable | Meaning | Example |
| -------- | ------- | ------- |
| KUBECONFIG | kubeconfig file path for running outside of a cluster | `$(pwd)/kubeconfig`
| REMOTE_S3_HOST | S3 API Server to mirror the local buckets to | `https://s3.amazonaws.com`
| REMOTE_S3_ACCESS_KEY | S3 Access Key for the target server | `username`
| REMOTE_S3_SECRET_KEY | S3 Secret Key for the target server | `passw0rd`
| REMOTE_S3_BUCKET_PREFIX | Prefix to prepend to created buckets in the target | `ds`
| REMOTE_S3_BUCKET_SUFFIX | Suffix to append to created buckets in the target | `s3-backup`
| LOCAL_S3_URL | Override for the local S3 server in case your cluster domain is not reachable | `s3.yourdomain.de`
