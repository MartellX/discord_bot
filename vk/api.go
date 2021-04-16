package vk

import (
	"MartellX/discord_bot/config"
	"encoding/json"
	"errors"
	"fmt"
	_ "github.com/Bogdan-D/go-socks4"
	"github.com/tidwall/gjson"
	"golang.org/x/net/proxy"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

var token = config.Cfg.VKTOKEN
var login, _ = os.LookupEnv("VK_LOGIN")
var passwd, _ = os.LookupEnv("VK_PASSWD")
var useragent = "KateMobileAndroid/56 lite-460 (Android 4.4.2; SDK 19; x86; unknown Android SDK built for x86; en)"

var timeout = time.Second * 20
var client http.Client = http.Client{
	Timeout: timeout,
}

var proxies []string
var proxyIndex = 0

var proxyChanging = sync.Mutex{}
var isProxyChanging = false

func init() {

	if login != "" && passwd != "" {
		//fmt.Println(login, passwd)
		tries := 0
		for {
			tries++
			fmt.Println("Tries getting tokens:", tries)
			tokenVk := getKateToken(login, passwd)
			if tokenVk != "" {
				token = tokenVk
				fmt.Println("Token successfully set")
				break
			}
			if tries > 10 {
				fmt.Println("Tries exceeded, using default token")
				break
			}
			time.Sleep(time.Second * 20)
		}
	}

	if token == "" {
		panic("VK token is not set!")
	}

	proxiesStr, ok := os.LookupEnv("PROXIES")
	if ok {
		if CheckCountry() == "RU" {
			return
		}
		proxies = strings.Split(proxiesStr, ";")
		rand.Shuffle(len(proxies), func(i, j int) {
			proxies[i], proxies[j] = proxies[j], proxies[i]
		})
		_, err := SwitchProxy()
		if err != nil {
			fmt.Println(err)
		}
	}
}

func SwitchProxy() (bool, error) {
	if isProxyChanging {
		return false, errors.New("already changing")
	}

	proxyChanging.Lock()
	isProxyChanging = true
	defer func() {
		isProxyChanging = false
		proxyChanging.Unlock()
	}()
	for i := 0; i < len(proxies); i++ {
		proxyIndex++
		proxyIndex %= len(proxies)
		if connectProxy(proxies[proxyIndex]) {
			return true, nil
		}
		client = http.Client{
			Timeout: timeout,
		}
		fmt.Println("Подключение через прокси не удалось")
	}
	return false, nil
}

func connectProxy(proxyURL string) bool {

	tbProxyURL, err := url.Parse(proxyURL)
	if err != nil {
		fmt.Printf("Failed to parse proxy URL: %v\n", err)
	}

	var tbTransport *http.Transport
	if strings.HasPrefix(tbProxyURL.Scheme, "socks") {
		tbDialer, err := proxy.FromURL(tbProxyURL, proxy.Direct)
		if err != nil {
			fmt.Printf("Failed to obtain proxy dialer: %v\n", err)
		}

		tbTransport = &http.Transport{Dial: tbDialer.Dial}
	} else {
		tbTransport = &http.Transport{Proxy: http.ProxyURL(tbProxyURL)}
	}

	client = http.Client{
		Transport: tbTransport,
		Timeout:   timeout,
	}
	fmt.Println("Проверяю прокси", tbProxyURL.String())
	req, _ := http.NewRequest(http.MethodGet, "https://api.myip.com/", nil)
	if resp, err := client.Do(req); err != nil {

		fmt.Println(err)
		return false
	} else {
		body, _ := ioutil.ReadAll(resp.Body)
		fmt.Println(string(body))

		// Checking vk
		CheckCountry()
		_, err = SearchAudio("Infected")
		if err != nil {
			fmt.Println(err)
			return false
		}

		fmt.Println("Подключение успешно")
		return true
	}
}

func CheckCountry() string {
	u, err := url.Parse("https://api.vk.com/method/account.getInfo")
	if err != nil {
		fmt.Println(err)
		return ""
	}

	query := u.Query()
	query.Add("access_token", token)
	query.Add("v", "5.126")
	u.RawQuery = query.Encode()
	request, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		fmt.Println(err)
		return ""
	}
	responseVk, err := client.Do(request)
	if err != nil {
		fmt.Println(err)
		return ""
	}
	body, _ := ioutil.ReadAll(responseVk.Body)
	country := gjson.GetBytes(body, "response.country").Str
	fmt.Println(country)
	return country
}

func getKateToken(login, password string) string {
	pythonexec := exec.Command("python", "gettingtoken.py", login, password)

	pythonexec.Stderr = os.Stderr
	pythonOut, err := pythonexec.Output()
	if err != nil {
		fmt.Println(err)
		return ""
	}
	fmt.Println(string(pythonOut))
	token := strings.TrimSpace(string(pythonOut))
	return token
}

