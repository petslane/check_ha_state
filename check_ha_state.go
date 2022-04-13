package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/mkideal/cli"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type HAState struct {
	EntityId    string `json:"entity_id"`
	State       string `json:"state"`
	LastChanged string `json:"last_changed"`
	LastUpdated string `json:"last_updated"`
}

type arguments struct {
	Help           bool   `cli:"h,help" usage:"show help"`
	Url            string `cli:"*url" usage:"HomeAssistant url. Example: http://127.0.0.1:8123"`
	EntityId       string `cli:"*e,entity" usage:"HomeAssistant entity id"`
	Token          string `cli:"*token" usage:"HomeAssistant API token"`
	LastUpdatedAge int    `cli:"u,last_updated_age" usage:"Maximum last updated age in seconds"`
	LastChangedAge int    `cli:"c,last_changed_age" usage:"Maximum last changed age in seconds"`
	Debug          bool   `cli:"debug" usage:"Show debug info"`
}

var token = ""
var maxAge int64 = 100
var debug = false

func requestState(haUrl string, entity string) (string, error) {
	url := haUrl
	if !strings.HasSuffix(url, "/") {
		url += "/"
	}
	url += "api/states/" + entity
	if debug {
		os.Stderr.WriteString("Requesting " + url + "\n")
	}

	req, err := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if debug {
		os.Stderr.WriteString("Response status: " + resp.Status + "\n")
		os.Stderr.WriteString("Response headers:\n")
		for key, val := range resp.Header {
			os.Stderr.WriteString("  " + key + ": " + strings.Join(val, "") + "\n")
		}
	}
	if resp.StatusCode != 200 {
		return "", errors.New("State not found: " + resp.Status)
	}

	body, _ := ioutil.ReadAll(resp.Body)
	if debug {
		os.Stderr.WriteString("Response: " + string(body) + "\n")
	}

	return string(body), nil
}

func getState(body string) HAState {
	var state HAState
	json.Unmarshal([]byte(body), &state)

	return state
}

func checkAge(dateTimeStr string, maxAge int) (int64, bool) {
	if maxAge == 0 {
		return 0, false
	}

	t, _ := time.Parse(time.RFC3339Nano, dateTimeStr)
	ago := time.Now().Unix() - t.Unix()

	return ago, ago > int64(maxAge)
}

func (argv *arguments) AutoHelp() bool {
	return argv.Help
}

func main() {
	os.Exit(cli.Run(new(arguments), func(ctx *cli.Context) error {
		argv := ctx.Argv().(*arguments)

		if argv.Debug {
			debug = true
		}
		token = argv.Token

		body, error := requestState(argv.Url, argv.EntityId)
		if error != nil {
			fmt.Println("CRITICAL - " + error.Error())
			os.Exit(2)
		}

		var state HAState = getState(body)

		if strings.ToUpper(state.State) == "UNKNOWN" {
			fmt.Println("CRITICAL - " + state.EntityId + " value UNKNONN")
			os.Exit(2)
		}
		if strings.ToUpper(state.State) == "UNAVAILABLE" {
			fmt.Println("CRITICAL - " + state.EntityId + " value UNAVAILABLE")
			os.Exit(2)
		}

		if ago, problem := checkAge(state.LastUpdated, argv.LastUpdatedAge); problem == true {
			fmt.Println("CRITICAL - " + state.EntityId + " last update too long ago (" + strconv.FormatInt(ago, 10) + "s > " + strconv.FormatInt(int64(argv.LastUpdatedAge), 10) + "s)")
			os.Exit(2)
		}

		if ago, problem := checkAge(state.LastChanged, argv.LastChangedAge); problem == true {
			fmt.Println("CRITICAL - " + state.EntityId + " last change too long ago (" + strconv.FormatInt(ago, 10) + "s > " + strconv.FormatInt(int64(argv.LastChangedAge), 10) + "s)")
			os.Exit(2)
		}

		fmt.Println("OK - " + state.EntityId + " | state=" + state.State + " last_updated=" + state.LastUpdated + " last_changed=" + state.LastChanged)
		os.Exit(0)

		return nil
	}))
}
