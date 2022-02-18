package main

import (
	"errors"
	"strconv"
	"strings"

	"github.com/dgraph-io/badger/v3"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

func cacheCheckpoint(checkpoints chan pair) {
	var dbLogger = logrus.New()
	dbLogger.SetLevel(log.WarnLevel)
	var db, err = badger.Open(badger.DefaultOptions("checkpoints").WithLogger(dbLogger))
	handleErr("badger open", err)

	txn := db.NewTransaction(true) // Read-write txn
	var i int
	for cp := range checkpoints {
		i++
		err = txn.Set([]byte("checkpoint"), []byte(strconv.Itoa(cp.I)+" "+strconv.Itoa(cp.J)))
		handleErr("txn.set", err)

		if i%50 == 0 {
			err = txn.Commit()
			handleErr("txn commit", err)
			txn = db.NewTransaction(true)
		}
	}
	err = txn.Commit()
	handleErr("txn commit", err)

	err = db.Close()
	handleErr("db close", err)
}

func getCheckpoints() (int, int) {
	var dbLogger = logrus.New()
	dbLogger.SetLevel(log.WarnLevel)
	var db, err = badger.Open(badger.DefaultOptions("checkpoints").WithLogger(dbLogger))
	handleErr("badger open", err)

	var valBytes []byte
	err = db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("checkpoint"))
		if errors.Is(err, badger.ErrKeyNotFound) {
			valBytes = []byte("0 0")
			return nil
		}
		handleErr("tnx get", err)

		valBytes, err = item.ValueCopy(valBytes)
		handleErr("Value copy", err)
		return nil
	})
	handleErr("db view", err)

	var valSlice = strings.Split(string(valBytes), " ")
	startI, err := strconv.Atoi(valSlice[0])
	handleErr("atoi: "+valSlice[0], err)
	startJ, err := strconv.Atoi(valSlice[1])
	handleErr("atoi: "+valSlice[1], err)
	err = db.Close()
	handleErr("db close", err)

	return startI, startJ
}
