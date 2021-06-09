package sync

import (
	"github.com/larrabee/s3sync/pipeline"
	"github.com/larrabee/s3sync/storage"
)

// FilterObjectsModified accepts an input object and checks if it matches the filter
// This filter read object meta from target storage and compare object ETags. If Etags are equal object will be skipped
// For FS storage xattr support are required for proper work.
var AdvancedObjectFilter pipeline.StepFn = func(group *pipeline.Group, stepNum int, input <-chan *storage.Object, output chan<- *storage.Object, errChan chan<- error) {
	for obj := range input {
    // got an object from the input.
    // we download the metadata of the object in the destination
    // then we compare the ETags.

		destObj := &storage.Object{
			Key:       obj.Key,
			VersionId: obj.VersionId,
		}
		err := group.Target.GetObjectMeta(destObj)

    if err != nil || obj.ETag == nil {
      // Destination doesn't exist or object has no etag, so we force this object to be synced
      output <- obj
      continue
    }

    // We have a valid source ETag *and* the destination is already existing
    if (destObj.ETag != nil && *obj.ETag == *destObj.ETag) {
      // Destination object has the same ETag -> definitely matches
      continue
    }

    // Destination object exists, has an ETag which is different from the original
    // This happens for the case when the destination is uploaded in a different chunking than the original file
    // Therefore, we annotate the ETag into X-Original-ETag metadata and compare that again

    if origETag, ok := destObj.Metadata["X-Original-ETag"]; ok && origETag != nil && *obj.ETag == *origETag {
      // Destination object has the same ETag set in its metadata
      continue
    }
    // Annotate the original ETag into metadata, so we will not need to copy it again
    if obj.Metadata == nil {
      obj.Metadata = make(map[string]*string)
    }
    obj.Metadata["X-Original-ETag"] = obj.ETag
    output <- obj
	}
}
