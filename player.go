package main

import (
	"fmt"
	"os"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
	"github.com/faiface/beep/wav"
)

type AudioPlayer struct {
	buffers   []*beep.Buffer
	streamers []*beep.StreamSeeker
	ctrl      *beep.Ctrl
	tick      int
}

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

func NewAudioPlayer(audios []string, beatInterval time.Duration) (*AudioPlayer, error) {
	var buffers []*beep.Buffer
	for i, audios := range audios {
		streamer, audioFormat := Read(audios)

		if i == 0 {
			format = audioFormat
			err := speaker.Init(format.SampleRate, 44100/30)
			if err != nil {
				return nil, fmt.Errorf("error while initializing speaker: %v", err)
			}
		}
		buffer := beep.NewBuffer(format)
		buffer.Append(streamer)
		buffers = append(buffers, buffer)
		streamer.Close()
	}

	streamers := make([]*beep.StreamSeeker, len(buffers))
	for i := range buffers {
		streamer := buffers[i].Streamer(0, buffers[i].Len())
		streamers[i] = &streamer
	}

	return &AudioPlayer{
		buffers:   buffers,
		streamers: streamers,
		tick:      0,
	}, nil
}

func (ap *AudioPlayer) PlayTick(index int) {
	if ap.ctrl != nil {
		ap.ctrl.Streamer = nil
	}

	*ap.streamers[index] = ap.buffers[index].Streamer(0, ap.buffers[index].Len())
	ap.ctrl = &beep.Ctrl{
		Streamer: *ap.streamers[index],
		Paused:   false,
	}

	speaker.Play(ap.ctrl, beep.Callback(func() {
		ap.tick++
	}))
}
