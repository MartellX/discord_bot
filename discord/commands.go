package discord

import (
	"MartellX/discord_bot/utils"
	"MartellX/discord_bot/vk"
	"fmt"
	"github.com/Kintar/dgc"
	"github.com/bwmarrin/discordgo"
	"math/rand"
	"net/url"
	"strconv"
	"time"
)

func onHelp(ctx *dgc.Ctx) {
	commands := ctx.Router.Commands

	pages := []*discordgo.MessageEmbed{}

	var message *discordgo.MessageEmbed
	for i, command := range commands {
		if i%5 == 0 {
			message = &discordgo.MessageEmbed{
				Title: "Список команд (" + strconv.Itoa(i/5+1) + "/" + strconv.Itoa((len(commands)-1)/5+1) + ")",
				Description: "Напишите `yo help <имя команды>` для помощи по конкректной команде\n" +
					"Для получения большинства треков с вк используется прокси (если оно задано), поэтому иногда ответы могут быть долгими (до 20 минут). " +
					"Используйте `yo changeproxy` для попытки смены прокси. Прокси используется только для получения информации с ВК, не для воспроизведения и всего остального.",
				Color:  0x0062ff,
				Fields: []*discordgo.MessageEmbedField{},
			}
			pages = append(pages, message)
		}
		als := ""
		if len(command.Aliases) > 0 {
			als = fmt.Sprint(command.Aliases)
		}
		field := discordgo.MessageEmbedField{
			Name:  fmt.Sprintln(command.Name, als),
			Value: fmt.Sprint("_", command.Description, "_\n__", "Использование:__ ", "`", ctx.Router.Prefixes[0], command.Usage, "`"),
		}
		message.Fields = append(message.Fields, &field)
	}

	m, _ := ctx.Session.ChannelMessageSendEmbed(ctx.Event.ChannelID, pages[0])

	NewMessage(ctx.Session, nil, m.ID, m.ChannelID, pages, nil)

}

func onPing(ctx *dgc.Ctx) {
	err := ctx.RespondTextEmbed("", &discordgo.MessageEmbed{
		Color:       0x0062ff,
		Description: "Pong",
	})

	if err != nil {
		fmt.Println(err)
	}
}

func onConnect(ctx *dgc.Ctx) {
	sess := ctx.Session
	mess := ctx.Event

	as, ok := AudioSessions[mess.GuildID]

	if !ok {
		as = NewAudioSession(nil, sess, mess.ChannelID, mess.GuildID)
	}
	err := as.CheckForConnectionAndChangeVC(mess.Message)
	if (!as.IsPaused) && as.AudioEncoder.Status() != utils.Run && err == nil {
		as.Play()
	}
}

func onPlay(ctx *dgc.Ctx) {
	sess := ctx.Session
	event := ctx.Event

	searchArg := ctx.Arguments.Raw()

	fmt.Println("Gettin argument: " + searchArg)
	var tracks []*vk.Track
	_, err := url.ParseRequestURI(searchArg)
	isPlaylist := false
	if err != nil {
		tracks, err = vk.SearchAudio(searchArg)
	} else {
		fmt.Println("This is playlist")
		tracks, err = vk.GetPlaylist(searchArg)
		isPlaylist = true
	}
	if err != nil {
		fmt.Println(err)
		ctx.RespondText("Произошла ошибка")
		return
	}
	if len(tracks) <= 0 {
		ctx.RespondText("Ничего не найдено")
		return
	}

	as, ok := AudioSessions[event.GuildID]

	if !ok {
		as = NewAudioSession(nil, sess, event.ChannelID, event.GuildID)
	}

	prevQueueLen := len(as.Queue)

	if isPlaylist {
		as.AddTracks(tracks)
	} else {
		as.AddTrack(tracks[0])
	}
	err = as.CheckForConnectionAndChangeVC(event.Message)
	if (!as.IsPaused || prevQueueLen == 0) && as.AudioEncoder.Status() != utils.Run && err == nil {
		as.Play()
	}
}

