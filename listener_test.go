package crawler_test

import (
	"github.com/agamble/btc-crawler"
	"github.com/btcsuite/btcd/wire"
	"os"
	"testing"
	"time"
)

var staticTestDir string = "test-snapshot"

// func newSingleNodeImage() {
// 	return crawler.NewImage(crawler.NewSeed())
// }

func NewStampedInv() *crawler.StampedInv {
	var msg *wire.MsgInv
	c := time.After(0 * time.Second)

	select {
	case <-c:
		msg = NewInvBlkMsg()
	case <-c:
		msg = NewInvTxMsg()
	}

	return &crawler.StampedInv{
		Timestamp: time.Now(),
		InvVects:  msg.InvList,
	}
}

func tempOutDirGenerator() func() string {
	now := time.Now()
	dataDirName := "snapshot-" + now.Format(time.Stamp)

	os.Mkdir(dataDirName, 0777)

	return func() string {
		return dataDirName
	}

}

func cleanupDirectory(dirName string) {
	os.RemoveAll(dirName)
}

func TestWriteInvMessage(t *testing.T) {
	node := crawler.NewSeed()
	node.ListenTxs = true
	node.ListenBlks = true

	tempDir := tempOutDirGenerator()
	defer cleanupDirectory(tempDir())

	invWriterC := make(chan *crawler.StampedInv, 1)
	go node.InvWriter(tempDir(), invWriterC)
	defer close(invWriterC)

	invWriterC <- NewStampedInv()

	time.Sleep(100 * time.Millisecond)
}

func TestReadInvMessage(t *testing.T) {
	decoder := crawler.NewDecoder(staticTestDir)
	decoder.Decode()
}
