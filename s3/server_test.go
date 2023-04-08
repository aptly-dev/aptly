package s3

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const debug = true

type s3Error struct {
	statusCode int
	XMLName    struct{} `xml:"Error"`
	Code       string
	Message    string
	BucketName string
	RequestID  string
	HostID     string
}

type action struct {
	srv   *Server
	w     http.ResponseWriter
	req   *http.Request
	reqID string
}

// Config controls the internal behaviour of the Server. A nil config is the default
// and behaves as if all configurations assume their default behaviour. Once passed
// to NewServer, the configuration must not be modified.
type Config struct {
	// Send409Conflict controls how the Server will respond to calls to PUT on a
	// previously existing bucket. The default is false, and corresponds to the
	// us-east-1 s3 enpoint. Setting this value to true emulates the behaviour of
	// all other regions.
	// http://docs.amazonwebservices.com/AmazonS3/latest/API/ErrorResponses.html
	Send409Conflict bool
}

func (c *Config) send409Conflict() bool {
	if c != nil {
		return c.Send409Conflict
	}
	return false
}

// Request stores the method and URI of an HTTP request.
type Request struct {
	Method     string
	RequestURI string
}

// Server is a fake S3 server for testing purposes.
// All of the data for the server is kept in memory.
type Server struct {
	url      string
	reqID    int
	listener net.Listener
	mu       sync.Mutex
	buckets  map[string]*bucket
	config   *Config
	// Requests holds a log of all requests received by the server.
	Requests []Request
}

type bucket struct {
	name    string
	acl     string
	objects map[string]*object
}

type object struct {
	name     string
	mtime    time.Time
	meta     http.Header // metadata to return with requests.
	checksum []byte      // also held as Content-MD5 in meta.
	data     []byte
}

// A resource encapsulates the subject of an HTTP request.
// The resource referred to may or may not exist
// when the request is made.
type resource interface {
	put(a *action) interface{}
	get(a *action) interface{}
	post(a *action) interface{}
	delete(a *action) interface{}
}

func NewServer(config *Config) (*Server, error) {
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return nil, fmt.Errorf("cannot listen on localhost: %v", err)
	}
	srv := &Server{
		listener: l,
		url:      "http://" + l.Addr().String(),
		buckets:  make(map[string]*bucket),
		config:   config,
	}
	go http.Serve(l, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		srv.serveHTTP(w, req)
	}))
	return srv, nil
}

// Quit closes down the server.
func (srv *Server) Quit() {
	srv.listener.Close()
}

// URL returns a URL for the server.
func (srv *Server) URL() string {
	return srv.url
}

func fatalError(code int, codeStr string, errf string, a ...interface{}) {
	panic(&s3Error{
		statusCode: code,
		Code:       codeStr,
		Message:    fmt.Sprintf(errf, a...),
	})
}

// serveHTTP serves the S3 protocol.
func (srv *Server) serveHTTP(w http.ResponseWriter, req *http.Request) {
	// ignore error from ParseForm as it's usually spurious.
	req.ParseForm()

	srv.mu.Lock()
	defer srv.mu.Unlock()

	if debug {
		log.Printf("s3test %q %q", req.Method, req.URL)
	}
	srv.Requests = append(srv.Requests, Request{req.Method, req.RequestURI})
	a := &action{
		srv:   srv,
		w:     w,
		req:   req,
		reqID: fmt.Sprintf("%09X", srv.reqID),
	}
	srv.reqID++

	var r resource
	defer func() {
		switch err := recover().(type) {
		case *s3Error:
			switch r := r.(type) {
			case objectResource:
				err.BucketName = r.bucket.name
			case bucketResource:
				err.BucketName = r.name
			}
			err.RequestID = a.reqID
			// TODO HostId
			w.Header().Set("Content-Type", `xml version="1.0" encoding="UTF-8"`)
			w.WriteHeader(err.statusCode)
			xmlMarshal(w, err)
		case nil:
		default:
			panic(err)
		}
	}()

	r = srv.resourceForURL(req.URL)

	var resp interface{}
	switch req.Method {
	case "PUT":
		resp = r.put(a)
	case "GET", "HEAD":
		resp = r.get(a)
	case "DELETE":
		resp = r.delete(a)
	case "POST":
		resp = r.post(a)
	default:
		fatalError(400, "MethodNotAllowed", "unknown http request method %q", req.Method)
	}
	if resp != nil && req.Method != "HEAD" {
		xmlMarshal(w, resp)
	}
}

