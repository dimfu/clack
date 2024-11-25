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
	"github.com/gosuri/uilive"
)

var (
	command      string
	timeSig      TimeSignature
	format       beep.Format
	audios       = []string{"static/hi.wav", "static/lo.wav"}
	beatInterval time.Duration

	//flags
	tempo   = flag.Int64("tempo", 120, "the speed at which a passage of this metronome should be played")
	timesig = flag.String("timesig", "4/4", "indicate how many beats are in each measure")
	config  = flag.String("config", "", "the name of your saved config settings")

	circleEmpty  rune = '\u25CB' // ○
	circleFilled rune = '\u25CF' // ●
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

	// init beat arr to visualize the beat
	beatsArr := make([]rune, timeSig.Beats)
	for i := range beatsArr {
		beatsArr[i] = circleEmpty
	}

	switch timeSig.Beats {
	case 6, 9, 12:
		beatInterval = time.Duration((60.0 / (float64(*tempo) / 0.5) * float64(time.Second)))
	default:
		beatInterval = time.Duration(60.0 / float64(*tempo) * float64(time.Second))
	}

	player, err := NewAudioPlayer(audios, beatInterval)
	if err != nil {
		log.Fatal(err.Error())
	}

	sig := make(chan os.Signal, 1)
	pauseChan := make(chan bool)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	ticker := time.NewTicker(beatInterval)
	defer ticker.Stop()

	writer := uilive.New()
	writer.Start()

	ClearTerminal()

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
				player.ctrl.Paused = !player.ctrl.Paused
				pauseChan <- player.ctrl.Paused
			}

			if key == keyboard.KeyEsc {
				sig <- syscall.SIGTERM
				break
			}
		}
	}()

	for {
		select {
		case <-ticker.C:
			if player.ctrl != nil && player.ctrl.Paused {
				select {
				case <-sig:
					speaker.Clear()
					return
				default:
					continue
				}
			}

			modTick := player.tick % int(timeSig.Beats)

			audioIdx := 1
			if modTick != 0 {
				audioIdx = 0
			}

			player.PlayTick(audioIdx)

			var beatStr string
			for idx := range beatsArr {
				if idx == modTick {
					beatsArr[idx] = circleFilled
				} else {
					beatsArr[idx] = circleEmpty
				}
				beatStr += fmt.Sprintf("%v", string(beatsArr[idx]))
			}

			fmt.Fprintf(writer, "%s\n", beatStr)
			fmt.Fprintf(writer, "Press [ESC] to quit, [SPACEBAR] to pause the metronome\n")
		case paused := <-pauseChan:
			if player.ctrl != nil {
				player.ctrl.Paused = paused
			}
			if paused {
				fmt.Fprintf(writer, "Paused\n")
			} else {
				fmt.Fprintf(writer, "Resumed\n")
			}
		case <-sig:
			speaker.Clear()
			writer.Stop()
			return
		}
	}
}
