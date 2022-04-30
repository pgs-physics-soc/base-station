from paho.mqtt import client as mqtt
import requests


broker = 'localhost'
port = 1883


def connect_mqtt(cid):
    def on_connect(client, userdata, flags, rc):
        if rc == 0:
            print("Connected to MQTT Broker!")
        else:
            print("Failed to connect, return code %d\n", rc)
    # Set Connecting Client ID
    client = mqtt.Client(cid)
    client.on_connect = on_connect
    client.connect(broker, port)
    return client


def subscribe(client, topic):
    def on_message(client, userdata, msg):
        print(f"Received `{msg.payload.decode()}` from `{msg.topic}` topic")

    client.subscribe(topic)
    client.on_message = on_message


def publish(client, topic, msg):
    result = client.publish(topic, msg)
    status = result[0]
    if status == 0:
        print(f"Send `{msg}` to topic `{topic}`")
    else:
        print(f"Failed to send message to topic {topic}")


x = requests.get('http://localhost:8080/newsensor?type=binary-pressure-sensor')
cid = x.content.decode()
print("I am", cid)

client = connect_mqtt(cid)
client.loop_start()
subscribe(client, f"{cid}/ctrlBtS")
publish(client, f"{cid}/ctrlStB", "ping")

data = 0
while True:
    input("> ")
    publish(client, f"{cid}/data", f"{data}")
    data = [1, 0][data]
