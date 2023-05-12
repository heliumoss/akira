package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/mux"
)

type AverageResponse struct {
	Message string `json:"message"`
	Error   bool   `json:"error"`
}

type ImageReturn struct {
	Images []Image `json:"images"`
	Error  bool    `json:"error"`
}

type Image struct {
	Size   string `json:"size"`
	Base64 string `json:"base64"`
}

func main() {
	// initialize router, set up routes
	router := mux.NewRouter()
	router.HandleFunc("/ping", GetPing).Methods("GET")
	router.HandleFunc("/", GetRoot).Methods("GET")
	router.HandleFunc("/resize", HandleImage).Methods("POST")
	router.Use(LoggingMiddleware)
	router.NotFoundHandler = http.HandlerFunc(NotFound)
	// fancy ascii art
	log.Println("    :::     :::    ::: ::::::::::: :::::::::      :::     ")
	log.Println("  :+: :+:   :+:   :+:      :+:     :+:    :+:   :+: :+:   ")
	log.Println(" +:+   +:+  +:+  +:+       +:+     +:+    +:+  +:+   +:+  ")
	log.Println("+#++:++#++: +#++:++        +#+     +#++:++#:  +#++:++#++: ")
	log.Println("+#+     +#+ +#+  +#+       +#+     +#+    +#+ +#+     +#+ ")
	log.Println("#+#     #+# #+#   #+#      #+#     #+#    #+# #+#     #+# ")
	log.Println("###     ### ###    ### ########### ###    ### ###     ### ")
	log.Println("")
	log.Println("Listening on port 8000. Ctrl+C to exit.")
	// create server

	srv := &http.Server{
		Handler:      router,
		Addr:         ":8000",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	// handle ctrl+c, exit gracefully
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Println("Oyasuminasai!")
		if err := srv.Shutdown(context.TODO()); err != nil {
			log.Fatal("Something happened while trying to gracefully shutdown: ", err)
		}
		os.Exit(1)
	}()
	// start server
	log.Fatal(srv.ListenAndServe())
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println(r.Method, r.RemoteAddr, r.URL)
		next.ServeHTTP(w, r)
	})
}

func GetPing(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AverageResponse{Message: "Pong! You've pinged Akira. This endpoint will be used to get stats for a status page in the future.", Error: false})
}

func GetRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AverageResponse{Message: "You've bumped into Akira. You probably shouldn't be here.", Error: false})
}

func HandleImage(w http.ResponseWriter, r *http.Request) {
	reqTime := time.Now()
	r.ParseMultipartForm(10 << 20)
	if !r.PostForm.Has("size") {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(AverageResponse{Message: "You need to provide an image and a size.", Error: true})
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	quality := 100

	if r.PostForm.Has("quality") {
		conv, err := strconv.Atoi(r.PostFormValue("quality"))
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(AverageResponse{Message: "Quality must be a number.", Error: true})
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if conv > 100 || conv < 0 {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(AverageResponse{Message: "Quality must be between 0 and 100.", Error: true})
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		quality = conv
	}

	// size is a string formatted like this: 64x64;128x128;256x256;512x512;1024x1024.
	// we need to split this into an array of strings.

	sizes := strings.Split(r.PostFormValue("size"), ";")

	file, _, err := r.FormFile("image")
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(AverageResponse{Message: "Something went wrong while trying to read the image.", Error: true})
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer file.Close()

	start := time.Now()
	byteContainer, err := ioutil.ReadAll(file)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(AverageResponse{Message: "Something went wrong while trying to read the image.", Error: true})
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	log.Println("Read image in", time.Since(start))

	// loop through the array of sizes and resize the image for each size. preferably using goroutines.
	// also create an array to hold these images.

	poolSize := 5
	jobs := make(chan string, len(sizes))
	results := make(chan Image, len(sizes))

	for i := 0; i < poolSize; i++ {
		go func() {
			for size := range jobs {
				if size == "" {
					results <- Image{
						Size:   "",
						Base64: "",
					}
					continue
				}
				start := time.Now()
				img, err := processImage(byteContainer, size, quality)
				if err != nil {
					log.Println(err)
					results <- img
					continue
				}
				log.Println("Processed image in", time.Since(start), " | The size requested was", size)
				results <- img
			}
		}()
	}

	for _, size := range sizes {
		jobs <- size
	}
	close(jobs)

	var images []Image

	for i := 0; i < len(sizes); i++ {
		img := <-results
		if img.Base64 == "" {
			continue
		}
		images = append(images, img)
	}

	// return the images array as json
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ImageReturn{Error: false, Images: images})
	log.Println("Request took", time.Since(reqTime), "to complete. Sizes requested:", sizes)
	return

}

func NotFound(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AverageResponse{Message: "Route does not exist.", Error: true})
	w.WriteHeader(http.StatusNotFound)
}
