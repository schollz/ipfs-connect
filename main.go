package main

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/denisbrodbeck/machineid"
	log "github.com/schollz/logger"
)

func main() {
	connector := ""
	if len(os.Args) > 1 {
		connector = os.Args[1]
	}
	err := run(connector)
	if err != nil {
		log.Error(err)
	}
}

type Message struct {
	ID        string
	Addresses []string
}

func run(connector string) (err error) {
	id, iderr := machineid.ID()
	token := make([]byte, 16)
	rand.Read(token)
	if iderr != nil {
		id = fmt.Sprintf("%x", token)
	}
	if connector == "" {
		token = make([]byte, 16)
		rand.Read(token)
		connector = fmt.Sprintf("%x", token)
	}
	// fmt.Printf("your id: %s\n", id)
	fmt.Printf("add another computer to your swarm by running:\n\nipfs-connect %s\n\n", connector)

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
			err = sendAddresses(msg.ID, msg.ID)
			if err != nil {
				log.Error(err)
				return
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
	if len(addresses) == 0 {
		err = fmt.Errorf("no addresses")
		return
	}
	foo := strings.Split(addresses[0], "/")
	fmt.Printf("establishing connection to %s...", foo[len(foo)-1])
	var wg sync.WaitGroup
	wg.Add(len(addresses))
	didConnect := false
	for _, addr := range addresses {
		go func() {
			connected := connectToAddress(addr)
			if connected {
				didConnect = true
			}
			wg.Done()
		}()
	}
	wg.Wait()

	if didConnect {
		fmt.Printf("success!\n")
	} else {
		fmt.Printf("failed.\n")
	}
	return
}

func connectToAddress(addr string) (connected bool) {
	cmd := exec.Command("ipfs", "swarm", "connect", addr, "--encoding", "json")
	out, err := cmd.CombinedOutput()
	if err != nil {
		connected = false
	} else {
		connected = bytes.Contains(out, []byte("success"))
	}
	return
}
