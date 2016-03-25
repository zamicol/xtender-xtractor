package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var (
	lineCount  int = 0 //Increment for each line
	successful int = 0 //Sucessfully copied files
	failed     int = 0 //Failed to copy count.
	duplicates int = 0 //duplicates found in the sort file
	skipped    int = 0 //offset rows (future other skips)
	logFile    *os.File
	flatOut    *os.File //"index" file out.  Append existing rows plus new file path.
	errorOut   *os.File //Error lines
	configFile          = "config.json"

	last     string = "" //Remeber last object ID processed.  Prevents duplicates.
	lastPath        = "" //Path of the last object.
)

type Configuration struct {
	OutDir             string
	InDir              string
	FlatFileIn         string
	FlatFileOut        string
	FlatFileErrorLines string
	Log                string
	InFileExt          string
	OutFileExt         string
	OutFileNameInt     bool
	DirDepth           int
	FolderSize         int
	Delimiter          string
	RowOffset          int
	ColObjectID        int
	ColFileName        int
	ColFileExt         int
	ColPath            int
}

func main() {
	//Open config
	file, _ := os.Open(configFile)
	decoder := json.NewDecoder(file)
	c := Configuration{}
	err := decoder.Decode(&c)
	if err != nil {
		panic("Unable to process config file. " + configFile + ". This probably means that the file isn't valid json.")
	}
	//logging
	initLog(&c)
	defer stopLog()

	//Create out dir if not exist, only one deep
	_, e := os.Stat(c.OutDir)
	if os.IsNotExist(e) {
		fmt.Println("Out directory does not exist.", c.OutDir)
		e := os.Mkdir(c.OutDir, 0777)
		if e == nil {
			fmt.Println("Created output directory: ", c.OutDir)
		} else {
			fmt.Println("Unable to create output directory: ", c.OutDir)
		}
	} else {
		fmt.Println("Out directory exists.", c.OutDir)
	}

	flatOut, err = os.Create(c.FlatFileOut)
	if err != nil {
		log.Fatal(err)
	}

	errorOut, err = os.Create(c.FlatFileErrorLines)
	if err != nil {
		log.Fatal(err)
	}
	defer flatOut.Close()

	//Process file
	readFile(c.FlatFileIn, &c)
}

func readFile(flat string, c *Configuration) {
	file, err := os.Open(flat)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for i := c.RowOffset; i > 0; i-- {
		scanner.Scan()
		writeLine(scanner.Text(), flatOut)
		lineCount++
		skipped++
	}

	for scanner.Scan() {
		var l = scanner.Text()
		lineCount++
		processLine(&l, c)
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}

func processLine(line *string, c *Configuration) {

	columns := strings.Split(*line, c.Delimiter)

	d := "|"

	//Check to see if object was already processed.
	if last == columns[c.ColObjectID] {
		log.Println("Skipping duplicate.", columns[c.ColObjectID])
		writeLine(*line+d+lastPath, flatOut)
		duplicates++
		return
	} else {
		last = columns[c.ColObjectID]
	}

	//Get parent path for in file.
	var extension string
	if c.InFileExt == "" {
		extension = columns[c.ColFileExt]
	} else {
		extension = c.InFileExt
	}

	//Full file path in
	var subpath string
	subpath, err := getPathFromId(columns[c.ColObjectID], c)

	//Use a static path OR path from flat file dump row
	var inPath string
	if c.InDir != "" {
		inPath = c.InDir
	} else {
		inPath = columns[c.ColFileName]
	}

	fullpath := filepath.Join(inPath, subpath, columns[c.ColObjectID]) + extension
	if err != nil {
		log.Println("Unable to process line: ", lineCount, err.Error())
	}

	//full file path out.
	var filename string
	if c.OutFileNameInt == true {
		filename = strconv.Itoa(successful)
	} else {
		filename = columns[c.ColFileName]
	}

	out := filepath.Join(c.OutDir, filename) + c.OutFileExt
	lastPath = out

	fmt.Println(fullpath, out, subpath, columns)

	//Copy file
	err = copy(fullpath, out)
	if err == nil {
		writeLine(*line+d+out, flatOut)
	} else {
		writeLine(*line, errorOut)
	}
}

func writeLine(s string, f *os.File) {
	f.WriteString(s + "\n")
}

func copy(in, out string) (e error) {
	i, err := os.Open(in)
	if err != nil {
		message := fmt.Sprint("File does not exist.", in)
		log.Println(message)
		fmt.Println(message)
		failed++
		return errors.New("File not found")
	}

	o, err := os.Create(out)
	if err != nil {
		log.Println("Cannot create file out.  Stopping program", out)
		panic(err)
	}
	defer i.Close()
	defer o.Close()

	w, err := io.Copy(o, i)
	if err != nil {
		log.Println("Copying failed.  Stopping program", out)
		panic(err)
	} else {
		successful++
	}

	fmt.Println(w)
	return nil
}

func getPathFromId(s string, c *Configuration) (p string, e error) {
	id, e := strconv.Atoi(s)

	for i := c.DirDepth; i > 0; i-- {
		powered := math.Pow(float64(c.FolderSize), float64(i))
		subpath := int(math.Mod((float64(id) / powered), float64(c.FolderSize)))
		p = filepath.Join(p, strconv.Itoa(subpath))
		//fmt.Println("Path:", p, "Id value:", id, "Subpath:", subpath, "I:", i, "Powered: ", powered)
	}
	return p, e
}

//Init log
func initLog(c *Configuration) {
	f, err := os.OpenFile(c.Log, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		panic("error opening file log file.")
	} else {
		fmt.Println("Created and opened log:", c.Log)
	}

	log.SetOutput(f)
	logFile = f
	log.Println("Started process.")
	m := fmt.Sprintf("Configuration: %+v\n", *c)
	fmt.Println(m)
	log.Println(m)

}

func stopLog() {
	exitMessage := fmt.Sprint("Process stopped. \nLines processed: ", lineCount,
		"\nSkipped rows:", skipped,
		"\nSucessfully copied: ", successful,
		"\nDuplicates skipped: ", duplicates,
		"\nFailed:", failed)
	log.Println(exitMessage)
	fmt.Println(exitMessage)
	defer logFile.Close()
}
