package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/mkideal/cli"
	"gopkg.in/yaml.v2"
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
	Config         string `cli:"config" usage:"YAML configuration file containing \"url\" and \"token\" properties"`
	Url            string `cli:"url" usage:"HomeAssistant url. Example: http://127.0.0.1:8123"`
	Token          string `cli:"token" usage:"HomeAssistant API token"`
	EntityId       string `cli:"*e,entity" usage:"HomeAssistant entity id"`
	LastUpdatedAge int    `cli:"u,last_updated_age" usage:"Maximum last updated age in seconds"`
	LastChangedAge int    `cli:"c,last_changed_age" usage:"Maximum last changed age in seconds"`
	Debug          bool   `cli:"debug" usage:"Show debug info"`
}

type conf struct {
	Url   string `yaml:"url"`
	Token string `yaml:"token"`
}

var debug = false

func (c *conf) getConf(filename string) (*conf, error) {
	if debug {
		os.Stderr.WriteString("Reading config file \"" + filename + "\"...\n")
	}
	yamlFile, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		return nil, errors.New(filename + " not a yaml file")
	}

	return c, nil
}

func requestState(haUrl string, token string, entity string) (string, error) {
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

func (argv *arguments) Validate(ctx *cli.Context) error {
	hasUrl := len(argv.Url) > 0
	hasToken := len(argv.Token) > 0
	hasConfig := len(argv.Config) > 0
	if !hasUrl && !hasToken && !hasConfig {
		nagiosCritical("Missing requred arguments --config or --url and --token!")
	}
	if (hasUrl || hasToken) && hasConfig {
		nagiosCritical("Remove --url and --token arguments if --config argument is used!")
	}
	if (!hasUrl || !hasToken) && !hasConfig {
		nagiosCritical("Both --url and --token arguments are required!")
	}

	return nil
}

func getUrlAndToken(argv *arguments) (string, string) {
	url := argv.Url
	token := argv.Token

	if len(argv.Config) > 0 {
		var c conf
		conf, err := c.getConf(argv.Config)
		if err != nil {
			nagiosCritical(err.Error())
		}
		if len(conf.Url) == 0 {
			nagiosCritical("config file must contain \"url\" property")
		}
		if len(conf.Token) == 0 {
			nagiosCritical("config file must contain \"token\" property")
		}
		url = conf.Url
		token = conf.Token
	}

	return url, token
}

func main() {
	os.Exit(cli.Run(new(arguments), func(ctx *cli.Context) error {
		argv := ctx.Argv().(*arguments)

		if argv.Debug {
			debug = true
		}

		url, token := getUrlAndToken(argv)

		body, error := requestState(url, token, argv.EntityId)
		if error != nil {
			nagiosCritical(error.Error())
		}

		var state HAState = getState(body)

		if strings.ToUpper(state.State) == "UNKNOWN" {
			nagiosCritical(state.EntityId + " value UNKNONN")
		}
		if strings.ToUpper(state.State) == "UNAVAILABLE" {
			nagiosCritical(state.EntityId + " value UNAVAILABLE")
		}

		if ago, problem := checkAge(state.LastUpdated, argv.LastUpdatedAge); problem == true {
			nagiosCritical(state.EntityId + " last update too long ago (" + strconv.FormatInt(ago, 10) + "s > " + strconv.FormatInt(int64(argv.LastUpdatedAge), 10) + "s)")
		}

		if ago, problem := checkAge(state.LastChanged, argv.LastChangedAge); problem == true {
			nagiosCritical(state.EntityId + " last change too long ago (" + strconv.FormatInt(ago, 10) + "s > " + strconv.FormatInt(int64(argv.LastChangedAge), 10) + "s)")
		}

		nagiosOK(state.EntityId + " | state=" + state.State + " last_updated=" + state.LastUpdated + " last_changed=" + state.LastChanged)

		return nil
	}))
}

func nagiosCritical(message string) {
	fmt.Println("CRITICAL - " + message)
	os.Exit(2)
}

func nagiosOK(message string) {
	fmt.Println("OK - " + message)
	os.Exit(0)
}
