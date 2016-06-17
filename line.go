package main

//writeLine writes given string to given file with a newline appended at the end.
import (
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

//Line represents line contents and settings.
type Line struct {
	*Configuration
	Line    string   //String of the entire line
	Columns []string //Parsed columns
	ID      int      //uniqueobject ID.  Used for path calculation.
	LastID  int      //Remeber last object ID processed.  Prevents duplicates.
	Dir     string   //Directory of object file
	Path    string   //Full path of object calc. from objectID
	File    *os.File //File to write the line out to
}

//ProcessLine copies file to output.
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
	if l.LastID == l.ID {
		duplicates++
		log.Println("Skipping duplicate.", l.ID)
		writeLine(l.Line, outDups)
		return false
	}

	//Remember this line to compare with the next line to check for duplicates
	l.LastID = l.ID

	//Get full path for file in
	l.Path, err = l.GetInPath()
	if err != nil {
		return errorLine(&l.Line, err)
	}

	//Create new Line for line out.  Copy values from line in.
	lo := *l
	//What object out are we on?  Should be successful plus offset
	current := successful + l.OutFileRenameIntOffset
	lo.ID = current

	//init out file
	lo.OutLineFile()
	defer lo.File.Close()

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

	//Columns
	err = lo.GenLineFromColumns()
	if err != nil {
		return errorLine(&l.Line, err)
	}
	writeLine(lo.Line+lo.Delimiter+lo.Path, lo.File)

	return true
}

//GetInPath Get full path for file in
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
	subpath, err = l.GetPathFromID()
	if err != nil {
		return "", err
	}

	var fileName = strconv.Itoa(l.ID)

	//Full path for file in.
	fullPath = filepath.Join(parentPath, subpath, fileName) + inExtension
	return fullPath, nil
}

//GetOutPath Get full path for out file
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

	//Are we batching?  If so, add batch name to path.
	if l.OutAutoBatch == true {
		l.Dir = l.GetBatch()
	}

	//Create full path for out file
	//Create parent path first.
	if l.OutXtenderStructure == true {
		var subpath string
		subpath, err = l.GetPathFromID()
		if err != nil {
			return "", err
		}

		//Add the subpath to the current dir.
		l.Dir = filepath.Join(l.Dir, subpath)
	}

	//Add the parent Out directory.  This give the final directory.
	l.Dir = filepath.Join(l.OutDir, l.Dir)

	//Full path including file name.
	l.Path = filepath.Join(l.Dir, filename) + outExtension

	return l.Path, nil
}

//OutLineFile Create the "OutLines" file.
//Should call defer os.Close
func (l *Line) OutLineFile() {
	outFile := l.OutLinesName
	b := l.GetBatch()

	if b != "" {
		outFile = b + "_" + l.OutLinesName
	}
	outPath := filepath.Join(l.OutDir, outFile)
	var err error
	l.File, err = os.OpenFile(outPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		log.Fatal(err)
	}
}

//GetBatch Return the batch name, including incrementer
func (l *Line) GetBatch() string {
	//If batch is false, there is no batch.  Return blank string
	if l.OutAutoBatch == false {
		return ""
	}

	var batchCount int
	batchCount = l.ID / l.OutAutoBatchCount

	//Zero pad our batch
	pad := strconv.Itoa(l.OutAutoBatchZeroPad)
	bcn := fmt.Sprintf("%0"+pad+"d", batchCount)

	return l.OutAutoBatchName + bcn
}

//GenLineFromColumns Instead of copying the line from flat in, grab only some columns and write those to out.
func (l *Line) GenLineFromColumns() (err error) {

	var line string
	cols := strings.Split(l.OutLinesColomns, ",")
	for _, v := range cols {
		i, err := strconv.Atoi(v)
		if err != nil {
			return err
		}

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

//GetPathFromID Calculate the ApplicationXtender from a given object id, s
func (l *Line) GetPathFromID() (p string, e error) {
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
