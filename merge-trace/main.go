package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/golang/glog"
)

var defaultValue = flag.String("default", "0", "Default value to assign before the first merge point from the second file")

func main() {
	flag.Parse()

	if len(flag.Args()) != 3 {
		fmt.Printf("USAGE: merger <file to merge into> <file to merge> <output file>\n")
		return
	}

	first := openCSVRead(flag.Arg(0))
	second := openCSVRead(flag.Arg(1))
	output := openCSVWrite(flag.Arg(2))

	// Merge the first lines, usually contain the titles.
	titles, err := first.Read()
	if err != nil {
		glog.Fatal(err)
	}
	secondTitles, err := second.Read()
	if err != nil {
		glog.Fatal(err)
	}

	// The first column of the second file is the timestamp.
	err = output.Write(append(titles, secondTitles[1:]...))
	if err != nil {
		glog.Fatal(err)
	}

	// Grab first merge point.
	oldValue := *defaultValue
	mergeTime, mergeValue, err := getLine(second)
	if err != nil {
		glog.Fatal(err)
	}

	// Merge second file into first file.
	for {
		// Read line from the first file.
		values, err := first.Read()
		if err != nil {
			if err == io.EOF {
				break
			}

			glog.Errorf("Failed to parse line from first file: %v", err)
			continue
		}
		curTime, err := parseTime(values[0])
		if err != nil {
			glog.Errorf("Failed to parse time of line %v: %v", values, err)
			continue
		}

		// Use the old value until we reach the new merge time.
		// Zero merge time means no more content in the file.
		if !mergeTime.IsZero() && !curTime.Before(mergeTime) {
			oldValue = mergeValue
			mergeTime, mergeValue, err = getLine(second)
			if err != nil {
				if err == io.EOF {
					mergeTime = zero
				} else {
					glog.Errorf("Failed to read line from second file: %v", err)
				}
			}
		}

		// Append the second file's content into the first.
		err = output.Write(append(values, oldValue))
		if err != nil {
			glog.Errorf("Failed to write output to file: %v", err)
		}
		output.Flush()
	}
}

var zero time.Time

// Get a line from the second file.
func getLine(r *csv.Reader) (time.Time, string, error) {
	record, err := r.Read()
	if err != nil {
		return zero, "", err
	}

	if len(record) != 2 {
		return zero, "", fmt.Errorf("record had unexpected amount of fields: %v", record)
	}
	unixTime, err := parseTime(record[0])
	if err != nil {
		return zero, "", err
	}

	return unixTime, record[1], nil

}

// Parse time from a string with a UNIX time.
func parseTime(timeStr string) (time.Time, error) {
	unixTime, err := strconv.ParseInt(timeStr, 10, 64)
	if err != nil {
		return zero, fmt.Errorf("failed to parse UNIX timestamp from %q: %v", timeStr, err)
	}

	return time.Unix(unixTime, 0), nil
}

// Open a CSV file for R/W and create if it doesn't exist.
func openCSVWrite(filename string) *csv.Writer {
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		glog.Fatalf("Failed to open %q: %v", filename, err)
	}
	return csv.NewWriter(file)
}

// Open a CSV file for reading.
func openCSVRead(filename string) *csv.Reader {
	file, err := os.Open(filename)
	if err != nil {
		glog.Fatalf("Failed to open %q: %v", filename, err)
	}
	return csv.NewReader(file)
}
