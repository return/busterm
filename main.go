package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

var (
	// flag variables.
	api  = flag.Bool("api", false, "launch the api")
	help = flag.Bool("h", false, "this help")

	// json errors.
	unable = `{"error":"unable to fetch buses."}`

	// baseurl.
	baseurl = "http://yorkshire.acisconnect.com/Text/WebDisplay.aspx"
)

// Bus struct holds information about a Bus from Yorkshire Buses.
type Bus struct {
	Service      int    `json:"bus"`
	To           string `json:"to"`
	Time         string `json:"time"`
	DoubleDecker bool   `json:"double_decker"`
}

// String converts a Bus into a string representable format.
func (bus Bus) String() string {
	var str string

	// Check if the time has the "Due" string.
	if bus.Time == "Due" {
		str = fmt.Sprintf("Bus %d going to %s is %s", bus.Service, bus.To, bus.Time)
		return str
	}

	// Check if the time has a colon seperator.
	if strings.ContainsAny(bus.Time, ":") == true {
		str = fmt.Sprintf("Bus %d going to %s @ %s", bus.Service, bus.To, bus.Time)
		return str
	}
	str = fmt.Sprintf("Bus %d going to %s in %s", bus.Service, bus.To, bus.Time)
	return str
}

// parse parses a HTML document and returns a collection of Buses. ([]Bus)
func parse(gs *goquery.Document) []Bus {

	// Create an array of Bus structs.
	buses := []Bus{}

	// Find the table tag (<table></table>) and table row tag (<tr></tr>) in the document.
	// Then iterate through them.
	gs.Find("table tr ").Each(func(y int, s *goquery.Selection) {

		// Create a bus structure to hold the current bus...
		bus := Bus{}

		// Find the table data tag (<td></td>) then iterate through them.
		s.Find("td").Each(func(x int, t *goquery.Selection) {
			// Use the index 'x' as a guide as the cell heading:
			// 0 = Service
			// 1 = To
			// 2 = Time
			// 3 = Low Floor (Small Bus)
			switch x {
			case 0:
				service, _ := strconv.Atoi(t.Text()) // should really check for error here.
				bus.Service = service
				break
			case 1:
				bus.To = t.Text()
				break
			case 2:
				bus.Time = t.Text()
			case 3:
				var f = true
				// False if its a small bus.
				// True if its a double decker.
				if t.Text() == "Yes" {
					f = false
				}
				bus.DoubleDecker = f
				break
			}
		})

		// ...and append a completed bus on each iteration.
		buses = append(buses, bus)
	})
	// Chop the first element off. (First element is the table heading)
	return buses[1:]
}

// getBuses fetches an array of buses by scraping from Yorkshire Buses.
func getBuses(ref string) ([]Bus, error) {
	// Make our very own HTTP client.
	client := &http.Client{}

	// Make custom useragent for the request.
	ua := "Mozilla/5.0 (Windows NT 6.2; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/30.0.1599.17 Safari/537.36"
	req, err := http.NewRequest("GET", baseurl+"?stopRef="+ref, nil)
	if err != nil {
		fmt.Println(err)
	}

	// Add user-agent for the request.
	req.Header.Add("User-Agent", ua)

	res, perr := client.Do(req) // Execute login request.
	if perr != nil {
		return []Bus{}, err
	} else if res.StatusCode != 200 {
		return []Bus{}, errors.New("status != 200: status:" + res.Status)
	}

	// Get a new HMTL document from yorkshire.acisconnect.com
	document, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return []Bus{}, err
	}

	// Parse the document.
	buses := parse(document)
	return buses, nil
}

// API launches the gotimetravel API server.
func API(ref string) {
	// Create /check_buses route for our server.
	http.HandleFunc("/check_buses", func(w http.ResponseWriter, r *http.Request) {
		log.Println(r) // TODO: Logging looks ugly, change at will.

		// Add headers.
		w.Header().Add("Accept", "application/json")
		w.Header().Add("Content-Type", "application/json")

		// Get Buses.
		buses, err := getBuses(ref)
		if err != nil {
			fmt.Fprintf(w, string(unable))
			return
		}

		// Turn buses into JSON.
		data, err := json.Marshal(buses)
		if err != nil {
			fmt.Fprintf(w, string(unable))
			return
		}

		fmt.Fprintf(w, string(data))
		return
	})

	// Listen on port :7654
	// TODO: For production usecases change 'localhost' to 7654.
	// Only do this when deploying on a real server.
	port := "7654"
	fmt.Println("gotimetravel API is up on port :" + port)
	http.ListenAndServe("localhost:"+port, nil)
}

func main() {
	// Command Line Flags.
	flag.Parse()

	// Reference code. TODO: (Hardcoded) Can be changed.
	ref := "22001688"

	// Serve the API.
	if *api == true {
		API(ref)
	}

	// Help docs.
	if *help == true {
		fmt.Println("gotimetravel help")
		flag.PrintDefaults()
		os.Exit(0)
	}

	// Get Buses.
	buses, err := getBuses(ref)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Parse current time in simple form. (3:04PM)
	now := time.Now().Format(time.Kitchen)

	// Print timetable with time and stop reference.
	fmt.Println("Departure information for at " + now)
	fmt.Println("Stop Ref: " + ref)
	fmt.Println("------------------------------------")

	// Print timetable.
	for _, bus := range buses {
		fmt.Println(bus)
	}
}
