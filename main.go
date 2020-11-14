package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/xlab/treeprint"
	"golang.org/x/net/html"
)

// "/wiki/Special:random" for a random article

// Link is struct for links
type Link struct {
	topic string
	node  treeprint.Tree
}

func main() {
	// HTTP server
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	log.Printf("Listening on :%s...", port)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatal(err)
	}

	// Controls
	maxTopics := 50
	pagesCrawled := 0

	// check root topic
	if len(os.Args) <= 1 {
		log.Fatal("Please run command with a topic. e.g. go run crawl_seealso.go Bees")
	}
	rootTopic := os.Args[1]
	resp, err := http.Get(fmt.Sprintf("https://wikipedia.org/wiki/%s", rootTopic))
	if err != nil {
		log.Fatal("Couldn't fetch webpage. Are you connected to the internet?")
	}
	if resp.StatusCode != 200 {
		log.Fatal("That topic does not exist")
	}

	numTopics := 0
	root := treeprint.New()
	queue := []Link{}

	root.SetValue(rootTopic)
	queue = append(queue, Link{topic: rootTopic, node: root})

	fmt.Printf("Running")

	for len(queue) > 0 && numTopics < maxTopics {
		link := queue[0]
		queue = queue[1:]

		terms, _ := getLinks(link.topic)
		if err != nil {
			fmt.Printf("x")
		} else {
			fmt.Printf(".")
			pagesCrawled++
		}

		for _, t := range terms {
			newBranch := link.node.AddBranch(t)
			queue = append(queue, Link{topic: t, node: newBranch})
			numTopics++
		}
	}

	fmt.Println("")
	fmt.Println("")
	fmt.Println(root.String())
	fmt.Printf("# topics crawled: %d\n", numTopics)
	fmt.Printf("# pages crawled %d\n", pagesCrawled)
	fmt.Printf("# leftover topics: %d\n", len(queue))
}

func getLinks(s string) ([]string, error) {
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