func onVksearch(ctx *dgc.Ctx) {

	sess := ctx.Session
	event := ctx.Event

	searchArg := ctx.Arguments.Raw()

	tracks, err := vk.SearchAudio(searchArg)
	if err != nil {
		fmt.Println(err)
		ctx.RespondText("Произошла ошибка")
		return
	}
	if len(tracks) <= 0 {
		ctx.RespondText("Ничего не найдено")
		return
	}

	pages := []*discordgo.MessageEmbed{}

	var page *discordgo.MessageEmbed

	for i, track := range tracks {
		if i%10 == 0 {
			page = &discordgo.MessageEmbed{
				Title: "(" + strconv.Itoa(i/10+1) + "/" + strconv.Itoa(len(tracks)/10) +
					")Найденные треки:",
				Description: "",
				Color:       0x0062ff,
			}
			pages = append(pages, page)
		}

		page.Description += strconv.Itoa(i+1) + ". `" + track.String() + "`\n"

	}

	m, err := sess.ChannelMessageSendEmbed(event.ChannelID, pages[0])

	as, ok := AudioSessions[event.GuildID]

	if !ok {
		as = NewAudioSession(nil, sess, event.ChannelID, event.GuildID)
	}

	NewMessage(sess, as, m.ID, m.ChannelID, pages, map[string]interface{}{"tracks": tracks}).listenForMessages()

}

func onPause(ctx *dgc.Ctx) {
	event := ctx.Event
	as, ok := AudioSessions[event.GuildID]
	if ok {
		as.Pause()
		ctx.RespondText("Ставлю на паузу")
	}
}

func onResume(ctx *dgc.Ctx) {
	event := ctx.Event
	as, ok := AudioSessions[event.GuildID]
	if ok && as.IsPaused {
		if len(as.Queue) == 0 {
			ctx.RespondText("Нет треков в очереди")
			return
		}
		err := as.CheckForConnectionAndChangeVC(event.Message)
		if as.AudioEncoder.Status() != utils.Run && err == nil {
			as.Resume()
		}
	}
}

func onQueue(ctx *dgc.Ctx) {
	event := ctx.Event
	as, ok := AudioSessions[event.GuildID]
	if ok {
		message := discordgo.MessageEmbed{
			Title:  "Текущая очередь",
			Color:  0x0062ff,
			Fields: []*discordgo.MessageEmbedField{},
		}

		pages := []*discordgo.MessageEmbed{&message}
		queue := as.Queue
		if len(queue) == 0 {
			message.Description = "Нет треков в очереди"
		} else {
			sumDuration := time.Duration(0)
			message.Description = ""
			first := queue.GetFirstTrack()
			sumDuration += first.GetDuration()

			message.Fields = append(message.Fields, &discordgo.MessageEmbedField{
				"__Сейчас играет:__", "`" + first.String() + "`", false,
			})

			if len(queue) > 1 {
				nextField := &discordgo.MessageEmbedField{
					Name:   "__Следующие в очереди:__",
					Value:  "",
					Inline: false,
				}
				message.Fields = append(message.Fields, nextField)
				for i, track := range queue[1:] {
					if i >= 10 && i%10 == 0 {
						message := discordgo.MessageEmbed{
							Title:  "Текущая очередь",
							Color:  0x0062ff,
							Fields: []*discordgo.MessageEmbedField{},
						}
						nextField = &discordgo.MessageEmbedField{
							Name:   "__Следующие в очереди:__",
							Value:  "",
							Inline: false,
						}
						message.Fields = append(message.Fields, nextField)
						pages = append(pages, &message)
					}
					sumDuration += track.GetDuration()
					nextField.Value += strconv.Itoa(i+1) + ". `" + track.String() + "`\n"
				}
			}
			message.Footer = &discordgo.MessageEmbedFooter{
				Text: "Всего треков: " + strconv.Itoa(len(queue)) + "\nОбщая продолжительность: `" + sumDuration.String() + "`",
			}

		}

		session := ctx.Session

		m, err := session.ChannelMessageSendEmbed(event.ChannelID, &message)

		NewMessage(session, as, m.ID, m.ChannelID, pages, nil)

		if err != nil {
			fmt.Println(err)
		}
	}
}

