package deb

import (
	"fmt"
	"sort"
	"testing"
)

func BenchmarkReflistSimpleMerge(b *testing.B) {
	const count = 4096

	l := NewPackageRefList()
	r := NewPackageRefList()

	for i := 0; i < count; i++ {
		if i%2 == 0 {
			l.Refs = append(l.Refs, []byte(fmt.Sprintf("Pamd64 pkg%d %d", i, i)))
		} else {
			r.Refs = append(r.Refs, []byte(fmt.Sprintf("Pamd64 pkg%d %d", i, i)))
		}
	}

	sort.Sort(l)
	sort.Sort(r)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.Merge(r, false, true)
	}
}

func BenchmarkReflistDecode(b *testing.B) {
	const count = 4096

	r := NewPackageRefList()
	for i := 0; i < count; i++ {
		r.Refs = append(r.Refs, []byte(fmt.Sprintf("Pamd64 pkg%d %d", i, i)))
	}

	sort.Sort(r)
	data := r.Encode()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		(&PackageRefList{}).Decode(data)
	}
}
