package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/ulikunitz/xz"
	bolt "go.etcd.io/bbolt"
)

var BackupInterval = time.Hour * 24

type BackupSaver interface {
	/*
		Interface that recieves backups of the database
		to save in a variety of ways
	*/
	Save(io.Reader)
	/*
		The struct is responsible for determing the name
		of the backup (such as the date).

		The body will be compressed using the Lempel–Ziv–
		Markov chain algorithm (.xz), this should be
		reflected in the filename that is saved.
	*/
}

type FileSaver struct {
	Prefix string
}

func (f FileSaver) Save(in io.Reader) {
	filename := fmt.Sprintf("%s%s.xz", f.Prefix, time.Now().Format(time.RFC3339))
	file, err := os.Create(filename)
	if err != nil {
		fmt.Printf("Error during creating backup file: %s\n", err.Error())
		return
	}
	defer file.Close()
	io.Copy(file, in)
}

var fileSaver FileSaver

func runBackup() {
	/*
		Write database to xz reader
		Give this to each BackupSaver
	*/
	fileSaverBuffer := new(bytes.Buffer)
	compressedOut := io.MultiWriter(fileSaverBuffer)

	xzWriter, err := xz.NewWriter(compressedOut)
	if err != nil {
		fmt.Printf("Error while opening xzWriter: %s\n", err.Error())
		return
	}

	if err := db.View(func(tx *bolt.Tx) error {
		_, err := tx.WriteTo(xzWriter)
		return err
	}); err != nil {
		fmt.Printf("Error while reading database for backup: %s\n", err.Error())
		return
	}

	if err := xzWriter.Close(); err != nil {
		fmt.Printf("Error while closing xzWriter: %s\n", err.Error())
		return
	}

	if Configuration.Backup.File.Enabled {
		fileSaver.Save(fileSaverBuffer)
	}

}

/*
This makes sure that the above objects all
satisfy the BackupSaver interface.
*/
var _ = []BackupSaver{
	FileSaver{},
}