func onSkip(ctx *dgc.Ctx) {

	event := ctx.Event
	as, ok := AudioSessions[event.GuildID]
	if ok {
		if len(as.Queue) == 0 {
			ctx.RespondText("Нет треков в очереди!")
			return
		}
		n, err := ctx.Arguments.Get(0).AsInt()
		if err != nil {
			ctx.RespondText("Пропускаю `" + as.NowPlaying.Artist + " - " + as.NowPlaying.Title + "`")
			as.SkipTrack()
		} else {
			message := discordgo.MessageEmbed{
				Title:       "Пропускаю треки:",
				Color:       0x0062ff,
				Description: "",
			}
			if n < 0 {
				ctx.RespondText("Неверный параметр")
				return
			}
			if n > len(as.Queue) {
				ctx.RespondText("Слишком большое число (используйте `" + ctx.Router.Prefixes[0] + "clear` для очистки очереди)")
				return
			}
			for _, track := range as.Queue[:n] {
				skipping := "`" + track.Artist + " - " + track.Title + "[" + track.GetDuration().String() + "]`\n"
				message.Description += skipping
			}
			ctx.RespondEmbed(&message)
			as.SkipTrackN(n)
		}
	}
}

func onClear(ctx *dgc.Ctx) {

	event := ctx.Event
	as, ok := AudioSessions[event.GuildID]
	if ok {
		if len(as.Queue) == 0 {
			ctx.RespondText("Нет треков в очереди!")
			return
		}
		as.Clear()
		ctx.RespondText("Очередь очищена")
	}
}

func onMove(ctx *dgc.Ctx) {
	event := ctx.Event
	as, ok := AudioSessions[event.GuildID]
	if ok {
		n, err := ctx.Arguments.Get(0).AsInt()
		if err != nil {
			ctx.RespondText("Не дана позиция трека в очереди")
			return
		}
		if !(1 < n && n < len(as.Queue)) {
			ctx.RespondText("Такой позиции нет в очереди или он уже следующий")
			return
		}
		track := as.Queue[n]
		m, err := ctx.Arguments.Get(1).AsInt()
		if err != nil {
			as.MoveTrack(n, 1)
			ctx.RespondText("`" + track.String() + "` будет следующим")
		} else {
			if !(1 <= m && m < len(as.Queue)) {
				ctx.RespondText("Целевой позиции нет в очереди")
				return
			}
			as.MoveTrack(n, m)
			ctx.RespondText("`" + track.String() + "` перемещен на " + strconv.Itoa(m) + " позицию")
		}
	}
}

func onShuffle(ctx *dgc.Ctx) {

	event := ctx.Event
	as, ok := AudioSessions[event.GuildID]
	if ok {
		if len(as.Queue) > 1 {
			rand.Shuffle(len(as.Queue)-1, func(i, j int) {
				as.Queue[i+1], as.Queue[j+1] = as.Queue[j+1], as.Queue[i+1]
			})
			ctx.RespondText("Очередь перемешана")
		}
	}
}

func onLeave(ctx *dgc.Ctx) {

	sess := ctx.Session
	event := ctx.Event
	guild := event.GuildID

	vc, ok := sess.VoiceConnections[guild]

	if ok {
		as, ok := AudioSessions[guild]
		if ok {
			as.Pause()
			as.IsPaused = false
		}
		vc.Disconnect()
	}

}

func onTime(ctx *dgc.Ctx) {

	event := ctx.Event
	as, ok := AudioSessions[event.GuildID]
	if ok {
		if len(as.Queue) > 0 {
			ctx.RespondText("Осталось `" + (as.Queue.GetFirstTrack().GetDuration() - as.Queue.GetFirstTrack().PlayedTime).String() + "` до конца текущего трека")
		}
	}
}

func onSwitchProxy(ctx *dgc.Ctx) {
	ctx.RespondText("Пытаюсь сменить прокси")
	if ok, err := vk.SwitchProxy(); ok {
		ctx.RespondText("Прокси сменён")
	} else if err != nil {
		fmt.Println("Прокси уже сменяется")
	} else {
		ctx.RespondText("Не получилось сменить прокси, использую стандартное подключение (с воспроизведением треков с лицензией могут быть проблемы)")
	}

}
