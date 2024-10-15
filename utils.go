package main

import (
	"errors"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/faiface/beep"
	"github.com/faiface/beep/wav"
)

func Read(path string) (beep.StreamSeekCloser, beep.Format) {
	f, err := os.Open(path)
	if err != nil {
		panic("reading audio file failed" + err.Error())
	}

	streamer, format, err := wav.Decode(f)
	if err != nil {
		panic("error while decoding audio" + err.Error())
	}

	return streamer, format
}

func ValidTempo(input int64) bool {
	return input > MIN_TEMPO && input < MAX_TEMPO
}

func ValidTimeSig(input string) (TimeSignature, error) {
	parts := strings.Split(input, "/")
	if len(parts) != 2 {
		return TimeSignature{}, errors.New("invalid time signature format")
	}

	beats, err1 := strconv.ParseInt(parts[0], 10, 64)
	noteValue, err2 := strconv.ParseInt(parts[1], 10, 64)
	if err1 != nil || err2 != nil {
		return TimeSignature{}, errors.New("invalid number in time signature")
	}

	for _, ts := range TIME_SIGNATURES {
		if ts.Beats == beats && ts.NoteValue == noteValue {
			return TimeSignature{Beats: beats, NoteValue: noteValue}, nil
		}
	}

	return TimeSignature{}, errors.New("time signature not found")
}

func runCmd(name string, arg ...string) {
	cmd := exec.Command(name, arg...)
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	if err != nil {
		log.Fatal(err.Error())
	}
}

func ClearTerminal() {
	switch runtime.GOOS {
	case "darwin":
		runCmd("clear")
	case "linux":
		runCmd("clear")
	case "windows":
		runCmd("cmd", "/c", "cls")
	default:
		runCmd("clear")
	}
}

func UserHomeDir() string {
	if runtime.GOOS == "windows" {
		home := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		return home + "\\"
	}
	return os.Getenv("HOME") + "/"
}
