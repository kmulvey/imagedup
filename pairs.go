package main

import "github.com/dgraph-io/badger/v3"

var empmtyByteSlice = make([]byte, 0)

func getPair(db *badger.DB, file1, file2 string) bool {
	var found bool
	var err = db.View(func(txn *badger.Txn) error {
		_, err := txn.Get([]byte(file1 + file2))
		if err == nil {
			found = true
		}
		return nil

	})
	handleErr("txn.get", err)
	return found
}

func setPair(db *badger.DB, file1, file2 string) {
	txn := db.NewTransaction(true)

	var err = txn.Set([]byte(file1+file2), empmtyByteSlice)
	handleErr("txn.set", err)
	err = txn.Set([]byte(file2+file1), empmtyByteSlice)
	handleErr("txn.set", err)

	err = txn.Commit()
	handleErr("txn commit", err)
}