// xmlMarshal is the same as xml.Marshal except that
// it panics on error. The marshalling should not fail,
// but we want to know if it does.
func xmlMarshal(w io.Writer, x interface{}) {
	if err := xml.NewEncoder(w).Encode(x); err != nil {
		panic(fmt.Errorf("error marshalling %#v: %v", x, err))
	}
}

// In a fully implemented test server, each of these would have
// its own resource type.
var unimplementedBucketResourceNames = map[string]bool{
	"acl":            true,
	"lifecycle":      true,
	"policy":         true,
	"location":       true,
	"logging":        true,
	"notification":   true,
	"versions":       true,
	"requestPayment": true,
	"versioning":     true,
	"website":        true,
	"uploads":        true,
}

var unimplementedObjectResourceNames = map[string]bool{
	"uploadId": true,
	"acl":      true,
	"torrent":  true,
	"uploads":  true,
}

var pathRegexp = regexp.MustCompile("/(([^/]+)(/(.*))?)?")

// resourceForURL returns a resource object for the given URL.
func (srv *Server) resourceForURL(u *url.URL) (r resource) {

	if u.Path == "/" {
		return serviceResource{
			buckets: srv.buckets,
		}
	}

	m := pathRegexp.FindStringSubmatch(u.Path)
	if m == nil {
		fatalError(404, "InvalidURI", "Couldn't parse the specified URI")
	}
	bucketName := m[2]
	objectName := m[4]
	if bucketName == "" {
		return nullResource{} // root
	}
	b := bucketResource{
		name:   bucketName,
		bucket: srv.buckets[bucketName],
	}
	q := u.Query()
	if objectName == "" {
		for name := range q {
			if unimplementedBucketResourceNames[name] {
				return nullResource{}
			}
		}
		return b

	}
	if b.bucket == nil {
		fatalError(404, "NoSuchBucket", "The specified bucket does not exist")
	}
	objr := objectResource{
		name:    objectName,
		version: q.Get("versionId"),
		bucket:  b.bucket,
	}
	for name := range q {
		if unimplementedObjectResourceNames[name] {
			return nullResource{}
		}
	}
	if obj := objr.bucket.objects[objr.name]; obj != nil {
		objr.object = obj
	}
	return objr
}

// nullResource has error stubs for all resource methods.
type nullResource struct{}

func notAllowed() interface{} {
	fatalError(400, "MethodNotAllowed", "The specified method is not allowed against this resource")
	return nil
}

func (nullResource) put(a *action) interface{}    { return notAllowed() }
func (nullResource) get(a *action) interface{}    { return notAllowed() }
func (nullResource) post(a *action) interface{}   { return notAllowed() }
func (nullResource) delete(a *action) interface{} { return notAllowed() }

const timeFormat = "2006-01-02T15:04:05Z"

type serviceResource struct {
	buckets map[string]*bucket
}

func (serviceResource) put(a *action) interface{}    { return notAllowed() }
func (serviceResource) post(a *action) interface{}   { return notAllowed() }
func (serviceResource) delete(a *action) interface{} { return notAllowed() }

// GET on an s3 service lists the buckets.
// http://docs.aws.amazon.com/AmazonS3/latest/API/RESTServiceGET.html
func (r serviceResource) get(a *action) interface{} {
	type respBucket struct {
		Name string
	}

	type response struct {
		Buckets []respBucket `xml:">Bucket"`
	}

	resp := response{}

	for _, bucketPtr := range r.buckets {
		bkt := respBucket{
			Name: bucketPtr.name,
		}
		resp.Buckets = append(resp.Buckets, bkt)
	}

	return &resp
}

