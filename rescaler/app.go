package main

import (
	"bytes"
	"fmt"
	"image"
	"net/http"
	"os"

	_ "image/gif"
	"image/jpeg"
	_ "image/png"

	cloudinary "github.com/gotsunami/go-cloudinary"
	"github.com/nfnt/resize"
	"github.com/streadway/amqp"
)

func processImage(imageID string) {
	service, err := cloudinary.Dial(os.Getenv("CLOUDINARY_URL"))
	if err != nil {
		fmt.Println(err)
		return
	}

	url := service.Url(imageID, cloudinary.ImageType)

	resp, err := http.Get(url)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()

	image, _, err := image.Decode(resp.Body)
	if err != nil {
		fmt.Println(err)
		return
	}

	resized := resize.Resize(32, 0, image, resize.Lanczos3)
	buffer := new(bytes.Buffer)
	jpeg.Encode(buffer, resized, nil)

	service.UploadImage(imageID+".jpg", buffer, "resized")
}

func main() {
	amqpURL := os.Getenv("CLOUDAMQP_URL")

	connection, err := amqp.Dial(amqpURL)
	if err != nil {
		panic(err)
	}
	defer connection.Close()

	channel, err := connection.Channel()
	if err != nil {
		panic(err)
	}
	defer channel.Close()

	queue, err := channel.QueueDeclare("images", true, false, false, false, nil)
	if err != nil {
		panic(err)
	}

	messages, err := channel.Consume(queue.Name, "", true, false, false, false, nil)
	if err != nil {
		panic(err)
	}

	for msg := range messages {
		imageID := string(msg.Body)
		processImage(imageID)
	}
}
