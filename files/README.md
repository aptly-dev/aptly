## Downloaded packages

### SHA-256

For each uploaded _Debian_ package a [SHA-256](https://en.wikipedia.org/wiki/SHA-2) checksum is computed.
This checksum is used to create a file tree where each package will reside, with following hierarchy.

#### Directory and sub-directories structure

  - 1st and 2nd characters of SHA-256 checksum as sub-directory of **rootDir**/pool directory.
  - 3rd and 4th characters of SHA-256 checksum as sub-directory of the former

ex:

 sha256sum **476e**0cdac6bc757dd2b78bacc1325323b09c45ecb41d4562deec2a1c7c148405 my-package_1.2.3_all.deb

```bash
${rootDir}/pool # rootDir defined in aptly.conf
└── 47
    └── 6e
```

#### Filename

The following items are concatenated to form the filename under which package is stored.
  - 5th to the 31st characters of SHA-256 checksum
  - "\_" (undescore)
  - filename of uploaded _Debian_ as defined in [Debian package file names](https://www.debian.org/doc/manuals/debian-reference/ch02.en.html#_debian_package_file_names)

ex:

 sha256sum 476e**0cdac6bc757dd2b78bacc13253**23b09c45ecb41d4562deec2a1c7c148405 **my-package_1.2.3_all.deb**

```
 0cdac6bc757dd2b78bacc13253_my-package_1.2.3_all.deb
```

### MD5

For each uploaded _Debian_ package a [MD5](https://en.wikipedia.org/wiki/MD5) checksum is computed.
This checksum is used to create a file tree where each package will reside, with following hierarchy

**Note:** [MD5](https://en.wikipedia.org/wiki/MD5) is only legacy layout. Its support is limited to
 'read' files from the pool, it never puts files this way for new package files.

#### Directory and sub-directories structure

  - 1st and 2nd characters of MD5 checksum as sub-directory name of **rootDir**/pool directory
  - 3rd and 4th characters of MD5 chacksum as sub-directory name of the former

ex:

 md5sum **feea**3c0c3e823615bf2d417b052a96b4 my-package_1.2.3_all.deb


```bash
${rootDir}/pool # rootDir defined in aptly.conf
└── fe
    └── ea
```

#### Filename

Uploaded _Debian_ is stored as-is and not renamed.

### Example

```bash
${rootDir}/pool # rootDir defined in aptly.conf
├── 00
│   ├── 25
│   │   └── yet_another_package-0.6.0_all.deb
│   ├── 60
│   ├── 97
│   │   └── 80ced73165f92fea490f2561a7c4_my-package_0.0.1_all.deb
│   ├── 6e 
│   │   └── 0cdac6bc757dd2b78bacc13253_my-package_1.2.3_all.deb # sha256sum 476e0cdac6bc757dd2b78bacc1325323b09c45ecb41d4562deec2a1c7c148405
│   └── db
│       └── yet_another_package-0.5.8_all.deb # md5sum 00db7ada61aa28a6931267f1714cbb15
...
├── 2a                                                                                                                
│   ├── 10                                                                                                            
│   │   └── yet_another_package-0.5.9_all.deb
│   ├── 64
│   │   └── 80ced73165f92fea490f2561a7c4_my-other-package_2.3.2_amd64.deb
│   ├── 4c                                                                                                            
│   ├── 5c                                                                                                            
│   │   └── yet_another_package-0.6.1_all.deb
│   ├── 77                                                                                                            
│   ├── b5                                                                                                            
│   │   └── 4b2eb349236cf5c4af7eca68a43b_my-package_0.2.0_amd64.deb
...
└── ff
    ├── 4c                                                                                                            
    ├── 5a                                                                                                            
    │   └── 8868dd8661bbe25c51bdd9b2d25c_my-package_0.2.0_amd64.deb                                          
    └── dc
```
