package state

import (
	"errors"

	"github.com/dgraph-io/badger/v4"
)

var ErrNotImplemented = errors.New("badger store: function not implemented yet")

type BadgerStore struct {
	db *badger.DB
}

func NewBadgerStore(path string) (*BadgerStore, error) {
	opts := badger.DefaultOptions(path).WithLogger(nil)
	db, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}
	return &BadgerStore{db: db}, nil
}

func (s *BadgerStore) Get(key []byte) ([]byte, error) {
	return nil, ErrNotImplemented
}

func (s *BadgerStore) Set(key []byte, value []byte) error {
	return ErrNotImplemented
}

func (s *BadgerStore) Delete(key []byte) error {
	return ErrNotImplemented
}

func (s *BadgerStore) Close() error {
	return s.db.Close()
}