type Track struct {
	Artist     string `json:"artist"`
	Id         int64  `json:"id"`
	Owner_id   int64  `json:"owner_id"`
	Title      string `json:"title"`
	Duration   int64  `json:"duration"`
	Access_key string `json:"access_key"`

	Url string `json:"url"`

	PlayedTime time.Duration
}

func (tr Track) String() string {

	result := tr.Artist + " - " + tr.Title

	if tr.Duration != 0 {
		result += " [" + tr.GetDuration().String() + "]"
	}
	return result
}

func (tr Track) GetDuration() time.Duration {
	dur, _ := time.ParseDuration(strconv.FormatInt(tr.Duration, 10) + "s")
	return dur
}

func SearchAudio(search string) (result []*Track, err error) {

	u, err := url.Parse("https://api.vk.com/method/audio.search")

	if err != nil {
		return nil, err
	}

	query := u.Query()
	query.Add("access_token", token)
	query.Add("count", "200")
	query.Add("q", search)
	query.Add("v", "5.126")
	u.RawQuery = query.Encode()

	request, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	//request.Header.Add("User-Agent", "KateMobileAndroid/69 lite-485 (Android 10; SDK 29; arm64-v8a; Xiaomi IN2013; ru)")
	request.Header.Add("User-Agent", useragent)
	fmt.Println("Sending request:", request.URL)
	resp, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	res := gjson.GetBytes(body, "response")
	if !res.Exists() {
		err := gjson.GetBytes(body, "error")
		return nil, errors.New(err.Get("error_msg").String())
	}
	count := res.Get("count").Int()

	items := make([]*Track, 0, count)
	for i, re := range res.Get("items").Array() {
		if i == 0 {
			fmt.Println(re.Raw)
		}
		track := Track{}
		json.Unmarshal([]byte(re.Raw), &track)
		if track.Url != "" {
			items = append(items, &track)
		}
	}

	return items, nil

}

func GetPlaylist(rawurl string, n int) (result []*Track, err error) {

	owner_id, album_id, access_key, err := getPlaylistParamsFromUrl(rawurl)

	if err != nil {
		return nil, err
	}

	// Getting tracks
	u, err := url.Parse("https://api.vk.com/method/audio.get")

	if err != nil {
		return nil, err
	}

	query := u.Query()
	query.Add("access_token", token)
	query.Add("owner_id", owner_id)
	query.Add("album_id", album_id)
	query.Add("access_key", access_key)
	query.Add("v", "5.126")

	if album_id == "" && access_key == "" {
		if n <= 0 {
			query.Add("count", "10")
		} else {
			query.Add("count", strconv.Itoa(n))
		}
	}
	u.RawQuery = query.Encode()

	request, err := http.NewRequest(http.MethodGet, u.String(), nil)

	if err != nil {
		return nil, err
	}

	//request.Header.Add("User-Agent", "KateMobileAndroid/69 lite-485 (Android 10; SDK 29; arm64-v8a; Xiaomi IN2013; ru)")
	request.Header.Add("User-Agent", useragent)
	fmt.Println("Sending request:", request.URL)
	resp, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	res := gjson.GetBytes(body, "response")

	count := res.Get("count").Int()

	items := make([]*Track, 0, count)
	for _, re := range res.Get("items").Array() {
		track := Track{}
		json.Unmarshal([]byte(re.Raw), &track)
		if track.Url != "" {
			items = append(items, &track)
		}
	}

	fmt.Println("Response:\n", string(body))

	return items, nil
}

func getPlaylistParamsFromUrl(rawurl string) (owner_id, album_id, access_key string, err error) {

	// Parsing playlist url
	u, err := url.Parse(rawurl)
	if err != nil {
		return "", "", "", err
	}

	query := u.Query().Get("z")
	if query == "" {
		path := u.Path
		if !strings.Contains(path, "/album/") && !strings.Contains(path, "/playlist/") && !strings.Contains(path, "/audios") {
			return "", "", "", errors.New("неизвестная ссылка")
		}

		paths := strings.Split(path, "/")
		if strings.Contains(path, "/audios") {
			return strings.Replace(paths[len(paths)-1], "audios", "", 1), "", "", nil
		}
		params := strings.Split(paths[len(paths)-1], "_")

		if len(params) < 2 {
			return "", "", "", errors.New("не удалось считать ссылку")
		}
		owner_id, album_id = params[0], params[1]
		if len(params) > 2 {
			access_key = params[2]
		}
	} else {
		if !strings.Contains(query, "audio_playlist") {
			return "", "", "", errors.New("неизвестная ссылка")
		}
		query = strings.Replace(query, "audio_playlist", "", 1)
		query = strings.Replace(query, "/", "_", 1)
		params := strings.Split(query, "_")

		if len(params) < 2 {
			return "", "", "", errors.New("не удалось считать ссылку")
		}
		owner_id, album_id = params[0], params[1]
		if len(params) > 2 {
			access_key = params[2]
		}
	}

	return owner_id, album_id, access_key, nil
}
