package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
)

var (
	timeSig TimeSignature
	format  beep.Format
	buffers []*beep.Buffer

	// flags
	tempo   = flag.Int64("tempo", 120, "the speed at which a passage of this metronome should be played")
	timesig = flag.String("timesig", "4/4", "indicate how many beats are in each measure")
)

func init() {
	flag.Parse()
	if !ValidTempo(*tempo) {
		log.Fatalf("tempo is not valid make sure its above %v and below %v", MIN_TEMPO, MAX_TEMPO)
	}

	validSig, err := ValidTimeSig(*timesig)
	if err != nil {
		log.Fatalf("%v", err.Error())
	}

	timeSig = validSig
}

func main() {
	audios := []string{"./static/hi.wav", "./static/lo.wav"}

	for i, audios := range audios {
		streamer, audioFormat := Read(audios)
		defer streamer.Close()

		if i == 0 {
			format = audioFormat
			speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
		}
		buffer := beep.NewBuffer(format)
		buffer.Append(streamer)
		buffers = append(buffers, buffer)
	}

	beatInterval := time.Duration(60.0 / float64(*tempo) * float64(time.Second))

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	playtick := func(index int) {
		shot := buffers[index].Streamer(0, buffers[index].Len())
		speaker.Play(shot)
	}

	ticker := time.NewTicker(beatInterval)
	defer ticker.Stop()

	nextTick := time.Now()
	tick := 0

	playtick(1)

	for {
		select {
		case now := <-ticker.C:
			drift := now.Sub(nextTick)
			if drift > 10*time.Millisecond || drift < -10*time.Millisecond {
				nextTick = now
			}

			nextTick = nextTick.Add(beatInterval)
			tick++
			audioIdx := 0

			if tick%int(timeSig.Beats) == 0 {
				audioIdx = 1
			}

			playtick(audioIdx)
		case <-sig:
			speaker.Clear()
			return
		}
	}
}
