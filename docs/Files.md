# File Operations
<div>
Upload package files temporarily to aptly service. These files
could be added to local repositories using [local repositories API](/doc/api/repos).

All uploaded files are stored under `<rootDir>/upload` directory (see [configuration](/doc/configuration)).
This directory would be created automatically if it doesn't exist.

Uploaded files are grouped by directories to support concurrent uploads from multiple
package sources. Local repos add API can operate on directory (adding all files from directory) or
on individual package files. By default, all successfully added package files would be removed.

### List Directories

`GET /api/files`

List all directories.

Response: list of directory names.

Example:

    $ curl http://localhost:8080/api/files
    ["aptly-0.9"]

### Upload File(s)

`POST /api/files/:dir`

Parameter `:dir` is upload directory name. Directory would be created if it doesn't exist.

Any number of files can be uploaded in one call, aptly would preserve filenames. No check is performed
if existing uploaded would be overwritten.

Response: list of uploaded files as `:dir/:file`.

Example:

    $ curl -X POST -F file=@aptly_0.9~dev+217+ge5d646c_i386.deb http://localhost:8080/api/files/aptly-0.9
    ["aptly-0.9/aptly_0.9~dev+217+ge5d646c_i386.deb"]

### List Files in Directory

`GET /api/files/:dir`

Returns list of files in directory.

Response: list of filenames.

HTTP Errors:

 Code     | Description
----------|-------------------------
 404      | directory doesn't exist

Example:

    $ curl http://localhost:8080/api/files/aptly-0.9
    ["aptly_0.9~dev+217+ge5d646c_i386.deb"]


### Delete Directory

`DELETE /api/files/:dir`

Deletes all files in upload directory and directory itself.

Example:

    $ curl -X DELETE http://localhost:8080/api/files/aptly-0.9
    {}

### Delete File in Directory

`DELETE /api/files/:dir/:file`

Delete single file in directory.

Example:

    $ curl -X DELETE http://localhost:8080/api/files/aptly-0.9/aptly_0.9~dev+217+ge5d646c_i386.deb
    {}

</div>