type bucketResource struct {
	name   string
	bucket *bucket // non-nil if the bucket already exists.
}

type Owner struct {
	ID          string
	DisplayName string
}

type CommonPrefix struct {
	Prefix string
}

// The ListResp type holds the results of a List bucket operation.
type ListResp struct {
	Name       string
	Prefix     string
	Delimiter  string
	Marker     string
	NextMarker string
	MaxKeys    int
	// IsTruncated is true if the results have been truncated because
	// there are more keys and prefixes than can fit in MaxKeys.
	// N.B. this is the opposite sense to that documented (incorrectly) in
	// http://goo.gl/YjQTc
	IsTruncated    bool
	Contents       []Key
	CommonPrefixes []CommonPrefix
}

// The Key type represents an item stored in an S3 bucket.
type Key struct {
	Key          string
	LastModified string
	Size         int64
	// ETag gives the hex-encoded MD5 sum of the contents,
	// surrounded with double-quotes.
	ETag         string
	StorageClass string
	Owner        Owner
}

// GET on a bucket lists the objects in the bucket.
// http://docs.amazonwebservices.com/AmazonS3/latest/API/RESTBucketGET.html
func (r bucketResource) get(a *action) interface{} {
	if r.bucket == nil {
		fatalError(404, "NoSuchBucket", "The specified bucket does not exist")
	}
	delimiter := a.req.Form.Get("delimiter")
	marker := a.req.Form.Get("marker")
	maxKeys := -1
	if s := a.req.Form.Get("max-keys"); s != "" {
		i, err := strconv.Atoi(s)
		if err != nil || i < 0 {
			fatalError(400, "invalid value for max-keys: %q", s)
		}
		maxKeys = i
	}
	prefix := a.req.Form.Get("prefix")
	a.w.Header().Set("Content-Type", "application/xml")

	if a.req.Method == "HEAD" {
		return nil
	}

	var objs orderedObjects

	// first get all matching objects and arrange them in alphabetical order.
	for name, obj := range r.bucket.objects {
		if strings.HasPrefix(name, prefix) {
			objs = append(objs, obj)
		}
	}
	sort.Sort(objs)

	if maxKeys <= 0 {
		maxKeys = 1000
	}
	resp := &ListResp{
		Name:      r.bucket.name,
		Prefix:    prefix,
		Delimiter: delimiter,
		Marker:    marker,
		MaxKeys:   maxKeys,
	}

	var prefixes []CommonPrefix
	for _, obj := range objs {
		if !strings.HasPrefix(obj.name, prefix) {
			continue
		}
		name := obj.name
		isPrefix := false
		if delimiter != "" {
			if i := strings.Index(obj.name[len(prefix):], delimiter); i >= 0 {
				name = obj.name[:len(prefix)+i+len(delimiter)]
				if prefixes != nil && prefixes[len(prefixes)-1].Prefix == name {
					continue
				}
				isPrefix = true
			}
		}
		if name <= marker {
			continue
		}
		if len(resp.Contents)+len(prefixes) >= maxKeys {
			resp.IsTruncated = true
			break
		}
		if isPrefix {
			prefixes = append(prefixes, CommonPrefix{name})
		} else {
			// Contents contains only keys not found in CommonPrefixes
			resp.Contents = append(resp.Contents, obj.s3Key())
		}
	}
	resp.CommonPrefixes = prefixes
	return resp
}

// orderedObjects holds a slice of objects that can be sorted
// by name.
type orderedObjects []*object

func (s orderedObjects) Len() int {
	return len(s)
}
func (s orderedObjects) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s orderedObjects) Less(i, j int) bool {
	return s[i].name < s[j].name
}

