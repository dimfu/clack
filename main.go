package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/eiannone/keyboard"
	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
)

var (
	command string
	ctrl    *beep.Ctrl
	timeSig TimeSignature
	format  beep.Format
	buffers []*beep.Buffer
	audios  = []string{"./static/hi.wav", "./static/lo.wav"}

	//flags
	tempo   = flag.Int64("tempo", 120, "the speed at which a passage of this metronome should be played")
	timesig = flag.String("timesig", "4/4", "indicate how many beats are in each measure")
	config  = flag.String("config", "", "the name of your saved config settings")
)

func printHelp() {
	fmt.Println("Usage:")
	fmt.Println("  clack [command] [flags]")
	fmt.Println("\nCommands:")
	fmt.Println("  run                   Run the metronome")
	fmt.Println("  add <name>            Add metronome config")
	fmt.Println("  delete <name>         Remove specific config")
	fmt.Println("  siglist               Print the list of available time signatures")
	fmt.Println("  help                  Print this help view")
	fmt.Println("\nFlags:")
	fmt.Println("  --tempo <bpm>         Set the tempo (default: 120 bpm)")
	fmt.Println("  --timesig <value>     Set the time signature (default: 4/4)")
	fmt.Println("  --config <value>      Your saved config setting")
	fmt.Println("\nExamples:")
	fmt.Println("  clack siglist")
	fmt.Println("  clack --tempo 100 --timesig 3/4 run")
	fmt.Println("  clack --config=\"cfg1\" run")

	os.Exit(0)
}

func printSigList() {
	timeSigArr := []string{}
	for _, v := range TIME_SIGNATURES {
		timeSigArr = append(timeSigArr, fmt.Sprintf("-%d/%d\n", v.Beats, v.NoteValue))
	}
	fmt.Println("Available Time Signatures:\n", strings.Join(timeSigArr, " "))
	os.Exit(0)
}

func init() {
	flag.Parse()
	flag.Usage = printHelp

	args := make([]string, 0)
	for i := len(os.Args) - len(flag.Args()) + 1; i < len(os.Args); {
		if i > 1 && os.Args[i-2] == "--" {
			break
		}
		args = append(args, flag.Arg(0))
		if err := flag.CommandLine.Parse(os.Args[i:]); err != nil {
			log.Fatal("error while parsing arguments")
		}

		i += 1 + len(os.Args[i:]) - len(flag.Args())
	}
	args = append(args, flag.Args()...)

	if len(args) < 1 {
		flag.Usage()
		os.Exit(0)
	}

	command = args[0]

	switch command {
	case "run":
		if len(*config) > 0 {
			key := *config
			cfg, err := LoadConf(key)
			if err != nil {
				log.Fatal(err)
			}
			tempo = &cfg.Tempo
			timesig = &cfg.Timesig
		}

		if !ValidTempo(*tempo) {
			log.Fatalf("tempo is not valid make sure its above %v and below %v", MIN_TEMPO, MAX_TEMPO)
		}

		validSig, err := ValidTimeSig(*timesig)
		if err != nil {
			log.Fatalf("%v", err.Error())
		}

		timeSig = validSig
		// continue
	case "add":
		if len(args) > 1 {
			confName := args[1]
			err := CreateConf(Config{Key: confName, Tempo: *tempo, Timesig: *timesig})
			if err != nil {
				log.Fatal(err)
			}
			fmt.Printf("%v is added to the config", confName)
			os.Exit(0)
		}
	case "delete":
		if len(args) > 1 {
			confName := args[1]
			err := DeleteConfig(confName)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Printf("%v is deleted from the config", confName)
			os.Exit(0)
		}
	case "siglist":
		printSigList()
	case "help":
		flag.Usage()
	default:
		log.Fatal("command not found, please refer to `help` command")
		os.Exit(0)
	}
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

	ClearTerminal()
	fmt.Println("Press [ESC] to quit, [SPACEBAR] to pause the metronome")
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
