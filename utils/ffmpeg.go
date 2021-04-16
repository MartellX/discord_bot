package utils

import (
	"MartellX/discord_bot/config"
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"time"

	"layeh.com/gopus"
)

var ffmpegPath string

func init() {
	path := config.Cfg.FFMPEGPATH
	if path == "" {
		panic("Ffmpeg path is not set")
	} else {
		ffmpegPath = path
	}
}

const (
	Stop = iota
	Pause
	Run
)

type AudioEncoder struct {
	ffmpegExec  *exec.Cmd
	opusEncoder *gopus.Encoder

	status int
	out    chan []byte

	PlayedSecs float32
}

func NewAudioEncoder() *AudioEncoder {
	ae := &AudioEncoder{}
	ae.opusEncoder, _ = gopus.NewEncoder(48000, 2, gopus.Audio)
	return ae
}

func (ae *AudioEncoder) Status() int {
	return ae.status
}

func (ae *AudioEncoder) SetStatus(status int) {
	ae.status = status
}

func (ae *AudioEncoder) OutChannel() chan []byte {
	return ae.out
}

func (ae *AudioEncoder) SetInput(input string, customParams string) {

	if ae.ffmpegExec != nil {
		ae.status = 0
		ae.ffmpegExec.Process.Kill()
	}

	ae.out = make(chan []byte, 32)

	// statusId response for controlling ffmpeg encoding
	// 0 - kill encoding
	// 1 - pausing it
	// 2 - run status
	ae.status = 2

	ae.ffmpegExec = exec.Command(ffmpegPath, "-i", input, "-f", "s16le", "-ar", "48000", "-ac", "2", "pipe:1")
	ffmpegout, err := ae.ffmpegExec.StdoutPipe()
	if err != nil {
		fmt.Println(err)
		return
	}

	isLogs, ok := os.LookupEnv("CMD_LOG")

	if ok {
		isLog, err := strconv.ParseBool(isLogs)
		if err == nil {
			if isLog {
				ae.ffmpegExec.Stderr = os.Stderr
			} else {
				ae.ffmpegExec.Stderr = nil
			}
		}
	}
	//ae.ffmpegExec.Stderr = os.Stderr

	fmt.Println("Running command:\n", ae.ffmpegExec.String())
	err = ae.ffmpegExec.Start()

	ffmpegbuf := bufio.NewReaderSize(ffmpegout, 16834)

	go func() {
		defer close(ae.out)
		raw := make(chan []int16, 32)
		go func() {
			defer func() {
				close(raw)
				ffmpegout.Close()
			}()
			for {
				// pausing it
				for ae.status == 1 {
					time.Sleep(time.Millisecond * 10)
				}

				// killing ffmpeg reading
				if ae.status == 0 {
					break
				}

				audioBuffer := make([]int16, 960*2)
				err = binary.Read(ffmpegbuf, binary.LittleEndian, &audioBuffer)
				if err == io.EOF || err == io.ErrUnexpectedEOF {
					break
				}

				raw <- audioBuffer
			}
		}()

		sumBytes := 0
		for audioBuffer := range raw {
			// pausing it
			for ae.status == 1 {
				time.Sleep(time.Millisecond * 10)
			}
			// killing encoding
			if ae.status == 0 {
				break
			}
			opus, err := ae.opusEncoder.Encode(audioBuffer, 960, 960*2*2)
			if err != nil {
				fmt.Println("Encoding error,", err)
				break
			}
			ae.out <- opus
			sumBytes += len(audioBuffer)
			ae.PlayedSecs = float32(sumBytes) / (48000 * 2)
		}
	}()
}

func ReadFileToOpus(input string) (out chan []byte, status *uint8) {

	out = make(chan []byte, 8)

	// statusId response for controlling ffmpeg encoding
	// 0 - kill encoding
	// 1 - pausing it
	// 2 - run status
	statusId := uint8(2)
	run := exec.Command(ffmpegPath, "-i", input, "-f", "s16le", "-ar", "48000", "-ac", "2", "pipe:1")
	ffmpegout, err := run.StdoutPipe()
	if err != nil {
		fmt.Println(err)
		return
	}

	//run.Stderr = os.Stderr

	err = run.Start()

	ffmpegbuf := bufio.NewReaderSize(ffmpegout, 16834)

	encoder, err := gopus.NewEncoder(48000, 2, gopus.Audio)

	go func() {

		defer close(out)

		raw := make(chan []int16, 8)

		go func() {
			defer func() {
				close(raw)

				ffmpegout.Close()
				err := run.Process.Kill()
				fmt.Println("Killing ffmpeg")
				if err != nil {
					fmt.Println(err)
				}
			}()
			for {

				audioBuffer := make([]int16, 960*2)
				err = binary.Read(ffmpegbuf, binary.LittleEndian, &audioBuffer)
				if err == io.EOF || err == io.ErrUnexpectedEOF {
					break
				}

				// pausing it
				for statusId == 1 {
					time.Sleep(time.Millisecond * 10)
				}

				// killing ffmpeg reading
				if statusId == 0 {
					break
				}

				raw <- audioBuffer
			}
		}()

		for audioBuffer := range raw {
			// pausing it
			for statusId == 1 {
				time.Sleep(time.Millisecond * 10)
			}

			// killing encoding
			if statusId == 0 {
				break
			}

			opus, err := encoder.Encode(audioBuffer, 960, 960*2*2)
			if err != nil {
				fmt.Println("Encoding error,", err)
				break
			}

			out <- opus
		}
	}()

	return out, &statusId
}
