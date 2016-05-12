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

	skipped    int      = 0             //offset rows (future other skips)
	logFile    *os.File                 //Log file
	outFlat    *os.File                 //"index" file out.  Append existing rows plus new file path.
	outDups    *os.File                 //Copy over any lines that are skipped due to being duplicated object id's
	errorLines *os.File                 //Error lines
	configFile          = "config.json" //Configuration file.

	last int = 0 //Remeber last object ID processed.  Prevents duplicates.
)

//Configuration
//See README for description of each variable.
type Configuration struct {
	//In
	InFlatFile string
	InDir      string
	InFileExt  string

	//Out
	OutFlatFile         string
	OutDir              string
	OutErrorLines       string
	OutDuplicateLines   string
	OutFileExt          string
	OutFileRenameInt    bool
<<<<<<< HEAD
	OutColomns          string
=======
	OutColomns          []interface{}
>>>>>>> d86a57f2a0b02613154740cd042227fa299eb17f
	OutCountOffset      int
	OutXtenderStructure bool
	//AutoBatch
	OutAutoBatch      bool
	OutAutoBatchCount int
	OutAutoBatchName  string

	//Global
	Log             string
	RowOffset       int //Process ignorded rows.  Usefull for headers.  Will be copied to output.
	DirDepth        int
	FolderSize      int
	Delimiter       string //Applies to both in and out.
	ComputeChecksum bool

	//Columns
	ColObjectID   int
	ColFileName   int
	ColFileExtIn  int
	ColFileExtOut int
}

type Line struct {
	*Configuration
	Line    string   //String of the line
	Columns []string //Parsed columns
	ID      int      //uniqueobject ID.  Used for path calculation.
	Dir     string   //Directory of file
	Path    string   //Full path including file
}

//main opens and parses the config, Starts logging, and then call setup
func main() {
	//Load Config
	c := Config()

	//Logging
	initLog(c)
	defer stopLog()

	//Process file
	setup(c)
}

func Config() *Configuration {
	//Open config
	file, _ := os.Open(configFile)
	defer file.Close()
	//Config is json.
	decoder := json.NewDecoder(file)
	c := new(Configuration)
	err := decoder.Decode(c)
	if err != nil {
		panic("Unable to process config file. " + configFile +
			". This probably means that the file isn't valid json." + err.Error())
	}
	return c
}

//Setup
//Creates output files and directories and calls processIndex
func setup(c *Configuration) {
	var err error

	//Create out dir if not exist, only one deep
	Mkdir(c.OutDir)

	outFlat, err = os.OpenFile(c.OutFlatFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		log.Fatal(err)
	}
	defer outFlat.Close()

	outDups, err = os.OpenFile(c.OutDuplicateLines, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		log.Fatal(err)
	}
	defer outDups.Close()

	errorLines, err = os.OpenFile(c.OutErrorLines, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		log.Fatal(err)
	}
	defer errorLines.Close()

	//Process file
	processIn(c.InFlatFile, c)
}

