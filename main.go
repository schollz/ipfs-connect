package main

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
	"time"

	log "github.com/schollz/logger"
)

func main() {
	var flagDebug bool
	var flagConnector string
	flag.BoolVar(&flagDebug, "debug", false, "debug mode")
	flag.StringVar(&flagConnector, "connect", "", "connector to listen")
	flag.Parse()

	if flagDebug {
		log.SetLevel("debug")
	} else {
		log.SetLevel("info")
	}

	err := run(flagConnector)
	if err != nil {
		log.Error(err)
	}

}

type Message struct {
	ID        string
	Addresses []string
}

func run(connector string) (err error) {
	token := make([]byte, 16)
	rand.Read(token)
	id := fmt.Sprintf("%x", token)
	if connector == "" {
		token = make([]byte, 16)
		rand.Read(token)
		connector = fmt.Sprintf("%x", token)
	}
	fmt.Printf("id: %s\n", id)
	fmt.Printf("connector: %s\n", connector)

	go listenForAddresses(id, connector)
	go func() {
		time.Sleep(1 * time.Second)
		b, _ := json.Marshal(Message{
			ID: id,
		})
		err = postData(connector, b)
		if err != nil {
			panic(err)
		}
	}()

	for {
		var resp *http.Response
		resp, err = http.Get("https://duct.schollz.com/" + id)
		if err != nil {
			return
		}
		var b []byte
		b, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return
		}
		resp.Body.Close()

		var msg Message
		err = json.Unmarshal(b, &msg)
		if err != nil {
			log.Error(err)
			continue
		}
		log.Debugf("got addresses: %+v", msg)

	}
	return
}

func listenForAddresses(id, connector string) (err error) {
	for {
		var resp *http.Response
		resp, err = http.Get("https://duct.schollz.com/" + connector)
		if err != nil {
			return
		}
		var b []byte
		b, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return
		}
		resp.Body.Close()

		var msg Message
		err = json.Unmarshal(b, &msg)
		if err != nil {
			log.Error(err)
			continue
		}
		log.Debugf("got msg: %+v", msg)
		if msg.ID != "" && msg.ID != id {
			err = sendAddresses(msg.ID)
			if err != nil {
				panic(err)
			}
			requestAddresses(id, connector)
		}
	}
}

func requestAddresses(id, connector string) (err error) {
	b, _ := json.Marshal(Message{
		ID: id,
	})
	err = postData(connector, b)
	if err != nil {
		panic(err)
	}
	return
}
func listenForNeeds(connector string) (err error) {
	for {
		var resp *http.Response
		resp, err = http.Get("https://duct.schollz.com/need" + connector)
		if err != nil {
			return
		}
		var b []byte
		b, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return
		}
		resp.Body.Close()

		var addresses []string
		err = json.Unmarshal(b, &addresses)
		if err != nil {
			log.Error(err)
			continue
		}
		log.Debugf("got addresses: %+v", addresses)

	}
}

func sendAddresses(id string) (err error) {
	addresses, err := getAddresses()
	if err != nil {
		return
	}
	b, err := json.Marshal(Message{
		Addresses: addresses,
	})
	if err != nil {
		return
	}
	err = postData(id, b)
	return
}

func postData(connector string, data []byte) (err error) {
	body := bytes.NewReader(data)
	req, err := http.NewRequest("POST", "https://duct.schollz.com/"+connector+"?pubsub=true", body)
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	return
}

func getAddresses() (addresses []string, err error) {
	cmd := exec.Command("ipfs", "id", "--encoding", "json")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return
	}
	type IDData struct {
		ID        string
		Addresses []string
	}
	var iddata IDData
	err = json.Unmarshal(out, &iddata)
	if err != nil {
		return
	}
	addresses = iddata.Addresses
	return
}
