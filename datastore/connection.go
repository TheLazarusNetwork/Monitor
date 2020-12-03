package datastore

import (
	"context"
	"io/ioutil"
	"os"
	"time"

	"github.com/TheLazarusNetwork/Monitor/utility"

	"github.com/textileio/go-threads/common"
	"github.com/textileio/go-threads/core/thread"
	"github.com/textileio/go-threads/db"
	"github.com/textileio/go-threads/util"
)

// CreateDB for creating a Thread DB
func CreateDB() (*db.DB, func()) {
	dir, err := ioutil.TempDir("./", "logs")
	utility.CheckError(err)
	n, err := common.DefaultNetwork(dir, common.WithNetDebug(true), common.WithNetHostAddr(util.FreeLocalAddr()))
	utility.CheckError(err)
	id := thread.NewIDV1(thread.Raw, 32)
	d, err := db.NewDB(context.Background(), n, id)
	utility.CheckError(err)
	return d, func() {
		time.Sleep(time.Second) // Give threads a chance to finish work
		if err := n.Close(); err != nil {
			panic(err)
		}
		_ = os.RemoveAll(dir)
	}
}