func (obj *object) s3Key() Key {
	return Key{
		Key:          obj.name,
		LastModified: obj.mtime.Format(timeFormat),
		Size:         int64(len(obj.data)),
		ETag:         fmt.Sprintf(`"%x"`, obj.checksum),
		// TODO StorageClass
		// TODO Owner
	}
}

// DELETE on a bucket deletes the bucket if it's not empty.
func (r bucketResource) delete(a *action) interface{} {
	b := r.bucket
	if b == nil {
		fatalError(404, "NoSuchBucket", "The specified bucket does not exist")
	}
	if len(b.objects) > 0 {
		fatalError(400, "BucketNotEmpty", "The bucket you tried to delete is not empty")
	}
	delete(a.srv.buckets, b.name)
	return nil
}

// PUT on a bucket creates the bucket.
// http://docs.amazonwebservices.com/AmazonS3/latest/API/RESTBucketPUT.html
func (r bucketResource) put(a *action) interface{} {
	var created bool
	if r.bucket == nil {
		if !validBucketName(r.name) {
			fatalError(400, "InvalidBucketName", "The specified bucket is not valid")
		}
		if loc := locationConstraint(a); loc == "" {
			fatalError(400, "InvalidRequets", "The unspecified location constraint is incompatible for the region specific endpoint this request was sent to.")
		}
		// TODO validate acl
		r.bucket = &bucket{
			name: r.name,
			// TODO default acl
			objects: make(map[string]*object),
		}
		a.srv.buckets[r.name] = r.bucket
		created = true
	}
	if !created && a.srv.config.send409Conflict() {
		fatalError(409, "BucketAlreadyOwnedByYou", "Your previous request to create the named bucket succeeded and you already own it.")
	}
	r.bucket.acl = a.req.Header.Get("x-amz-acl")
	return nil
}

func (bucketResource) post(a *action) interface{} {
	fatalError(400, "Method", "bucket POST method not available")
	return nil
}

// validBucketName returns whether name is a valid bucket name.
// Here are the rules, from:
// http://docs.amazonwebservices.com/AmazonS3/2006-03-01/dev/BucketRestrictions.html
//
// Can contain lowercase letters, numbers, periods (.), underscores (_),
// and dashes (-). You can use uppercase letters for buckets only in the
// US Standard region.
//
// Must start with a number or letter
//
// Must be between 3 and 255 characters long
//
// There's one extra rule (Must not be formatted as an IP address (e.g., 192.168.5.4)
// but the real S3 server does not seem to check that rule, so we will not
// check it either.
//
func validBucketName(name string) bool {
	if len(name) < 3 || len(name) > 255 {
		return false
	}
	r := name[0]
	if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'z') {
		return false
	}
	for _, r := range name {
		switch {
		case r >= '0' && r <= '9':
		case r >= 'a' && r <= 'z':
		case r == '_' || r == '-':
		case r == '.':
		default:
			return false
		}
	}
	return true
}

var responseParams = map[string]bool{
	"content-type":        true,
	"content-language":    true,
	"expires":             true,
	"cache-control":       true,
	"content-disposition": true,
	"content-encoding":    true,
}

type objectResource struct {
	name    string
	version string
	bucket  *bucket // always non-nil.
	object  *object // may be nil.
}

