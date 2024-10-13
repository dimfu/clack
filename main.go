package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
	"github.com/faiface/beep/wav"
)

func main() {
	f, err := os.Open("./static/hi.wav")
	if err != nil {
		panic("reading audio file failed" + err.Error())
	}

	streamer, format, err := wav.Decode(f)
	if err != nil {
		panic("error while decoding audio" + err.Error())
	}
	defer streamer.Close()

	buffer := beep.NewBuffer(format)
	buffer.Append(streamer)
	streamer.Close()

	speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))

	bpm := 120.0
	beatInterval := time.Duration(60.0 / bpm * float64(time.Second))

	done := make(chan bool)
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	playtick := func() {
		shot := buffer.Streamer(0, buffer.Len())
		speaker.Play(shot, beep.Callback(func() {
			done <- true
		}))
	}

	ticker := time.NewTicker(beatInterval)
	defer ticker.Stop()

	nextTick := time.Now()

	playtick()

	for {
		select {
		case <-done:
		case now := <-ticker.C:
			drift := now.Sub(nextTick)
			if drift > 10*time.Millisecond || drift < -10*time.Millisecond {
				nextTick = now
			}
			nextTick = nextTick.Add(beatInterval)

			playtick()
		case <-sig:
			speaker.Clear()
			return
		}
	}
}
