package vk

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/tidwall/gjson"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

var token, _ = os.LookupEnv("VK_TOKEN")
var login, _ = os.LookupEnv("VK_LOGIN")
var passwd, _ = os.LookupEnv("VK_PASSWD")
var useragent = "KateMobileAndroid/56 lite-460 (Android 4.4.2; SDK 19; x86; unknown Android SDK built for x86; en)"

func init() {
	if login != "" && passwd != "" {
		tokenVk := getKateToken(login, passwd)
		if tokenVk != "" {
			token = tokenVk
			fmt.Println("Token successfully set")
		}
	}
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

var client http.Client = http.Client{}

//func init() {
//	tbProxyURL, err := url.Parse("socks5://127.0.0.1:9050")
//	if err != nil {
//		fmt.Printf("Failed to parse proxy URL: %v\n", err)
//	}
//
//	// Get a proxy Dialer that will create the connection on our
//	// behalf via the SOCKS5 proxy.  Specify the authentication
//	// and re-create the dialer/transport/client if tor's
//	// IsolateSOCKSAuth is needed.
//	tbDialer, err := proxy.FromURL(tbProxyURL, proxy.Direct)
//	if err != nil {
//		fmt.Printf("Failed to obtain proxy dialer: %v\n", err)
//	}
//
//	// Make a http.Transport that uses the proxy dialer, and a
//	// http.Client that uses the transport.
//	tbTransport := &http.Transport{Dial: tbDialer.Dial}
//	client = http.Client{Transport: tbTransport}
//
//}

func GetAudioById(id string) {

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

		items = append(items, &track)
	}

	return items, nil

}

func GetPlaylist(rawurl string) (result []*Track, err error) {

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

		items = append(items, &track)
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
		if !strings.Contains(path, "/album/") {
			return "", "", "", errors.New("неизвестная ссылка")
		}

		paths := strings.Split(path, "/")

		params := strings.Split(paths[len(paths)-1], "_")

		if len(params) != 3 {
			return "", "", "", errors.New("не удалось считать ссылку")
		}
		owner_id, album_id, access_key = params[0], params[1], params[2]
	} else {
		if !strings.Contains(query, "audio_playlist") {
			return "", "", "", errors.New("неизвестная ссылка")
		}
		query = strings.Replace(query, "audio_playlist", "", 1)
		query = strings.Replace(query, "/", "_", 1)
		params := strings.Split(query, "_")

		if len(params) != 3 {
			return "", "", "", errors.New("не удалось считать ссылку")
		}
		owner_id, album_id, access_key = params[0], params[1], params[2]
	}

	return owner_id, album_id, access_key, nil
}