// GET on an object gets the contents of the object.
// http://docs.amazonwebservices.com/AmazonS3/latest/API/RESTObjectGET.html
func (objr objectResource) get(a *action) interface{} {
	obj := objr.object
	if obj == nil {
		fatalError(404, "NoSuchKey", "The specified key does not exist.")
	}
	h := a.w.Header()
	// add metadata
	for name, d := range obj.meta {
		h[name] = d
	}
	// override header values in response to request parameters.
	for name, vals := range a.req.Form {
		if strings.HasPrefix(name, "response-") {
			name = name[len("response-"):]
			if !responseParams[name] {
				continue
			}
			h.Set(name, vals[0])
		}
	}
	if r := a.req.Header.Get("Range"); r != "" {
		fatalError(400, "NotImplemented", "range unimplemented")
	}
	// TODO Last-Modified-Since
	// TODO If-Modified-Since
	// TODO If-Unmodified-Since
	// TODO If-Match
	// TODO If-None-Match
	// TODO Connection: close ??
	// TODO x-amz-request-id
	h.Set("Content-Length", fmt.Sprint(len(obj.data)))
	h.Set("ETag", hex.EncodeToString(obj.checksum))
	h.Set("Last-Modified", obj.mtime.UTC().Format(http.TimeFormat))
	if a.req.Method == "HEAD" {
		return nil
	}
	// TODO avoid holding the lock when writing data.
	_, err := a.w.Write(obj.data)
	if err != nil {
		// we can't do much except just log the fact.
		log.Printf("error writing data: %v", err)
	}
	return nil
}

var metaHeaders = map[string]bool{
	"Content-MD5":         true,
	"x-amz-acl":           true,
	"Content-Type":        true,
	"Content-Encoding":    true,
	"Content-Disposition": true,
}

// PUT on an object creates the object.
func (objr objectResource) put(a *action) interface{} {
	// TODO Cache-Control header
	// TODO Expires header
	// TODO x-amz-server-side-encryption
	// TODO x-amz-storage-class

	// TODO is this correct, or should we erase all previous metadata?
	obj := objr.object
	if obj == nil {
		obj = &object{
			name: objr.name,
			meta: make(http.Header),
		}
	}

	var expectHash []byte
	if c := a.req.Header.Get("Content-MD5"); c != "" {
		var err error
		expectHash, err = base64.StdEncoding.DecodeString(c)
		if err != nil || len(expectHash) != md5.Size {
			fatalError(400, "InvalidDigest", "The Content-MD5 you specified was invalid")
		}
	}
	sum := md5.New()
	// TODO avoid holding lock while reading data.
	data, err := ioutil.ReadAll(io.TeeReader(a.req.Body, sum))
	if err != nil {
		fatalError(400, "TODO", "read error")
	}
	gotHash := sum.Sum(nil)
	if expectHash != nil && !bytes.Equal(gotHash, expectHash) {
		fatalError(400, "BadDigest", "The Content-MD5 you specified did not match what we received")
	}
	if a.req.ContentLength >= 0 && int64(len(data)) != a.req.ContentLength {
		fatalError(400, "IncompleteBody", "You did not provide the number of bytes specified by the Content-Length HTTP header")
	}

	// PUT request has been successful - save data and metadata
	for key, values := range a.req.Header {
		key = http.CanonicalHeaderKey(key)
		if metaHeaders[key] || strings.HasPrefix(key, "X-Amz-Meta-") {
			obj.meta[key] = values
		}
	}
	obj.data = data
	obj.checksum = gotHash
	obj.mtime = time.Now()
	objr.bucket.objects[objr.name] = obj
	return nil
}

func (objr objectResource) delete(a *action) interface{} {
	delete(objr.bucket.objects, objr.name)
	return nil
}

func (objr objectResource) post(a *action) interface{} {
	fatalError(400, "MethodNotAllowed", "The specified method is not allowed against this resource")
	return nil
}

type CreateBucketConfiguration struct {
	LocationConstraint string
}

// locationConstraint parses the <CreateBucketConfiguration /> request body (if present).
// If there is no body, an empty string will be returned.
func locationConstraint(a *action) string {
	var body bytes.Buffer
	if _, err := io.Copy(&body, a.req.Body); err != nil {
		fatalError(400, "InvalidRequest", err.Error())
	}
	if body.Len() == 0 {
		return ""
	}
	var loc CreateBucketConfiguration
	if err := xml.NewDecoder(&body).Decode(&loc); err != nil {
		fatalError(400, "InvalidRequest", err.Error())
	}
	return loc.LocationConstraint
}
