// Package files handles operation on filesystem for both public pool and published files
package files

// Repository directory structure:
// <root>
// \- pool
//    \- ab
//       \- ae
//          \- package.deb
// \- public
//    \- dists
//       \- squeeze
//          \- Release
//          \- main
//             \- binary-i386
//                \- Packages.bz2
//                   references packages from pool
//    \- pool
//       contains symlinks to main pool
