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
		sendAddresses(id, connector)
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
		go connectToAddresses(msg.Addresses)

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

		if msg.ID != "" && msg.ID != id {
			go connectToAddresses(msg.Addresses)
			log.Debugf("got msg: %+v", msg)
			err = sendAddresses(msg.ID, msg.ID)
			if err != nil {
				panic(err)
			}
		}
	}
}

func sendAddresses(id, sendto string) (err error) {
	addresses, err := getAddresses()
	if err != nil {
		return
	}
	b, err := json.Marshal(Message{
		ID:        id,
		Addresses: addresses,
	})
	if err != nil {
		return
	}
	err = postData(sendto, b)
	return
}

func postData(sendto string, data []byte) (err error) {
	body := bytes.NewReader(data)
	req, err := http.NewRequest("POST", "https://duct.schollz.com/"+sendto+"?pubsub=true", body)
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

func connectToAddresses(addresses []string) (err error) {
	for _, addr := range addresses {
		go connectToAddress(addr)
	}
	return
}

func connectToAddress(addr string) {
	log.Debugf("connecting to %s", addr)
	cmd := exec.Command("ipfs", "swarm", "connect", addr, "--encoding", "json")
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Debugf("err: %s", err.Error())
	}
	log.Debugf("ipfs swarm connect: %s", out)
}
