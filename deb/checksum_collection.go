package deb

import (
	"bytes"

	"github.com/smira/aptly/aptly"
	"github.com/smira/aptly/database"
	"github.com/smira/aptly/utils"
	"github.com/ugorji/go/codec"
)

// ChecksumCollection does management of ChecksumInfo in DB
type ChecksumCollection struct {
	db          database.Storage
	codecHandle *codec.MsgpackHandle
}

// NewChecksumCollection creates new ChecksumCollection and binds it to database
func NewChecksumCollection(db database.Storage) *ChecksumCollection {
	return &ChecksumCollection{
		db:          db,
		codecHandle: &codec.MsgpackHandle{},
	}
}

func (collection *ChecksumCollection) dbKey(path string) []byte {
	return []byte("C" + path)
}

// Get finds checksums in DB by path
func (collection *ChecksumCollection) Get(path string) (*utils.ChecksumInfo, error) {
	encoded, err := collection.db.Get(collection.dbKey(path))
	if err != nil {
		if err == database.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}

	c := &utils.ChecksumInfo{}

	decoder := codec.NewDecoderBytes(encoded, collection.codecHandle)
	err = decoder.Decode(c)
	if err != nil {
		return nil, err
	}

	return c, nil
}

// Update adds or updates information about checksum in DB
func (collection *ChecksumCollection) Update(path string, c *utils.ChecksumInfo) error {
	var encodeBuffer bytes.Buffer

	encoder := codec.NewEncoder(&encodeBuffer, collection.codecHandle)
	err := encoder.Encode(c)
	if err != nil {
		return err
	}

	return collection.db.Put(collection.dbKey(path), encodeBuffer.Bytes())
}

// Check interface
var (
	_ aptly.ChecksumStorage = &ChecksumCollection{}
)
