package sfapi

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	log "github.com/Sirupsen/logrus"
	"github.com/alecthomas/units"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"
)

type Client struct {
	MVIP             string
	SVIP             string
	Login            string
	Password         string
	Endpoint         string
	DefaultAPIPort   int
	DefaultVolSize   int64
	DefaultAccountID int64
}

type Config struct {
	TenantName     string
	EndPoint       string
	DefaultVolSize int64 //Default volume size in GiB
	MountPoint     string
	SVIP           string
	InitiatorIFace string //iface to use of iSCSI initiator
	//Types       []map[string]QoS
}

func ProcessConfig(fname string) (Config, error) {
	content, err := ioutil.ReadFile(fname)
	if err != nil {
		log.Fatal("Error processing config file: ", err)
	}
	var conf Config
	err = json.Unmarshal(content, &conf)
	if err != nil {
		log.Fatal("Error parsing config file: ", err)
	}
	return conf, nil
}

func New() (c *Client, err error) {
	endpoint := os.Getenv("SF_ENDPOINT")
	svip := os.Getenv("SF_SVIP")
	defaultSize := os.Getenv("SF_DEFAULT_SIZE")
	if endpoint == "" || svip == "" {
		return &Client{}, nil
	}
	defSize, _ := units.ParseStrictBytes(defaultSize)
	return NewWithArgs(endpoint, svip, "docker", defSize)
}

func NewWithArgs(endpoint, svip, accountName string, defaultSize int64) (client *Client, err error) {
	rand.Seed(time.Now().UTC().UnixNano())
	client = &Client{
		Endpoint:       endpoint,
		DefaultVolSize: defaultSize,
		SVIP:           svip}
	return client, nil
}

func (c *Client) Request(method string, params interface{}, id int) (response []byte, err error) {
	if c.Endpoint == "" {
		log.Error("Endpoint is not set, unable to issue requests")
		err = errors.New("Unable to issue json-rpc requests without specifying Endpoint")
		return nil, err
	}
	data, err := json.Marshal(map[string]interface{}{
		"method": method,
		"id":     id,
		"params": params,
	})

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	Http := &http.Client{Transport: tr}
	resp, err := Http.Post(c.Endpoint,
		"json-rpc",
		strings.NewReader(string(data)))
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return body, err
	}

	var prettyJson bytes.Buffer
	_ = json.Indent(&prettyJson, body, "", "  ")
	log.WithField("", prettyJson.String()).Debug("request:", id, " method:", method, " params:", params)

	errresp := APIError{}
	json.Unmarshal([]byte(body), &errresp)
	if errresp.Error.Code != 0 {
		err = errors.New("Received error response from API request")
		return body, err
	}
	return body, nil
}

func newReqID() int {
	return rand.Intn(1000-1) + 1
}

func NewReqID() int {
	return rand.Intn(1000-1) + 1
}
