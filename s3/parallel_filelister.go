// Implements listing the content of an S3 bucket traversing the content tree in
// a parallel fashion.
package s3

import (
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pkg/errors"
)

// S3ListObjectsMaxKeys is the maximum number of keys the S3 ListObjects API
// will return.
const S3ListObjectsMaxKeys = 1000

type pathAndMd5 struct {
	path string
	md5  string
	err  error
}

type prefixAndDepth struct {
	prefix string
	depth  int
}

// ParallelFilelister holds the data structures necessary to list the bucket.
type ParallelFilelister struct {
	Paths                 []string
	Md5s                  []string
	Errs                  []error
	s3                    *s3.S3
	bucket                string
	rootPrefix            string
	pathAndMd5Chan        chan *pathAndMd5
	collectPathAndMd5Done chan bool
	prefixAndDepthChan    chan *prefixAndDepth
	wg                    sync.WaitGroup
	hidePlusWorkaround    bool
}

// StartNewParallelFilelister create a ParallelFilelister object and start
// traversing the bucket in the background with up to `parallelNumber`
// concurrent requests.
// If `parallelNumber` is less than 1, it will be set to 1.
// Invoke `WaitForCompletion()` to block until listing is completed.
// Paths and MD5s will be stored in the `Paths` amd `Md5s` fields of the
// `ParallelFilelister` object.
// Errors will be stored in the `Errs` field.
func StartNewParallelFilelister(
	s3 *s3.S3, bucket, rootPrefix string, parallelNumber int, hidePlusWorkaround bool,
) *ParallelFilelister {
	filelister := &ParallelFilelister{
		Paths:                 make([]string, 0, 1024),
		Md5s:                  make([]string, 0, 1024),
		s3:                    s3,
		bucket:                bucket,
		rootPrefix:            rootPrefix,
		pathAndMd5Chan:        make(chan *pathAndMd5),
		collectPathAndMd5Done: make(chan bool),
		prefixAndDepthChan:    make(chan *prefixAndDepth),
		hidePlusWorkaround:    hidePlusWorkaround,
	}

	go filelister.collectPathsAndMd5s()

	for i := 0; i < parallelNumber; i++ {
		go filelister.filelistWorkerLoop()
	}
	maxDepth := 0
	if parallelNumber > 1 {
		maxDepth = -1
		prefixParts := strings.Split(strings.TrimRight(rootPrefix, "/"), "/")
		for i := len(prefixParts) - 1; i >= 0; i-- {
			if prefixParts[i] == "dists" {
				maxDepth = 0
				break
			} else if prefixParts[i] == "pool" {
				maxDepth = 2 - (len(prefixParts) - 1 - i)
				if maxDepth < 0 {
					maxDepth = 0
				}
				break
			}
		}
	}

	filelister.listPrefix(rootPrefix, maxDepth)
	return filelister
}

// WaitForCompletion blocks until all requests have terminated and all paths
// (and errors) have been collected.
func (filelister *ParallelFilelister) WaitForCompletion() {
	filelister.wg.Wait()
	close(filelister.prefixAndDepthChan)
	close(filelister.pathAndMd5Chan)
	<-filelister.collectPathAndMd5Done
}

// listPrefix queues a prefix for parallel listing up to the specified depth.
func (filelister *ParallelFilelister) listPrefix(prefix string, maxDepth int) {
	filelister.wg.Add(1)
	select {
	case filelister.prefixAndDepthChan <- &prefixAndDepth{prefix, maxDepth}:
	default:
		// Channel is full, all workers are busy, list the prefix in
		// this worker to avoid deadlock.
		filelister.filelistWorker(prefix, maxDepth)
	}
}

// filelistWorker list the content of a prefix in the bucket.
// If maxDepth is == 0, it will list the whole bucket sequentially.
// If maxDepth is < 0, it will list common prefixes in parallel with no depth
// limit.
// If maxDepth is > 0, it will list common prefixes in parallel for the next
// maxDepth levels, and then list sequentially.
func (filelister *ParallelFilelister) filelistWorker(prefix string, maxDepth int) {
	defer filelister.wg.Done()

	params := &s3.ListObjectsInput{
		Bucket:  aws.String(filelister.bucket),
		Prefix:  aws.String(prefix),
		MaxKeys: aws.Int64(S3ListObjectsMaxKeys),
	}
	if maxDepth != 0 {
		if strings.HasSuffix(prefix, "dists/") {
			// Do not list the content of dists/ in parallel, as it can have
			// hundreds of subdirectories.
			maxDepth = 0
		}
		if strings.HasSuffix(prefix, "pool/") {
			// List in parallel up to pool/<component>/<initial>, as there could
			// be hundreds of directories inside that.
			maxDepth = 2
		}
	}
	if maxDepth != 0 {
		// Not setting Delimiter will cause the whole prefix to be listed.
		params.Delimiter = aws.String("/")
	}

	err := filelister.s3.ListObjectsPages(params, func(contents *s3.ListObjectsOutput, lastPage bool) bool {
		for _, key := range contents.Contents {
			if filelister.hidePlusWorkaround && strings.Contains(*key.Key, " ") {
				// if we use plusWorkaround, we want to hide those duplicates
				/// from listing
				continue
			}

			filelister.pathAndMd5Chan <- &pathAndMd5{
				path: *key.Key,
				md5:  strings.Replace(*key.ETag, "\"", "", -1),
			}

		}
		for _, c := range contents.CommonPrefixes {
			if c.Prefix != nil {
				filelister.listPrefix(*c.Prefix, maxDepth-1)
			}
		}

		return true
	})

	if err != nil {
		filelister.pathAndMd5Chan <- &pathAndMd5{
			err: errors.WithMessagef(err, "error listing under prefix %s in %s: %s", prefix, filelister.bucket, err),
		}
	}
}

// filelistWorkerLoop received new prefixes to list and invokes filelistWorker()
func (filelister *ParallelFilelister) filelistWorkerLoop() {
	for i := range filelister.prefixAndDepthChan {
		filelister.filelistWorker(i.prefix, i.depth)
	}
}

// collectPathsAndMd5s collects paths, md5s, and errors produeced by filelistWorker()
func (filelister *ParallelFilelister) collectPathsAndMd5s() {
	for i := range filelister.pathAndMd5Chan {
		if i.path != "" {
			filelister.Paths = append(filelister.Paths, i.path[len(filelister.rootPrefix):])
		}
		if i.md5 != "" {
			filelister.Md5s = append(filelister.Md5s, i.md5)
		}
		if i.err != nil {
			filelister.Errs = append(filelister.Errs, i.err)
		}
	}
	close(filelister.collectPathAndMd5Done)
}
