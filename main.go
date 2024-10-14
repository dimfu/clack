package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/eiannone/keyboard"
	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
)

var (
	ctrl    *beep.Ctrl
	timeSig TimeSignature
	format  beep.Format
	buffers []*beep.Buffer
	audios  = []string{"./static/hi.wav", "./static/lo.wav"}

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
	if err := keyboard.Open(); err != nil {
		panic(err)
	}
	defer func() {
		_ = keyboard.Close()
	}()

	for i, audios := range audios {
		streamer, audioFormat := Read(audios)
		defer streamer.Close()

		if i == 0 {
			format = audioFormat
			err := speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
			if err != nil {
				log.Fatalf("error while initializing speaker: %v", err)
			}
		}
		buffer := beep.NewBuffer(format)
		buffer.Append(streamer)
		buffers = append(buffers, buffer)
	}

	beatInterval := time.Duration(60.0 / float64(*tempo) * float64(time.Second))

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	pauseChan := make(chan bool)

	playtick := func(index int) {
		shot := buffers[index].Streamer(0, buffers[index].Len())
		setCtrl := &beep.Ctrl{Streamer: shot, Paused: false}
		ctrl = setCtrl
		speaker.Play(ctrl)
	}

	ticker := time.NewTicker(beatInterval)
	defer ticker.Stop()

	nextTick := time.Now()
	tick := 0

	fmt.Println("Press ESC to quit")
	playtick(1) // init first tick

	go func() {
		for {
			_, key, err := keyboard.GetKey()
			if err != nil {
				// handling this error like this because whenever i quit during paused state
				// it will panicked over this error. had to do it like this
				if err.Error() == "operation canceled" {
					break
				}
				panic(err)
			}

			if key == keyboard.KeySpace {
				ctrl.Paused = !ctrl.Paused
				pauseChan <- ctrl.Paused
			}

			if key == keyboard.KeyEsc {
				sig <- syscall.SIGTERM
				break
			}
		}
	}()

	for {
		select {
		case now := <-ticker.C:
			if ctrl.Paused {
				// handle terminate signal when paused, otherwise it wont close the app lol
				select {
				case <-sig:
					speaker.Clear()
					return
				default:
					continue
				}
			}
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
		case paused := <-pauseChan:
			if paused {
				fmt.Println("paused")
			} else {
				fmt.Println("resumed")
			}
		case <-sig:
			speaker.Clear()
			return
		}
	}
}
