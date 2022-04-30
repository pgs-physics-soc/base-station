package main

// Note: I'm using .Wait() a lot. When this is working well, I should run statistics
// on the number of goroutines running and see if it's spiralling - if it is we might
// need to not .Wait().

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gorilla/websocket"
)

var sensorId int = 0
var sensorIdMut sync.Mutex
var client mqtt.Client
var upgrader = websocket.Upgrader{} // use default options

var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	log.Printf("received %s from %s, don't know what to do with it", msg.Payload(), msg.Topic())
}
var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	log.Printf("Connected to mosquitto broker")
}
var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	log.Printf("Connection lost: %v", err)
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
	typ := keys[0] // sensor needs to hit <ip>/newsensor?type=[sometype]

	sensorIdMut.Lock() // following code is atomic
	defer sensorIdMut.Unlock()
	sensorId += 1
	sensorname := fmt.Sprintf("sensor%d", sensorId)

	switch typ {
	case "binary-pressure-sensor":
		go binaryPressureSensorRunner(sensorname)
	default:
		log.Printf("unimplemented sensor type %s", typ)
		http.Error(w, "unimplemented sensor type", 400)
		sensorId -= 1
		return
	}

	fmt.Fprint(w, sensorname) // send the name to the sensor
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

	allBinaryPressureSensors = make(map[string]*BinaryPressureSensor)

	http.HandleFunc("/newsensor", newsensor)
	http.HandleFunc("/binaryPressureSensor", binaryPressureSensorWebpage)
	http.HandleFunc("/binaryPressureSensorWS", runBinaryPressureSensorWS)
	http.HandleFunc("/", root)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