//processIndex processes flat file dump file line by line.
func processIn(flat string, c *Configuration) {
	file, err := os.Open(flat)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	//Special case for RowOffset rows.
	for i := c.RowOffset; i > 0; i-- {
		//Get the nex line using the Scan() method
		scanner.Scan()

		//We will copy the RowOffset lines to the out file.
		writeLine(scanner.Text(), outFlat)
		lineCount++
		skipped++
	}

	//For all rows we want to process
	for scanner.Scan() {
		//Get the line to be processed.
		var l = scanner.Text()
		//Increment the counter and process the line
		lineCount++

		//Construct our line type
		//Get the columns in the line
		col := strings.Split(l, c.Delimiter)
		//New line object.
		line := &Line{
			Configuration: c,
			Line:          l,
			Columns:       col,
		}

		line.ProcessLine()
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}

//processLine copies file to output.
//Returns false in the event of error
func (l *Line) ProcessLine() (b bool) {
	var err error

	//Get object ID from column
	l.ID, err = strconv.Atoi(l.Columns[l.ColObjectID])
	if err != nil {
		return errorLine(&l.Line, err)
	}

	//Check to see if object was already processed.
	//We do this by comparing objectID, which represents a unique object
	if last == l.ID {
		duplicates++
		log.Println("Skipping duplicate.", l.ID)
		writeLine(l.Line, outDups)
		return false
	}

	//Remember this line to compare with the next line to check for duplicates
	last = l.ID

	//Get full path for file in
	l.Path, err = l.GetInPath()
	if err != nil {
		return errorLine(&l.Line, err)
	}

	//Create new Line for line out.  Copy values from line in.
	lo := *l
	//What object out are we on?  Should be successful plus offset
	current := successful + l.OutCountOffset
	lo.ID = current

	//Get full path for file out.
	lo.Path, err = lo.GetOutPath()
	if err != nil {
		//Write line in to error line.
		return errorLine(&l.Line, err)
	}

	//fmt.Println("inFullPath:", l.Path, "outFullPath:", lo.Path)
	//Create path for out.
	MkdirAll(lo.Dir)

	//Copy file
	err = copy(l.Path, lo.Path)
	//Write line to out file if successful
	if err != nil {
		return errorLine(&l.Line, err)
	}

	//TODO
	//Columns
	err = lo.GenLineFromColumns()
	fmt.Println("CurrentLine:", lo.Line)
	if err != nil {
		fmt.Println("There shouldn't be an error here.", err)
		return errorLine(&l.Line, err)
	}

	writeLine(lo.Line+lo.Delimiter+lo.Path, outFlat)

	return true
}

func (l *Line) GetInPath() (fullPath string, err error) {
	//Get extension for in file.
	var inExtension string
	if l.InFileExt == "" {
		inExtension = l.Columns[l.ColFileExtIn]
	} else {
		inExtension = l.InFileExt
	}

	//Use a static path OR path from flat file dump row
	var parentPath string
	if l.InDir != "" {
		parentPath = l.InDir
	} else {
		parentPath = l.Columns[l.ColFileName]
	}

	var subpath string
	subpath, err = l.GetPathFromId()
	if err != nil {
		return "", err
	}

	var fileName = strconv.Itoa(l.ID)

	//Full path for file in.
	fullPath = filepath.Join(parentPath, subpath, fileName) + inExtension
	return fullPath, nil
}

func (l *Line) GetOutPath() (out string, err error) {
	//full file path out.
	var filename string

	if l.OutFileRenameInt == true {
		//Use ID as file name
		filename = strconv.Itoa(l.ID)
	} else {
		//Get the file name from the in file
		filename = l.Columns[l.ColFileName]
	}

	//Extension for out file.
	//Use static extension.  If blank, assume this value is provided in row's column
	outExtension := l.OutFileExt
	if outExtension == "" {
		outExtension = l.Columns[l.ColFileExtOut]
	}

	//Are we batching?  If so, add batch to file name.
	if l.OutAutoBatch == true {

		//Calculate which batch.
		var batchCount int
		batchCount = l.ID / l.OutAutoBatchCount
		bcn := strconv.Itoa(batchCount)
		l.Dir = l.OutAutoBatchName + bcn
	}

	//Create full path for out file
	//Create parent path first.
	if l.OutXtenderStructure == true {
		var subpath string
		subpath, err = l.GetPathFromId()
		if err != nil {
			return "", err
		}

		//l.Dir can be populated with batch folder
		l.Dir = filepath.Join(l.OutDir, l.Dir, subpath)
	} else {
		l.Dir = l.OutDir
	}

	//Parent path plus file
	l.Path = filepath.Join(l.Dir, filename) + outExtension

	return l.Path, nil
}

func (l *Line) GenLineFromColumns() (err error) {

	var line string

	cols := strings.Split(l.OutColomns, ",")
	fmt.Println("cols:", cols)
	for _, v := range cols {
		i, _ := strconv.Atoi(v)

		//Fencepost
		if line == "" {
			line += l.Columns[i]
		} else {
			line += l.Delimiter + l.Columns[i]
		}
	}

	l.Line = line

	return nil
}

//Calculate the ApplicationXtender from a given object id, s
func (l *Line) GetPathFromId() (p string, e error) {
	//For each step of the path, we will calculate that portion of the path
	for i := l.DirDepth; i > 0; i-- {
		//Get maximum of how many objects there could be at this level.
		powered := math.Pow(float64(l.FolderSize), float64(i))
		//divide ID by max objects, and then Mod the id by how many objects per folder.
		subpath := int(math.Mod((float64(l.ID) / powered), float64(l.FolderSize)))
		//Add this portion of the path to the directory string.  Iterate next level until done.
		p = filepath.Join(p, strconv.Itoa(subpath))
		//fmt.Println("Path:", p, "Id value:", id, "Subpath:", subpath, "I:", i, "Powered: ", powered)
	}
	return p, e
}

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

//initLog Opens or creates log file, set log output.
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
	log.Println(m)
}

func Mkdir(dir string) {
	//Create out dir if not exist, only one deep
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		fmt.Println("Out directory does not exist.", dir)
		err := os.Mkdir(dir, 0777)
		if err == nil {
			fmt.Println("Created output directory: ", dir)
		} else {
			message := fmt.Sprint("Unable to create output directory.  Stopping execution ", dir, err)
			errors.New(message)
			panic(err)
		}
	} else {
		fmt.Println("Out directory exists.", dir)
	}
}

func MkdirAll(dir string) {
	//Create out dir if not exist, only one deep
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		err := os.MkdirAll(dir, 0777)
		if err != nil {
			message := fmt.Sprint("Unable to create output directory.  Stopping execution ", dir, err)
			errors.New(message)
			panic(err)
		}
	}
}

//stopLog closses the log file and prints the final exit message.
func stopLog() {
	exitMessage := fmt.Sprint(
		"Process stopped. \nLines processed: ", lineCount,
		"\nSkipped rows:", skipped,
		"\nSucessfully copied: ", successful,
		"\nDuplicates skipped: ", duplicates,
		"\nFailed:", failed)
	log.Println(exitMessage)
	fmt.Println(exitMessage)
	defer logFile.Close()
}

//writeLine writes given string to given file with a newline appended at the end.
func writeLine(s string, f *os.File) {
	f.WriteString(s + "\n")
}

//errorLine
//helper function for when encountering error lines.  Logs the line number with error and and writes the line to the error line file.
func errorLine(line *string, err error) (b bool) {
	writeLine(*line, errorLines)
	log.Println("Line:", lineCount, err)
	return false
}
