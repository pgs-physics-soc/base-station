package main

import (
	"log"
	"net/http"
	"strings"
	"sync"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gorilla/websocket"
)

type BinaryPressureSensor struct {
	name      string
	data      chan []byte
	ctrl      chan []byte
	ws        []*websocket.Conn
	currState []byte
}

var allBinaryPressureSensors map[string]*BinaryPressureSensor
var allBinaryPressureSensorsMut sync.RWMutex

func binaryPressureSensorRunner(name string) {
	bps := BinaryPressureSensor{
		name:      name,
		data:      make(chan []byte, 64), // sensible capacity defaults just to skip reallocs
		ctrl:      make(chan []byte, 64),
		ws:        make([]*websocket.Conn, 0, 8),
		currState: []byte("0"),
	}
	allBinaryPressureSensorsMut.Lock()
	allBinaryPressureSensors[name] = &bps
	allBinaryPressureSensorsMut.Unlock()

	token := client.Subscribe(name+"/data", 1, binaryPressureSensorData)
	token.Wait()
	token = client.Subscribe(name+"/ctrlStB", 1, binaryPressureSensorCtrl)
	token.Wait()

	for { // main event loop for a binary pressure sensor, write your code in here!
		select {
		case data := <-bps.data:
			bps.currState = data
			for _, ws := range bps.ws {
				log.Println("writing message")
				err := ws.WriteMessage(websocket.TextMessage, data)
				if err != nil {
					log.Println("ws write:", err)
				}
			}
		case ctrl := <-bps.ctrl:
			_ = ctrl
		}
	}
}

func binaryPressureSensorData(client mqtt.Client, msg mqtt.Message) {
	data := msg.Payload()
	sensorname := strings.Split(msg.Topic(), "/")[0]
	log.Printf("got data %s from binary pressure sensor %s", data, sensorname)

	allBinaryPressureSensorsMut.RLock() // find our struct from the sensor name
	bps, ok := allBinaryPressureSensors[sensorname]
	allBinaryPressureSensorsMut.RUnlock()
	if !ok {
		log.Printf("I don't know who that sensor is!")
		return
	}
	bps.data <- data // pipe to worker function
}

func binaryPressureSensorCtrl(client mqtt.Client, msg mqtt.Message) {
	ctrl := msg.Payload()
	sensorname := strings.Split(msg.Topic(), "/")[0]
	log.Printf("got ctrl %s from binary pressure sensor %s", ctrl, sensorname)

	if string(ctrl) == "ping" { // we know how to handle this one, doesn't need to go to worker
		token := client.Publish(sensorname+"/ctrlBtS", 0, false, "pong")
		token.Wait()
	}

	allBinaryPressureSensorsMut.RLock() // find our struct from the sensor name
	bps, ok := allBinaryPressureSensors[sensorname]
	allBinaryPressureSensorsMut.RUnlock()
	if !ok {
		log.Printf("I don't know who that sensor is!")
		return
	}
	bps.ctrl <- ctrl // pipe to worker function
}

func runBinaryPressureSensorWS(w http.ResponseWriter, r *http.Request) {
	keys, ok := r.URL.Query()["name"]
	if !ok || len(keys[0]) < 1 {
		log.Printf("url param type was missing")
		http.Error(w, "url param type missing", 400)
		return
	}
	name := keys[0]                     // sensor needs to hit ?name=[sensorname]
	allBinaryPressureSensorsMut.RLock() // find struct from the sensor name
	bps, ok := allBinaryPressureSensors[name]
	allBinaryPressureSensorsMut.RUnlock()
	if !ok {
		log.Printf("unknown sensor name")
		http.Error(w, "unknown sensor name", 400)
		return
	}

	c, err := upgrader.Upgrade(w, r, nil) // upgrade to a WS connection
	if err != nil {
		log.Print("ws upgrade:", err)
		return
	}

	bps.ws = append(bps.ws, c) // we now have a running WS
	delindex := len(bps.ws) - 1
	c.WriteMessage(websocket.TextMessage, bps.currState) // catch listener up

	for { // we just log incoming WS messages in this case, but ignoring
		_, message, err := c.ReadMessage()
		if err != nil {
			log.Println("ws read:", err)
			break
		}
		log.Printf("received from : %s", message)
	}

	c.Close()
	bps.ws[delindex] = bps.ws[len(bps.ws)-1]
	bps.ws = bps.ws[:len(bps.ws)-1]
}

func binaryPressureSensorWebpage(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "binaryPressureSensor.html")
}
