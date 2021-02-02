package discord

import (
	"fmt"
	"github.com/Kintar/dgc"
	"github.com/bwmarrin/discordgo"
	"os"
	"strings"
)

var token, _ = os.LookupEnv("DISCORD_TOKEN")

var (
	discordSession *discordgo.Session
	router         *dgc.Router
)

func Init() {
	ds, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Println("Error creating Discord session: ", err)
		return
	}

	ds.AddHandler(onReady)

	discordSession = ds
	err = ds.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	initRouter(ds)

}

func initRouter(session *discordgo.Session) {
	router = dgc.Create(&dgc.Router{
		Prefixes: []string{
			"yo ", "yo",
		},
		IgnorePrefixCase: true,
		BotsAllowed:      false,
		Commands:         []*dgc.Command{},
		PingHandler: func(ctx *dgc.Ctx) {
			ctx.RespondText("Yo!")
		},
	})
	router.RegisterCmd(&dgc.Command{
		Name:        "help",
		Aliases:     nil,
		Description: "Список команд",
		Usage:       "help",
		Flags:       nil,
		IgnoreCase:  true,
		SubCommands: nil,
		RateLimiter: nil,
		Handler:     onHelp,
	})
	//router.RegisterCmd(&dgc.Command{
	//	Name:        "ping",
	//	Aliases:     nil,
	//	Description: "Понг",
	//	Usage:       "ping",
	//	Flags:       nil,
	//	IgnoreCase:  true,
	//	SubCommands: nil,
	//	RateLimiter: nil,
	//	Handler: onPing,
	//})

	router.RegisterCmd(&dgc.Command{
		Name:        "vkplay",
		Aliases:     []string{"vkp"},
		Description: "Добавляет первый найденный трек по названию или плейлист по ссылке",
		Usage:       "vkplay <текстовый запрос или ссылка на плейлист>",
		Example: "vkplay Infected Mushrooms - Only Solutions\n" +
			router.Prefixes[0] + "vkp https://vk.com/music/album/-2000075157_10075157_62f69f45ff137f5bba\n" +
			router.Prefixes[0] + "vkplay https://vk.com/im?sel=152367856&z=audio_playlist-2000566053_3566053%2F28236d5f2ea5d5eef5",
		Flags:       nil,
		IgnoreCase:  true,
		SubCommands: nil,
		RateLimiter: nil,
		Handler:     onPlay,
	})

	router.RegisterCmd(&dgc.Command{
		Name:        "vksearch",
		Aliases:     []string{"vks"},
		Description: "Ищет и выдает список треков для добавления в очередь",
		Usage:       "vksearch <текстовый запрос>",
		Example:     "vksearch Infected Mushrooms",
		Flags:       nil,
		IgnoreCase:  true,
		SubCommands: nil,
		RateLimiter: nil,
		Handler:     onVksearch,
	})

	router.RegisterCmd(&dgc.Command{
		Name:        "queue",
		Aliases:     []string{"q"},
		Description: "Показывает текущую очередь треков",
		Usage:       "queue",
		Flags:       nil,
		IgnoreCase:  true,
		SubCommands: nil,
		RateLimiter: nil,
		Handler:     onQueue,
	})

	router.RegisterCmd(&dgc.Command{
		Name:        "pause",
		Aliases:     nil,
		Description: "Ставит паузу",
		Usage:       "pause",
		Flags:       nil,
		IgnoreCase:  true,
		SubCommands: nil,
		RateLimiter: nil,
		Handler:     onPause,
	})

	router.RegisterCmd(&dgc.Command{
		Name:        "resume",
		Aliases:     []string{"r"},
		Description: "Возобновляет проигрывание",
		Usage:       "resume",
		Flags:       nil,
		IgnoreCase:  true,
		SubCommands: nil,
		RateLimiter: nil,
		Handler:     onResume,
	})

	router.RegisterCmd(&dgc.Command{
		Name:        "skip",
		Aliases:     []string{"s"},
		Description: "Пропускает трек",
		Usage:       "skip",
		Flags:       nil,
		IgnoreCase:  true,
		SubCommands: nil,
		RateLimiter: nil,
		Handler:     onSkip,
	})

	router.RegisterCmd(&dgc.Command{
		Name:        "move",
		Aliases:     []string{"m"},
		Description: "Перемещает указанный трек на следующее место в очереди",
		Usage:       "move <позиция перемещаемого трека (в очереди)>",
		Flags:       nil,
		IgnoreCase:  true,
		SubCommands: nil,
		RateLimiter: nil,
		Handler:     onMove,
	})

	router.RegisterCmd(&dgc.Command{
		Name:        "connect",
		Aliases:     nil,
		Description: "Подключается к голосовому каналу",
		Usage:       "connect",
		Flags:       nil,
		IgnoreCase:  true,
		SubCommands: nil,
		RateLimiter: nil,
		Handler:     onConnect,
	})

	router.RegisterCmd(&dgc.Command{
		Name:        "leave",
		Aliases:     nil,
		Description: "Покидает канал",
		Usage:       "leave",
		Flags:       nil,
		IgnoreCase:  true,
		SubCommands: nil,
		RateLimiter: nil,
		Handler:     onLeave,
	})

	router.RegisterCmd(&dgc.Command{
		Name:        "shuffle",
		Description: "Перемешивает очередь",
		Usage:       "shuffle",
		Flags:       nil,
		IgnoreCase:  true,
		SubCommands: nil,
		RateLimiter: nil,
		Handler:     onShuffle,
	})

	router.RegisterCmd(&dgc.Command{
		Name:        "time",
		Description: "Показывает оставшееся время до конца текущего трека",
		Usage:       "time",
		Flags:       nil,
		IgnoreCase:  true,
		SubCommands: nil,
		RateLimiter: nil,
		Handler:     onTime,
	})
	router.Initialize(session)
}

func onReady(s *discordgo.Session, event *discordgo.Ready) {
	s.UpdateStatus(0, "yo")
}

func onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.Bot {
		return
	}

	if strings.HasPrefix(m.Content, "yo") {
		_, err := s.ChannelMessageSend(m.ChannelID, "yo")
		if err != nil {
			fmt.Println(err)
		}
	}

}

func Close() {
	CloseAllAudioSessions()
	discordSession.Close()
}