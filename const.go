package main

type TimeSignature struct {
	Beats     int64 // number of beats per meassure
	NoteValue int64 // note that represent that one beat
}

const (
	MIN_TEMPO = 0
	MAX_TEMPO = 600
)

var TIME_SIGNATURES = []TimeSignature{
	{4, 4},
	{3, 4},
	{2, 4},
	{2, 2},
	{3, 8},
	{6, 8},
	{9, 8},
	{12, 8},
	{5, 4},
	{6, 4},
}
