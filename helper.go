package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/jhoonb/archivex"
)

//copy copies file from in to out.
func copy(in, out string) (err error) {
	i, err := os.Open(in)
	if err != nil {
		failed++
		message := fmt.Sprint("File does not exist. ", in)
		fmt.Println(message)
		return errors.New(message)
	}
	defer i.Close()

	o, err := os.Create(out)
	if err != nil {
		log.Println("Cannot create file out.  Stopping execution", out)
		panic(err)
	}
	defer o.Close()

	_, err = io.Copy(o, i)
	if err != nil {
		log.Println("Copying failed.  Stopping execution", out)
		panic(err)
	} else {
		successful++
	}

	return nil
}

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
