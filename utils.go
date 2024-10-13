package main

import (
	"errors"
	"os"
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
