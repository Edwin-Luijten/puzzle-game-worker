package main

import (
	"encoding/json"
	"net/http"
	_ "expvar"
	"time"
	"flag"
	"os/exec"
	"os"
	"io"
	"strings"
	"net/url"
	"path"
	"io/ioutil"
	"bytes"
	"encoding/base64"
	"errors"
	"strconv"
	log "github.com/sirupsen/logrus"
)

const userAgent = "Game Worker"
const authToken = "eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJpZCI6InNvbHV0aW9uLXF1ZXVlLXdvcmtlciIsImlhdCI6MTUwMjk3NzA4Nn0.gT2Y-xEiDgk0Q1bGSI6ds-234pCkTsLcfVZOI_85Uhk"
const commandEndpointQueryKey = "command"
const jobsEndpointQueryValue = "pop-queued-solution"
const jobsResultEndpointQueryValue = "submit-result"

var dockerImages = map[string]string{
	"php": "php:cli",
}

type Job struct {
	Key              string `json:"key"`
	ChallengePayload string `json:"challengePayload"`
	Script           string `json:"script"`
	Language         string `json:"language"`
}

func main() {
	apiEndpoint := flag.String("endpoint", "http://localhost:8080/", "Queue endpoint url.")
	queueWait := flag.Int("wait", 0, "Seconds to wait before checking for new job. Default 0.")
	logFile := flag.String("log", "worker.log", "File path to log to. Default ./worker.log.")
	debug := flag.Bool("debug", true, "Log level. Default true")

	flag.Parse()

	log.SetOutput(os.Stdout)
	log.SetFormatter(&log.JSONFormatter{})

	if *debug {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.WarnLevel)
	}

	file, err := os.OpenFile(*logFile, os.O_CREATE|os.O_WRONLY, 0666)

	if err == nil {
		log.SetOutput(file)
	} else {
		log.Info("Failed to log to file, using default stderr")
	}

	for {
		//go func() {
		//	for {
		//		var m runtime.MemStats
		//		runtime.ReadMemStats(&m)
		//		log.Printf("\nAlloc = %v\nTotalAlloc = %v\nSys = %v\nNumGC = %v\n\n", m.Alloc / 1024, m.TotalAlloc / 1024, m.Sys / 1024, m.NumGC)
		//		time.Sleep(5 * time.Second)
		//	}
		//}()

		job, err := fetchJob(*apiEndpoint)

		if err != nil {
			log.Error(err.Error())
		} else {
			processJob(job, *apiEndpoint)
		}

		if *queueWait > 0 {
			time.Sleep(time.Duration(*queueWait) * time.Second)
		}
	}

}

// fetchJob checks against the given endpoint if there is a new job to process
func fetchJob(apiEndpoint string) (Job, error) {
	job := Job{}

	client := http.Client{}

	request, err := http.NewRequest(http.MethodGet, apiEndpoint, nil)

	query := request.URL.Query()
	query.Add(commandEndpointQueryKey, jobsEndpointQueryValue)

	request.URL.RawQuery = query.Encode()

	if err != nil {
		log.Error(err)
	}

	request.Header.Set("User-Agent", userAgent)
	request.Header.Set("Authorization", authToken)

	response, err := client.Do(request)
	defer response.Body.Close()

	if err != nil {
		log.Fatal(err)

		return job, errors.New("Request error: " + err.Error())
	}

	if response.StatusCode != 200 {
		return job, errors.New("Status code other other than 200. Got: " + strconv.Itoa(response.StatusCode))
	}

	body, err := ioutil.ReadAll(response.Body)

	if err != nil {
		log.Error(err)

		return job, errors.New("IO error: " + err.Error())
	}

	err = json.Unmarshal(body, &job)

	if err != nil {
		log.Error(err)

		return job, errors.New("Json error: " + err.Error())
	}

	return job, nil
}

// processJob Runs the job trough the docker flow
func processJob(job Job, apiEndpoint string) {
	processResult(apiEndpoint, job.Key, runDocker(job, apiEndpoint))

	os.Remove(job.Key)
}

func processResult(apiEndpoint string, key string, output string) {
	data := url.Values{}
	data.Add("key", key)
	data.Add("resultPayload", output)
	//data.Add("statistics", "")

	client := http.Client{}

	request, err := http.NewRequest(http.MethodPost, apiEndpoint, strings.NewReader(data.Encode()))

	query := request.URL.Query()
	query.Add(commandEndpointQueryKey, jobsResultEndpointQueryValue)

	request.URL.RawQuery = query.Encode()

	if err != nil {
		log.Error(err)
	}

	request.Header.Set("User-Agent", userAgent)
	request.Header.Set("Authorization", authToken)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response, err := client.Do(request)
	defer response.Body.Close()

	if err != nil {
		log.Error(err)
	}

	if response.StatusCode != 200 {
		log.Info("Status code other other than 200. Got: " + strconv.Itoa(response.StatusCode))
	}
}

// runDocker uses the downloaded script and provided docker image to get the solution payload
// once done we send the output back to the game server
func runDocker(job Job, apiEndpoint string) (string) {
	script := fetchScript(job.Script, apiEndpoint)

	mapLocal := "$(pwd)"
	mapRemote := "/app"
	dockerImage := getDockerImage(job.Language)

	log.Debug("Running docker with image: " + dockerImage)

	// Run docker container
	scriptPayload, err := base64.StdEncoding.DecodeString(job.ChallengePayload)

	if err != nil {
		log.Error("Payload error: " + err.Error())
	}

	command := "docker run --rm -v " + mapLocal + ":" + mapRemote + " -w " + mapRemote + " " + dockerImage + " " + job.Language + " " + script + " " + string(scriptPayload)
	cmd := exec.Command("/bin/sh", "-c", command)

	var output bytes.Buffer

	cmd.Stdout = &output
	err = cmd.Run()

	payload := ""

	if err != nil {
		log.Debug("Script error: " + err.Error())

		payload = err.Error()
	} else {
		log.Debug("Script output: " + output.String())

		payload = output.String()
	}

	return base64.StdEncoding.EncodeToString([]byte(payload))
}

func getDockerStats() {

}

// fetchScript downloads the url from the given url and stores this on disk
// this file is given as a parameter in docker
func fetchScript(script string, apiEndpoint string) (string) {
	ScriptAddress, err := url.Parse(apiEndpoint)
	ScriptAddress.Path = path.Join(script)

	tokens := strings.Split(ScriptAddress.String(), "/")
	fileName := tokens[len(tokens)-1]

	log.Debug("Downloading ", ScriptAddress.String(), "to", fileName)

	output, err := os.Create(fileName)
	defer output.Close()

	if err != nil {
		log.Error("Error while creating", fileName, "-", err)
		return ""
	}

	request, err := http.NewRequest(http.MethodGet, ScriptAddress.String(), nil)
	request.Header.Set("User-Agent", userAgent)
	request.Header.Set("Authorization", authToken)

	client := http.Client{}

	response, err := client.Do(request)
	defer response.Body.Close()

	if err != nil {
		log.Error("Error while downloading ", ScriptAddress.String(), "-", err)
		return ""
	}

	n, err := io.Copy(output, response.Body)
	if err != nil {
		log.Error("Error while downloading", ScriptAddress.String(), "-", err)
		return ""
	}

	log.Debug("Downloaded ", ScriptAddress.String(), "- wrote", n, "bytes")

	scriptPath := path.Join(path.Dir(fileName), fileName)

	return scriptPath
}

// getDockerImage matches the language to a docker image
func getDockerImage(language string) (string) {
	if image, ok := dockerImages[language]; ok {
		return image
	}

	return ""
}
