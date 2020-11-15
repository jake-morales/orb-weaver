package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"golang.org/x/net/html"
)

func getJSONbytes(topic string, maxPages int) ([]byte, error) {
	var js []byte
	var err error
	topic = strings.ToLower(topic)

	// Check the cache first
	js, err = checkCache(fmt.Sprintf("./static/cache/%s-%d.json", topic, maxPages))
	if err == nil {
		fmt.Println("Cache worked!")
		return js, nil
	}
	fmt.Printf("Did not use cache: %s\n", err)

	resp, err := http.Get(fmt.Sprintf("https://wikipedia.org/wiki/%s", topic))
	if err != nil {
		return js, errors.New("The server was unable to get the topic wiki page")
	}
	if resp.StatusCode != 200 {
		return js, errors.New("Topic does not exist")
	}

	js, err = crawl(topic, maxPages)
	if err != nil {
		return js, fmt.Errorf("Crawl failed: %s", err)
	}

	return js, nil
}

func checkCache(s string) ([]byte, error) {
	js, err := ioutil.ReadFile(s)
	if err != nil {
		return js, fmt.Errorf("Cache miss: %s", err)
	}

	return js, nil
}

// Node is for the treeeeees
type Node struct {
	Value    string  `json:"val"`
	Children []*Node `json:"children"`
}

func crawl(topic string, maxPages int) ([]byte, error) {
	numPages := 0
	numTopics := 0
	root := Node{Value: topic}
	queue := []*Node{}

	queue = append(queue, &root)

	for len(queue) > 0 && numPages < maxPages {
		node := queue[0]
		queue = queue[1:]

		links, err := scrape(node.Value)
		if err != nil {
			fmt.Printf("x")
		}
		numPages++

		for _, l := range links {
			newNode := Node{Value: l}
			node.Children = append(node.Children, &newNode)
			queue = append(queue, &newNode)
			numTopics++
		}
	}

	// create our file to write the json to
	w, err := os.Create(fmt.Sprintf("./static/cache/%s-%d.json", topic, maxPages))
	if err != nil {
		return []byte{}, fmt.Errorf("Error creating new json file %s", err)
	}
	defer w.Close()

	// serialize the JSON to our io.Writer stream
	err = json.NewEncoder(w).Encode(root)
	if err != nil {
		return []byte{}, fmt.Errorf("Error encoding, %s", err)
	}

	b, _ := json.Marshal(root)
	if err != nil {
		return []byte{}, fmt.Errorf("Error marshalling: %s", err)
	}
	return b, nil

	// fmt.Println("")
	// fmt.Printf("# topics crawled: %d\n", numTopics)
	// fmt.Printf("# pages crawled %d\n", numPages)
	// fmt.Printf("# leftover topics: %d\n", len(queue))

	// Print in console. make a better tree later?
	// fmt.Println("")
	// fmt.Println("")
	// b, _ := json.MarshalIndent(root, "", "\t")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// fmt.Println(string(b))
}

func scrape(s string) ([]string, error) {
	links := []string{}

	// Get wiki page
	res, err := http.Get(fmt.Sprintf("https://wikipedia.org/wiki/%s", s))
	if err != nil {
		return links, fmt.Errorf("Error: %s", err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return links, fmt.Errorf("Error getting webpage: " + res.Status)
	}
	if !strings.HasPrefix(res.Header.Get("Content-Type"), "text/html") {
		return links, fmt.Errorf("Response content type was %s not text/html", res.Header.Get("Content-Type"))
	}

	a := res.Body
	z := html.NewTokenizer(a)

	// The See Also section is structured like this in the html
	// <h2>
	// 		<span class="mw-headline" id="See_also">See also</span>
	// </h2>
	// <ul>
	// 		<li><a href="/wiki/Australian_native_bees" title="Australian native bees">Australian native bees</a></li>
	// 		<li><a href="/wiki/Superorganism" title="Superorganism">Superorganism</a></li>
	// </ul>

	for {
		tt := z.Next()
		switch tt {
		case html.ErrorToken:
			return links, nil
		case html.StartTagToken:
			token := z.Token()

			// check if it's a span html element with id="See_Also"
			if token.Data != "span" {
				continue
			}
			id := ""
			for _, a := range token.Attr {
				if a.Key == "id" {
					id = a.Val
				}
			}
			if id != "See_also" {
				continue
			}

			// grab all anchors until you see end </ul>
			for {
				tt := z.Next()
				switch tt {
				case html.StartTagToken:
					token := z.Token()
					if token.Data == "a" {
						for _, a := range token.Attr {
							if a.Key == "title" && !strings.HasPrefix(a.Val, "Edit section") {
								links = append(links, a.Val)
								break
							}
						}
					}
				case html.EndTagToken:
					token := z.Token()
					if token.Data == "ul" {
						return links, nil
					}
				}
			}
		}
	}
}
