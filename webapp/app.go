package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gotsunami/go-cloudinary"
	"github.com/streadway/amqp"
)

// Payload contains data for the view
type Payload struct {
	FullImages    []string
	ResizedImages []string
}

func sendToMQ(publicID string) {
	amqpURL := os.Getenv("CLOUDAMQP_URL")

	connection, _ := amqp.Dial(amqpURL)
	defer connection.Close()
	channel, _ := connection.Channel()
	defer channel.Close()

	msg := amqp.Publishing{
		DeliveryMode: 1,
		ContentType:  "text/plain",
		Body:         []byte(publicID),
	}

	channel.QueueDeclare("images", true, false, false, false, nil)
	channel.Publish("", "images", false, false, msg)
}

func uploadImage(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(5120)

	image, handler, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "", http.StatusBadRequest)
		return
	}
	defer image.Close()

	service, err := cloudinary.Dial(os.Getenv("CLOUDINARY_URL"))
	if err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	publicID, err := service.UploadImage(handler.Filename, image, "full")
	if err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
	sendToMQ(publicID)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func getImages(w http.ResponseWriter, r *http.Request) {
	service, err := cloudinary.Dial(os.Getenv("CLOUDINARY_URL"))
	if err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	resources, err := service.Resources(cloudinary.ImageType)
	if err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	fullImages := make([]string, 0)
	for i := range resources {
		if strings.HasPrefix(resources[i].PublicId, "full/") {
			fullImages = append(fullImages, resources[i].Url)
		}
	}

	resizedImages := make([]string, 0)
	for i := range resources {
		if strings.HasPrefix(resources[i].PublicId, "resized/") {
			resizedImages = append(resizedImages, resources[i].Url)
		}
	}

	json.NewEncoder(w).Encode(Payload{FullImages: fullImages, ResizedImages: resizedImages})
}

func main() {
	http.Handle("/", http.FileServer(http.Dir("webapp/static")))
	http.HandleFunc("/upload", uploadImage)
	http.HandleFunc("/images", getImages)

	fmt.Println("Listening to " + os.Getenv("PORT") + "...")
	http.ListenAndServe(":"+os.Getenv("PORT"), nil)
}
