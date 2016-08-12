package lib

import (
	"fmt"
	"log"
	"os"

	"github.com/jhoonb/archivex"
)

//Mkdir Create out dir if not exist, only one deep
func Mkdir(dir string) {
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		fmt.Println("Out directory does not exist.", dir)
		err := os.Mkdir(dir, 0777)
		if err == nil {
			fmt.Println("Created output directory: ", dir)
		} else {
			message := fmt.Sprint("Unable to create output directory.  Stopping execution ", dir, err)
			panic(message)
		}
	} else {
		fmt.Println("Out directory exists.", dir)
	}
}

//MkdirAll Create out dir if not exist recursivly
func MkdirAll(dir string) {
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		err := os.MkdirAll(dir, 0777)
		if err != nil {
			message := fmt.Sprint("Unable to create output directory.  Stopping execution ", dir, err)
			panic(message)
		}
	}
}

//zip zips directory
func zip(dir string, deleteSource bool) {
	message := fmt.Sprint("Zipping source directory:", dir)
	log.Println(message)
	fmt.Println(message)
	zip := new(archivex.ZipFile)
	zip.Create(dir + ".zip")
	zip.AddAll(dir, true)
	zip.Close()
	//Log is now not available since out is zipped.
	fmt.Println("Zipped source directory:", dir+".zip")
	//Delete sources
	if deleteSource {
		err := os.RemoveAll(dir)
		if err != nil {
			fmt.Println("Unable to remove directory.", err.Error())
		}
		fmt.Println("Removed zip source directory:", dir)
	}
}
