package motu

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

type Datastore interface {
	Fetch(key string) error

	GetInt(key string) (int64, error)
	GetFloat(key string) (float64, error)
	GetStr(key string) (string, error)

	SetInt(key string, value int64) error
	SetFloat(key string, value float64) error
	SetStr(key, value string) error
}

func getAtPath(thisData map[string]any, path string) (any, error) {
	thisEntry, remaining, ok := strings.Cut(path, "/")
	if !ok {
		return thisData[thisEntry], nil
	}
	nextData, ok := thisData[thisEntry]
	if !ok {
		// TODO: this message can be better
		return nil, fmt.Errorf("Path element %s in path %s not found", thisEntry, path)
	}
	switch nextData := nextData.(type) {
	case map[string]any:
		return getAtPath(nextData, remaining)
	default:
		return nil, fmt.Errorf("Path element %s in path %s has no children", thisEntry, path)
	}
}

func setAtPath(thisData map[string]any, path string, newData any) error {
	thisEntry, remaining, ok := strings.Cut(path, "/")
	if !ok {
		thisData[thisEntry] = newData
		return nil
	}
	// TODO: use reflect to type check here
	nextData, ok := thisData[thisEntry]
	if !ok {
		// TODO: this message can be better
		return fmt.Errorf("Path element %s in path %s not found", thisEntry, path)
	}
	switch nextData := nextData.(type) {
	case map[string]any:
		return setAtPath(nextData, remaining, newData)
	default:
		return fmt.Errorf("Path element %s in path %s has no children", thisEntry, path)
	}
}

type FileDatastore struct {
	File  io.Reader
	cache map[string]any
}

func (d *FileDatastore) Fetch(key string) error {
	var data map[string]any
	json.NewDecoder(d.File).Decode(data)
	for _, entry := range strings.SplitN(key, "/", -1) {
		newData, ok := data[entry]
		if !ok {
			return fmt.Errorf("Could not find %s in %s", entry, key)
		}
		return setAtPath(d.cache, key, newData)
	}
	return nil
}

func (d *FileDatastore) GetInt(key string) (int64, error) {
	val, err := getAtPath(d.cache, key)
	if err != nil {
		return 0, err
	}
	cast, ok := val.(int64)
	if !ok {
		return 0, fmt.Errorf("Type of %s is not int", key)
	}
	return cast, nil
}

func (d *FileDatastore) GetFloat(key string) (float64, error) {
	val, err := getAtPath(d.cache, key)
	if err != nil {
		return 0, err
	}
	cast, ok := val.(float64)
	if !ok {
		return 0, fmt.Errorf("Type of %s is not float", key)
	}
	return cast, nil
}

func (d *FileDatastore) GetStr(key string) (string, error) {
	val, err := getAtPath(d.cache, key)
	if err != nil {
		return "", err
	}
	cast, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("Type of %s is not float", key)
	}
	return cast, nil
}

func (d *FileDatastore) SetInt(key string, val int64) error {
	return setAtPath(d.cache, key, val)
}

func (d *FileDatastore) SetFloat(key string, val float64) error {
	return setAtPath(d.cache, key, val)
}

func (d *FileDatastore) SetStr(key string, val string) error {
	return setAtPath(d.cache, key, val)
}

type HTTPDatastore struct {
	Client http.Client
	url    string
	cache  map[string]any
}

func NewHTTPDatastore(url string) HTTPDatastore {
	return HTTPDatastore{
		Client: http.Client{},
		url:    url,
		cache:  make(map[string]any),
	}
}

func (d *HTTPDatastore) poll() {
	etag := 0
	for {
		req, err := http.NewRequest(http.MethodGet, d.url, nil)
		req.Header.Set("If-None-Match", fmt.Sprintf("%d", etag))
		if err != nil {
			panic(err)
		}
		resp, err := d.Client.Do(req)
		if err != nil {
			panic(err)
		}
		if resp.StatusCode != 304 {
			defer resp.Body.Close()
			if err != nil {
				panic(err)
			}
			etag, err = strconv.Atoi(resp.Header.Get("ETag"))
			if err != nil {
				panic(err)
			}
			if err := json.NewDecoder(resp.Body).Decode(&d.cache); err != nil {
				panic(err)
			}
		}
	}
}

// TODO: do this with proper struct types
func (d *HTTPDatastore) GetInt(key string) (int64, error) {
	val, err := getAtPath(d.cache, key)
	if err != nil {
		return 0, err
	}
	cast, ok := val.(int64)
	if !ok {
		return 0, fmt.Errorf("Type of %s is not int", key)
	}
	return cast, nil
}

func (d *HTTPDatastore) GetFloat(key string) (float64, error) {
	val, err := getAtPath(d.cache, key)
	if err != nil {
		return 0, err
	}
	cast, ok := val.(float64)
	if !ok {
		return 0, fmt.Errorf("Type of %s is not float", key)
	}
	return cast, nil
}

func (d *HTTPDatastore) GetStr(key string) (string, error) {
	val, err := getAtPath(d.cache, key)
	if err != nil {
		return "", err
	}
	cast, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("Type of %s is not string", key)
	}
	return cast, nil
}

func (d *HTTPDatastore) SetInt(key string, value int64) error {
	jsonData := []byte(fmt.Sprintf(`json={"%s":"%d"}`, key, value))

	req, err := http.NewRequest(http.MethodPatch, d.url, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Accept-Encoding", "*/*")
	if err != nil {
		return err
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		panic(resp.StatusCode)
	}

	return nil
}
