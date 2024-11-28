# Upload Package Files
<div>

In order to add debian package files to a local repository, files are first uploaded to a temporary directory.
Then the directory (or a specific file within) is added to a repository. After adding to a repositorty, the directory resp. files are removed bt default.

All uploaded files are stored under `<rootDir>/upload/<tempdir>` directory.

For concurrent uploads from CI/CD pipelines, make sure the tempdir is unique.


</div>
