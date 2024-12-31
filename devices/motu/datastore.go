package motu

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

type allowedTypes interface {
	int64 | float64 | string | bool
}

type (
	Effect[T allowedTypes] func(T) error
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

type HTTPDatastore struct {
	Client http.Client
	url    string
	cache  map[string]any

	actionInt   map[string]Effect[int64]
	actionFloat map[string]Effect[float64]
	actionStr   map[string]Effect[string]
	actionBool  map[string]Effect[bool]
}

func NewHTTPDatastore(url string) HTTPDatastore {
	return HTTPDatastore{
		Client:      http.Client{},
		url:         url,
		cache:       make(map[string]any),
		actionInt:   map[string]Effect[int64]{},
		actionFloat: map[string]Effect[float64]{},
		actionStr:   map[string]Effect[string]{},
		actionBool:  map[string]Effect[bool]{},
	}
}

func (d *HTTPDatastore) RegisterInt(key string, e Effect[int64]) {
	d.actionInt[key] = e
}

func (d *HTTPDatastore) RegisterFloat(key string, e Effect[float64]) {
	d.actionFloat[key] = e
}

func (d *HTTPDatastore) RegisterStr(key string, e Effect[string]) {
	d.actionStr[key] = e
}

func handleEffect[T allowedTypes](effects map[string]Effect[T], oldData, newData map[string]any) error {
	for k, e := range effects {
		if new, inNew := newData[k]; inNew {
			old, inOld := oldData[k]
			if !inOld {
				// TODO: this could panic; handle gracefully
				if err := e(new.(T)); err != nil {
					// TODO: handle this gracefully
					panic(err)
				}
			}
			if old != new {
				if err := e(new.(T)); err != nil {
					panic(err)
				}
			}
		}
	}
	return nil
}

// TODO: set up some kind of polling hooks for changed values
func (d *HTTPDatastore) poll() {
	etag := 0
	newData := map[string]any{}
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
			if err := json.NewDecoder(resp.Body).Decode(&newData); err != nil {
				panic(err)
			}
			if err := handleEffect[int64](d.actionInt, d.cache, newData); err != nil {
				panic(err)
			}
			// Check for changes in any entries that have an effect registered
			// Don't bother checking the rest
			for k, e := range d.actionInt {
				if new, inNew := newData[k]; inNew {
					old, inOld := d.cache[k]
					if !inOld {
						// TODO: this could panic; handle gracefully
						if err := e(new.(int64)); err != nil {
							// TODO: handle this gracefully
							panic(err)
						}
					}
					if old != new {
						if err := e(new.(int64)); err != nil {
							panic(err)
						}
					}
				}
			}
			for k, e := range d.actionFloat {
				if new, inNew := newData[k]; inNew {
					old, inOld := d.cache[k]
					if !inOld {
						// TODO: this could panic; handle gracefully
						if err := e(new.(float64)); err != nil {
							// TODO: handle this gracefully
							panic(err)
						}
					}
					if old != new {
						if err := e(new.(float64)); err != nil {
							panic(err)
						}
					}
				}
			}
			for k, e := range d.actionStr {
				if new, inNew := newData[k]; inNew {
					old, inOld := d.cache[k]
					if !inOld {
						// TODO: this could panic; handle gracefully
						if err := e(new.(string)); err != nil {
							// TODO: handle this gracefully
							panic(err)
						}
					}
					if old != new {
						if err := e(new.(string)); err != nil {
							panic(err)
						}
					}
				}
			}
			for k, e := range d.actionBool {
				if new, inNew := newData[k]; inNew {
					old, inOld := d.cache[k]
					if !inOld {
						// TODO: this could panic; handle gracefully
						if err := e(new.(bool)); err != nil {
							// TODO: handle this gracefully
							panic(err)
						}
					}
					if old != new {
						if err := e(new.(bool)); err != nil {
							panic(err)
						}
					}
				}
			}
			d.cache = newData
		}
	}
}

func (d *HTTPDatastore) GetInt(key string) (int64, error) {
	val, ok := d.cache[key]
	if !ok {
		return 0, fmt.Errorf("Could not find %s", key)
	}
	switch val := val.(type) {
	case int64:
		return int64(val), nil
	case int:
		return int64(val), nil
	case float32:
		return int64(val), nil
	case float64:
		return int64(val), nil
	case string:
		cast, err := strconv.Atoi(val)
		if err != nil {
			return 0, fmt.Errorf("Cannot cast %s to int", key)
		}
		return int64(cast), nil
	}
	panic(fmt.Sprintf("Unsupported type %T", val))
}

func (d *HTTPDatastore) GetFloat(key string) (float64, error) {
	val, ok := d.cache[key]
	if !ok {
		return 0, fmt.Errorf("Could not find %s", key)
	}
	switch val := val.(type) {
	case int64:
		return float64(val), nil
	case int:
		return float64(val), nil
	case float32:
		return float64(val), nil
	case float64:
		return float64(val), nil
	case string:
		cast, err := strconv.Atoi(val)
		if err != nil {
			return 0, fmt.Errorf("Cannot cast %s to int", key)
		}
		return float64(cast), nil
	}
	panic(fmt.Sprintf("Unsupported type %T", val))
}

func (d *HTTPDatastore) GetStr(key string) (string, error) {
	val, ok := d.cache[key]
	cast, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("Type of %s is not string", key)
	}
	return cast, nil
}

func (d *HTTPDatastore) GetBool(key string) (bool, error) {
	val, ok := d.cache[key]
	cast, ok := val.(bool)
	if !ok {
		return false, fmt.Errorf("Type of %s is not string", key)
	}
	return cast, nil
}

func (d *HTTPDatastore) SetInt(key string, value int64) error {
	d.cache[key] = value

	// if effect, ok := d.actionInt[key]; ok {
	// 	if err := effect(value); err != nil {
	// 		// TODO: log this instead of returning
	// 		return err
	// 	}
	// }

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

func (d *HTTPDatastore) SetFloat(key string, value float64) error {
	d.cache[key] = value

	// if effect, ok := d.actionFloat[key]; ok {
	// 	if err := effect(value); err != nil {
	// 		// TODO: log this instead of returning
	// 		return err
	// 	}
	// }

	jsonData := []byte(fmt.Sprintf(`json={"%s":"%f"}`, key, value))

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

func (d *HTTPDatastore) SetString(key string, value string) error {
	d.cache[key] = value

	// if effect, ok := d.actionStr[key]; ok {
	// 	if err := effect(value); err != nil {
	// 		// TODO: log this instead of returning
	// 		return err
	// 	}
	// }

	jsonData := []byte(fmt.Sprintf(`json={"%s":"%s"}`, key, value))

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

func (d *HTTPDatastore) SetBool(key string, value bool) error {
	d.cache[key] = value

	// if effect, ok := d.actionBool[key]; ok {
	// 	if err := effect(value); err != nil {
	// 		// TODO: log this instead of returning
	// 		return err
	// 	}
	// }

	jsonData := []byte(fmt.Sprintf(`json={"%s":"%s"}`, key, value))

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
