// From https://github.com/lazywei/go-opencv/blob/master/samples/webcam.go
package main

// +build linux darwin
// +build 386

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"image/jpeg"
	"log"
	"os"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/lazywei/go-opencv/opencv"
)

type mqttConfig struct {
	Host  string
	Port  string
	User  string
	Pass  string
	Topic string
}
type mqttMessage struct {
	Type  string `json:"type"`
	Image []byte `json:"dataURL,omitempty"`
}

var f MQTT.MessageHandler = func(client *MQTT.Client, msg MQTT.Message) {
	var message mqttMessage
	if err := json.Unmarshal(msg.Payload(), &message); err != nil {
		log.Println(err)
	}
	fmt.Printf("TOPIC: %s --> MSG: %s\n", msg.Topic(), message.Type)
}

func mqttConnect(config mqttConfig) *MQTT.Client {
	broker := "ssl://" + config.Host + ":" + config.Port
	log.Println("MQTT attempting to connect to", broker)
	opts := MQTT.NewClientOptions().AddBroker(broker)
	opts.SetClientID("go-webcam")
	opts.SetCleanSession(true)
	opts.SetUsername(config.User)
	opts.SetPassword(config.Pass)
	opts.SetDefaultPublishHandler(f)
	opts.SetWill("will", "Goodbye", 1, true)

	// create and start a client using the above ClientOptions
	c := MQTT.NewClient(opts)
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	return c
}

func mqttPublish(config mqttConfig, client *MQTT.Client, message *mqttMessage) {
	payload, _ := json.Marshal(message)
	token := client.Publish(config.Topic+"/photo", 0, false, string(payload))
	token.Wait()
}

func main() {
	win := opencv.NewWindow("Go-OpenCV Webcam")
	defer win.Destroy()

	cap := opencv.NewCameraCapture(0)
	if cap == nil {
		panic("can not open camera")
	}
	defer cap.Release()

	hostPtr := flag.String("host", "127.0.0.1", "MQTT Host")
	portPtr := flag.String("port", "1883", "MQTT Port")
	userPtr := flag.String("user", "username", "MQTT Username")
	passPtr := flag.String("pass", "password", "MQTT Password")
	flag.Parse()
	config := mqttConfig{Host: *hostPtr, Port: *portPtr, User: *userPtr, Pass: *passPtr, Topic: "gate"}
	mqttClient := mqttConnect(config)

	// subscribe to the topic and request messages to be delivered at a
	// maximum qos of zero, wait for the receipt to confirm the subscription
	var qos byte
	qos = 0
	if token := mqttClient.Subscribe(config.Topic+"/#", qos, nil); token.Wait() && token.Error() != nil {
		log.Println(token.Error())
		os.Exit(1)
	}

	mqttPublish(config, mqttClient, &mqttMessage{Type: "TRAIN"})

	log.Println("Press ESC to quit")
	for {
		if cap.GrabFrame() {
			imgCV := cap.RetrieveFrame(1)
			if imgCV != nil {
				imgCV = opencv.Resize(imgCV, 300, 300, 0)
				img := imgCV.ToImage()
				buf := new(bytes.Buffer)
				if err := jpeg.Encode(buf, img, nil); err != nil {
					log.Println("unable to encode image.")
				}
				imgEncoded := buf.Bytes()
				mqttPublish(config, mqttClient, &mqttMessage{Type: "FRAME", Image: imgEncoded})
				// TODO: Show annotated image
				win.ShowImage(imgCV)
				time.Sleep(2 * time.Second)
			} else {
				log.Println("Image is nil")
			}
		}
		key := opencv.WaitKey(10)

		if key == 27 {
			os.Exit(0)
		}
	}
}
