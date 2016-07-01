package main

//initLog Opens or creates log file, set log output.
import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

//InitLog starts log.
func InitLog(c *Configuration) {
	ol := filepath.Join(c.OutDir, c.OutLog)
	f, err := os.OpenFile(ol, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		panic("error opening file log file.")
	} else {
		fmt.Println("Created and opened log:", c.OutLog)
	}

	log.SetOutput(f)
	logFile = f
	log.Println("Started process.")
	m := fmt.Sprintf("Configuration: %+v\n", *c)
	log.Println(m)
}

//stopLog closses the log file and prints the final exit message.
func stopLog() {
	exitMessage := fmt.Sprint(
		"Process stopped. ",
		"\nLines processed: ", lineCount,
		"\nSkipped rows:", skipped,
		"\nSucessfully copied: ", successful,
		"\nDuplicates skipped: ", duplicates,
		"\nFailed:", failed)
	log.Println(exitMessage)
	fmt.Println(exitMessage)
	defer logFile.Close()
}
