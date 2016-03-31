package main

import (
	"os"
	"path/filepath"
	"strings"
	"fmt"
	"flag"
	"io/ioutil"
	"strconv"
	"time"
)

type EventInformation struct {
	STT int
	FileCreationTime time.Time
	EventTime		 time.Time
	CISCECreationTime time.Time
	Event 				string
}
var filelist []string = make([]string, 0)
var minEventToCreationDiff map[int]EventInformation = make(map[int]EventInformation)
func visitFile(path string, info os.FileInfo, err error) error {

	index := strings.LastIndex(path, ".")
	if (index > 0) {
		filelist = append(filelist, path)
	}
	return nil
}


func main() {
	flag.Parse()
	arguments := flag.Args()
	if (len(arguments) != 1) {
		fmt.Println("Please check argument length")
		return
	}
	path := arguments[0]
	filepath.Walk(path, visitFile)

	fmt.Printf("File paths parsed: %d\n", len(filelist))
	eventCounts := make(map[string]int)
	sttCounts := make(map[int]bool)
	for index := range filelist {
		eventInformation := readAndCreateEventInfo(filelist[index])
		eventInformation.isFileCreationSameAsCISCEEventCreation()
		eventInformation.isFileCreationSameAsEventTime("ICBK")
		eventCounts[eventInformation.Event] = eventCounts[eventInformation.Event]+1
		sttCounts[eventInformation.STT] = true

	}
	fmt.Println("STT;Filecreation/CISCE Creation date; Event date; DIFF;")
	for key, val := range minEventToCreationDiff {
		fmt.Printf("%d;%s;%s;%d;\n",key, dateAsString(val.FileCreationTime), dateAsString(val.EventTime), val.getMinDaysBetweenFileCreationAndEventTime())
	}

	fmt.Println("\n*** PRINT event occurences:")
	fmt.Println("Key;Count;\n")
	for key, val := range eventCounts {
		fmt.Printf("%s;%d;\n", key, val)
	}

	fmt.Printf("\n*** STT's analysed: %d\n", len(sttCounts))
}

const CESHP string = "CESHP___04"
const CEEVTSHP string = "CEEVTSHP04"
const CEHEADER string = "CEHEADER02"
func readAndCreateEventInfo(pathToCEfile string) EventInformation {

	bytes, err := ioutil.ReadFile(pathToCEfile)
	if (err != nil) {
		return EventInformation{STT : -1}
	}
	fileContent := string(bytes)
	lines := strings.Split(fileContent, "\n")
	stt := 0
	cisCECreationDateLine := ""
	shpEventLine := ""
	for index := range lines {
		line := lines[index]
		if strings.Index(line, CEHEADER) == 0 {
			cisCECreationDateLine = line
		} else if strings.Index(line, CESHP) == 0 {
				startOfSTT := len(CESHP)+1
				endOfSTT := strings.Index(line[startOfSTT:], "|")
				sttString := line[startOfSTT: startOfSTT+endOfSTT]
				stt, err = strconv.Atoi(sttString)
				if err != nil {
					fmt.Printf("Error parsing STT: %s", err.Error())
				}
		} else if strings.Index(line, CEEVTSHP) == 0 {
			shpEventLine = line
		}
	}
	return createEventInfo(pathToCEfile, stt, cisCECreationDateLine, shpEventLine)
}

const filePreFix string = "ce_event.cis."
func createEventInfo(pathToCEfile string, stt int, cisCECreationDateLine, line string) EventInformation {
	eventInfo := EventInformation{}
	eventInfo.FileCreationTime = parseFileCreationTimeFromFilePath(pathToCEfile)
	eventInfo.CISCECreationTime = parseCISCECreationDate(cisCECreationDateLine)
	eventInfo.Event = parseForValueAt(CEHEADER, line, 2)
	eventInfo.EventTime = createTimeObject(parseForValueAt(CEHEADER, line, 9))
	eventInfo.STT = stt
	return eventInfo

}

func parseFileCreationTimeFromFilePath(filePath string) time.Time{
	indexOfFileStart := strings.Index(filePath, filePreFix)
	filePath = filePath[indexOfFileStart + len(filePreFix):]
	indexOfNextPoint := strings.Index(filePath, ".")
	trimmedFileCreationTime := filePath[:indexOfNextPoint]
	return createTimeObject(trimmedFileCreationTime)
}


func parseCISCECreationDate(line string) time.Time {
	lineNoPreFix := line[len(CEHEADER)+1:]
	indexAfterPrjID := strings.Index(lineNoPreFix, "|")
	lineNoPrjID := lineNoPreFix[indexAfterPrjID+1:]
	indexAfterCreationDate := strings.Index(lineNoPrjID, "|")
	cisCECreationDate := lineNoPrjID[:indexAfterCreationDate]

	return createTimeObject(cisCECreationDate)
}

func parseForValueAt(preFix, line string, position int) string {
	line = line[len(preFix)+1:]

	index := 1
	for currPos := 1; currPos < position; currPos++ {
		index = strings.Index(line, "|")

		line = line[index+1:]
	}
	index = strings.Index(line, "|")
	//fmt.Printf("ParsedValue at pos(%d): %s\n", position, line[:index])
	return line[:index]
}


func createTimeObject(date string) time.Time {
	//fmt.Printf("date: %s len: %d \n", date, len(date))
	if (len(date) == 14) {
		date = date[:len(date)-6]
	}
	timeObj, err := time.Parse("20060102", date)
	if (err != nil) {
		return time.Time {}
	}
	return timeObj
}

func (this EventInformation) isFileCreationSameAsCISCEEventCreation() bool {
	fileCreationDays := getDaysAsInt(this.FileCreationTime)
	cisCeEventDays := getDaysAsInt(this.CISCECreationTime)
	if (fileCreationDays != cisCeEventDays) {
		fmt.Printf("FileVSCisCe: %d (f: %s / c: %s)\n", this.STT, dateAsString(this.FileCreationTime), dateAsString(this.CISCECreationTime))
		return false
	}
	return true
}

func (this EventInformation) isFileCreationSameAsEventTime(event string) {
	if strings.Compare(this.Event, event) != 0 {
		return
	}

	existingMin,bok := minEventToCreationDiff[this.STT]
	if !bok {
		minEventToCreationDiff[this.STT] = this
	} else if (existingMin.getMinDaysBetweenFileCreationAndEventTime() > this.getMinDaysBetweenFileCreationAndEventTime()) {
		minEventToCreationDiff[this.STT] = this
	}

}

func (this EventInformation) getMinDaysBetweenFileCreationAndEventTime() int {
	return int(this.FileCreationTime.Sub(this.EventTime).Hours()/24)
}

func getDaysAsInt(date time.Time) int {
	return (date.Year() -2014) * 365 + date.YearDay()
}

func dateAsString(date time.Time) string {
	return fmt.Sprintf("%d.%d.%d", date.Day(), date.Month(), date.Year())
}