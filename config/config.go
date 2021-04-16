package config

import (
	"encoding/json"
	"github.com/kelseyhightower/envconfig"
	"os"
)

func init() {
	FillCfgFromFile("config.json")
	FillCfrFromEnv()
}

var Cfg = Config{}

type Config struct {
	DISCORDTOKEN string `json:"DISCORD_TOKEN" envconfig:"DISCORD_TOKEN"`
	VKTOKEN      string `json:"VK_TOKEN" envconfig:"VK_TOKEN"`
	FFMPEGPATH   string `json:"FFMPEG_PATH" envconfig:"FFMPEG_PATH"`
}

func FillCfgFromFile(file string) {
	f, err := os.Open(file)
	if err != nil {
		println(file + ": " + err.Error())
		return
	}
	defer f.Close()

	decoder := json.NewDecoder(f)
	err = decoder.Decode(&Cfg)
	if err != nil {
		println(file + ": " + err.Error())
		return
	}
}

func FillCfrFromEnv() {
	envconfig.Process("", &Cfg)
}
