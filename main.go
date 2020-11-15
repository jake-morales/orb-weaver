package main

import (
	"log"
	"strconv"

	"net/http"
	"os"
)

// "/wiki/Special:random" for a random article

// TODO - add date to json name and redo after 5 days
// TODO - compress json with gzip
// TODO - make crawling concurrent

func main() {

	// Controls
	//topicPtr := flag.String("topic", "Bees", "The root topic")
	//maxPagesPtr := flag.Int("pages", 5, "The max number of pages to crawl")
	//flag.Parse()

	// HTTP server
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	// Query Endpoint
	http.HandleFunc("/query", queryHandler)

	// Get port #
	port := os.Getenv("PORT")
	if port == "" {
		port = "80"
	}

	// Start server
	log.Printf("Listening on :%s...", port)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func queryHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	topic := q.Get("topic")
	if topic == "" {
		http.Error(w, "no topic provided", http.StatusBadRequest)
		return
	}

	pages := q.Get("pages")
	if q.Get("pages") == "" {
		http.Error(w, "no pages provided", http.StatusBadRequest)
		return
	}
	maxPages, _ := strconv.Atoi(pages)
	if maxPages > 20 {
		http.Error(w, "too many pages", http.StatusBadRequest)
		return
	}

	js, err := getJSONbytes(topic, maxPages)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}
