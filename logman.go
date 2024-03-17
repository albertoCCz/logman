package logman

import (
	"os"
	"time"
	"strings"
	"strconv"
	"fmt"
	"errors"
)

type LogMan struct {
	path          string
	duration      time.Duration;
	size          int64;
	lastFileStart time.Time;
	layout        string;
}


func isLetter(c byte) bool {
	return ('a' <= c && c <= 'z') || ('A' <= c && c <= 'Z')
}

const (
	B   = "B"
	KiB = "KiB"
	MiB = "MiB"
	GiB = "GiB"
)

func parseFileSize(size string) (sz int64, err error) {
	var (
		num  int64
		mult int64
		unit string
	)

	for i := len(size)-1; i >= 0 && isLetter(size[i]); i-- {
		unit = string(size[i]) + unit
 	}

	switch unit {
	case B:
		mult = 1<<(10*0)
	case KiB:
		mult = 1<<(10*1)
	case MiB:
		mult = 1<<(10*2)
	case GiB:
		mult = 1<<(10*3)
	default:
		err = errors.New("logman: unknown file size units " + "'" + unit + "'")
		return
	}

	num, err = strconv.ParseInt(size[:len(size) - len(unit)], 10, 64)
	if err != nil {
		err = errors.New("logman: could not parse file size " + "'" + string(num) + "'")
		return
	}

	sz = mult * num
	return
}

func New(path string, duration string, size string) *LogMan {
	dur, err := time.ParseDuration(duration)
	if err != nil {
		fmt.Errorf("[ERROR] Could not parse duration: %s\n", err)
		os.Exit(1)
	}

	if dur < time.Second {
		fmt.Println("[ERROR] Duration must be greater than, or equal to, 1 second")
		os.Exit(1)
	}

	if !strings.HasSuffix(path, "/") {
		path += "/"
	}

	sz, err := parseFileSize(size)
	if err != nil {
		fmt.Printf("[ERROR] Could not parse file size: %s\n", err)
		os.Exit(1)
	}

	return &LogMan{
		path: path,
		duration: dur,
		size: sz,
		layout: time.RFC3339Nano,
	}
}

func (lm *LogMan) getLogFileName(logSize int64) (name string, err error) {
	// first write
	if lm.lastFileStart.IsZero() {
		lm.lastFileStart = time.Now()
		name = lm.lastFileStart.Format(lm.layout) + ".log"
		return
	}

	// duration interval passed
	if slots := time.Since(lm.lastFileStart) / lm.duration; slots >= 1 {
		newTime := lm.lastFileStart.Add(slots * lm.duration)
		lm.lastFileStart = newTime
		name = newTime.Format(lm.layout) + ".log"
		return
	}

	// file size limit reached
	lastFileName := lm.path + lm.lastFileStart.Format(lm.layout) + ".log"
	fInfo, err := os.Lstat(lastFileName)
	if err != nil {
		err = errors.New("logman: could not lstat " + lastFileName)
		return
	}
	if size := fInfo.Size() + logSize; size >= lm.size {
		newTime := time.Now()
		lm.lastFileStart = newTime
		name = newTime.Format(lm.layout) + ".log"
		return
	}

	name = lm.lastFileStart.Format(lm.layout) + ".log"
	return
}

func (lm *LogMan) Write(p []byte) (n int, err error) {
	fileName, err := lm.getLogFileName(int64(len(p)))
	if err != nil {
		err = errors.New(fmt.Sprintf("logman: could not retrieve log file name: %s", err))
		return
	}

	f, err := os.OpenFile(lm.path + fileName, os.O_WRONLY | os.O_APPEND | os.O_CREATE, 0666)
	if err != nil {
		err = errors.New(fmt.Sprintf("logman: could not open log file: %s", err))
		return
	}

	return f.Write(p)
}
