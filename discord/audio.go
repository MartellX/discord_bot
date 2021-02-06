package discord

import (
	"MartellX/discord_bot/utils"
	"MartellX/discord_bot/vk"
	"errors"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"time"
)

var AudioSessions = map[string]*AudioSession{}

func CloseAllAudioSessions() {
	for v, k := range AudioSessions {
		k.Close()
		delete(AudioSessions, v)
	}
}

type Queue []*vk.Track

func (q *Queue) SkipTrack() {
	*q = (*q)[1:]
}

func (q *Queue) SkipTrackN(n int) {
	*q = (*q)[n:]
}

func (q *Queue) GetFirstTrack() *vk.Track {
	return (*q)[0]
}

func (q *Queue) AddTrack(track *vk.Track) {
	*q = append(*q, track)
}

type AudioSession struct {
	VC           *discordgo.VoiceConnection
	Queue        Queue
	NowPlaying   *vk.Track
	AudioEncoder *utils.AudioEncoder

	Session   *discordgo.Session
	ChannelId string
	GuildID   string

	outChannel    chan []byte
	ffmpegControl *uint8
	IsPaused      bool
}

func NewAudioSession(VC *discordgo.VoiceConnection, session *discordgo.Session, channel string, guildID string) *AudioSession {
	var ff uint8 = 0
	as := &AudioSession{
		VC:            VC,
		Session:       session,
		ChannelId:     channel,
		Queue:         make([]*vk.Track, 0, 100),
		ffmpegControl: &ff,
		GuildID:       guildID,
		AudioEncoder:  utils.NewAudioEncoder(),
	}
	AudioSessions[guildID] = as
	return as
}

func (as *AudioSession) AddTrack(track *vk.Track) {
	playing := "Добавляю в очередь: `" + track.Artist + " - " + track.Title + "[" + track.GetDuration().String() + "]`"
	as.Session.ChannelMessageSend(as.ChannelId, playing)
	as.Queue.AddTrack(track)
}

func (as *AudioSession) AddTracks(tracks []*vk.Track) {
	message := discordgo.MessageEmbed{
		Title:       "Добавляю в очередь:",
		Color:       0x0062ff,
		Description: "",
	}
	for _, track := range tracks {
		playing := "`" + track.Artist + " - " + track.Title + "[" + track.GetDuration().String() + "]`\n"
		message.Description += playing
		as.Queue.AddTrack(track)

	}

	as.Session.ChannelMessageSendEmbed(as.ChannelId, &message)

}

func (as *AudioSession) SkipTrack() {
	if !as.IsPaused {
		as.Pause()

		defer func(as *AudioSession) {
			time.Sleep(5 * time.Millisecond)
			if len(as.Queue) > 0 {
				as.Play()
			} else {
				as.Session.ChannelMessageSend(as.ChannelId, "Не осталось треков в очереди")
				as.IsPaused = false
			}
		}(as)
	}

	//*as.ffmpegControl = 0
	as.AudioEncoder.SetStatus(utils.Stop)
	as.outChannel = nil
	as.Queue.SkipTrack()
}

func (as *AudioSession) SkipTrackN(n int) {
	if !as.IsPaused {
		as.Pause()

		defer func(as *AudioSession) {
			time.Sleep(5 * time.Millisecond)
			if len(as.Queue) > 0 {
				as.Play()
			} else {
				as.Session.ChannelMessageSend(as.ChannelId, "Не осталось треков в очереди")
				as.IsPaused = false
			}
		}(as)
	}

	//*as.ffmpegControl = 0
	as.AudioEncoder.SetStatus(utils.Stop)
	as.outChannel = nil
	as.Queue.SkipTrackN(n)
}

func (as *AudioSession) Clear() {
	if !as.IsPaused {
		as.Pause()
	}

	//*as.ffmpegControl = 0
	as.AudioEncoder.SetStatus(utils.Stop)
	as.outChannel = nil
	as.Queue = make([]*vk.Track, 0, 100)
	as.IsPaused = false
}

func (as *AudioSession) MoveTrack(from int, to int) {
	if from == to {
		return
	}
	movingTrack := as.Queue[from]
	//as.Queue = append(as.Queue[:from], as.Queue[from + 1:]...)
	if from > to {
		copy(as.Queue[to+1:from+1], as.Queue[to:from])

	} else {
		copy(as.Queue[from:to], as.Queue[from+1:to+1])
	}
	as.Queue[to] = movingTrack
}
func (as *AudioSession) CheckForConnectionAndChangeVC(message *discordgo.Message) (err error) {
	guild, _ := as.Session.State.Guild(message.GuildID)

	channelId := ""
	for _, state := range guild.VoiceStates {
		if state.UserID == message.Author.ID {
			channelId = state.ChannelID
		}
	}

	if channelId == "" {
		as.Session.ChannelMessageSendReply(as.ChannelId, "Не могу обнаружить голосовой канал", message.Reference())
		return errors.New("cant connect to the channel")
	}

	vc, ok := as.Session.VoiceConnections[message.GuildID]
	if !ok {
		vc, _ = as.Session.ChannelVoiceJoin(guild.ID, channelId, false, true)
	} else if vc.ChannelID != channelId {
		vc.Close()
		vc, _ = as.Session.ChannelVoiceJoin(guild.ID, channelId, false, true)
	}

	as.VC = vc
	return nil

}

func (as *AudioSession) Play() error {

	//if *as.ffmpegControl == 2 {
	//	err := errors.New("is already playoing")
	//	fmt.Println(err)
	//	return err
	//}

	as.IsPaused = false

	err := as.VC.Speaking(true)

	if err != nil {
		fmt.Println(err)
	}

	for !as.VC.Ready {
		time.Sleep(10 * time.Millisecond)
	}

	as.AudioEncoder.SetStatus(utils.Run)
	go func() {
		defer as.VC.Speaking(false)

	PlayingLoop:
		// While is track in queue
		for len(as.Queue) > 0 {

			// take first track in queue
			as.NowPlaying = as.Queue.GetFirstTrack()

			time.Sleep(50 * time.Millisecond)
			// if outChannel is not nil, track was paused and is no need for restarting it
			if as.outChannel == nil {
				playing := "Сейчас играет: `" + as.NowPlaying.Artist + " - " + as.NowPlaying.Title + "[" + as.NowPlaying.GetDuration().String() + "]`"
				as.Session.ChannelMessageSend(as.ChannelId, playing)
				as.AudioEncoder.SetInput(as.NowPlaying.Url, "")
				//as.outChannel, as.ffmpegControl = utils.ReadFileToOpus(as.NowPlaying.Url)
				as.outChannel = as.AudioEncoder.OutChannel()
			}

			// for each []byte in channel
			for out := range as.outChannel {
				// if audio session is paused or skipping track
				if as.IsPaused || as.outChannel == nil {
					break PlayingLoop
				}
				as.VC.OpusSend <- out
				as.NowPlaying.PlayedTime = time.Millisecond * time.Duration(as.AudioEncoder.PlayedSecs*1000)
			}
			go as.SkipTrack()
			break
		}
	}()

	return nil
}

func (as *AudioSession) Pause() {
	//*as.ffmpegControl = 1
	as.AudioEncoder.SetStatus(utils.Pause)
	as.IsPaused = true
}

func (as *AudioSession) Resume() {
	as.IsPaused = false
	//*as.ffmpegControl = 2
	as.Play()
}

func (as *AudioSession) Close() {
	as.Pause()
	//*as.ffmpegControl = 0
	as.AudioEncoder.SetStatus(0)
}
