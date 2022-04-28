Base station provides web server and MQTT broker

Sensor starts up and connects to base station's WiFi
Sensor makes a request to the web server, saying what type it is
Web server responds with a sensorname. These are unique per-boot of the base station.
Base station subscribes to relevant topics for new sensorname.
Sensor sends a ping on `sensorname/ctrlStB`, and base station then sends a ping back on `sensorname/ctrlBtS`.
If this hasn't occurred within some timeout the sensor is free to start again from the top.

Sensor publishes data on `sensorname/data`
Sensor responds to control events / sends metadata on `sensorname/ctrlStB`
Base station sends control events on `sensorname/ctrlBtS`

There is no formal teardown if a sensor leaves, as inactive topics do not cost us anything: they can just be left.
Web interface views of sensors can also persist so that students can copy up experiments.