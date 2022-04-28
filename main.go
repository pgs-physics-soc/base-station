package main

// Note: I'm using .Wait() a lot. When this is working well, I should run statistics
// on the number of goroutines running and see if it's spiralling - if it is we might
// need to not .Wait().

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var sensorId int = 0
var sensorIdMut sync.Mutex
var client mqtt.Client

var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	log.Printf("received %s from %s, don't know what to do with it", msg.Payload(), msg.Topic())
}

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	log.Printf("Connected to mosquitto broker")
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	log.Printf("Connection lost: %v", err)
}

func binaryPressureSensor(client mqtt.Client, msg mqtt.Message) {
	data := string(msg.Payload())
	sensorname := strings.Split(msg.Topic(), "/")[0]
	log.Printf("got data %s from binary pressure sensor %s", data, sensorname)
}

func binaryPressureSensorCtrl(client mqtt.Client, msg mqtt.Message) {
	ctrl := string(msg.Payload())
	sensorname := strings.Split(msg.Topic(), "/")[0]
	log.Printf("got ctrl %s from binary pressure sensor %s", ctrl, sensorname)

	if ctrl == "ping" {
		token := client.Publish(sensorname+"/ctrlBtS", 0, false, "pong")
		token.Wait()
	}
}

func root(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "web srvr\n")
}

func newsensor(w http.ResponseWriter, req *http.Request) {
	keys, ok := req.URL.Query()["type"]
	if !ok || len(keys[0]) < 1 {
		log.Printf("url param type was missing")
		http.Error(w, "url param type missing", 400)
		return
	}
	var dataHandler mqtt.MessageHandler
	var ctrlHandler mqtt.MessageHandler
	typ := keys[0] // sensor needs to hit <ip>/newsensor?type=[sometype]

	switch typ {
	case "binary-pressure-sensor":
		dataHandler = binaryPressureSensor
		ctrlHandler = binaryPressureSensorCtrl
	default:
		log.Printf("unimplemented sensor type %s", typ)
		http.Error(w, "unimplemented sensor type", 400)
		return
	}

	sensorIdMut.Lock() // following code is atomic
	sensorId += 1
	sensorname := fmt.Sprintf("sensor%d", sensorId)
	sensorIdMut.Unlock()

	fmt.Fprintf(w, sensorname) // send the name to the sensor
	token := client.Subscribe(sensorname+"/data", 1, dataHandler)
	token.Wait()
	token = client.Subscribe(sensorname+"/ctrlStB", 1, ctrlHandler)
	token.Wait()
}

func main() {
	var broker = "localhost"
	var port = 1883
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", broker, port))
	opts.SetClientID("server")
	opts.SetDefaultPublishHandler(messagePubHandler)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler
	client = mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}

	http.HandleFunc("/newsensor", newsensor)
	http.HandleFunc("/", root)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
