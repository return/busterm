package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/docopt/docopt-go"
	"gopkg.in/ukautz/clif.v1"
)

var usage = `busterm

View all the NapTAN buses directly in realtime in the terminal!

Usage:
	busterm [-t] -n <code> | --naptan <code> [<interval>] 
	busterm -a | --api
	busterm -h | --help
	busterm --version

Options:
	-h --help     Show this screen.
	--version     Show version.`

var (
	// json errors.
	unable        = `{"error":"unable to fetch buses."}`
	invalidNaptan = `{"error":"NapTAN code must be an 8 digit number."}`

	// baseurl.
	baseurl = "http://yorkshire.acisconnect.com/Text/WebDisplay.aspx"

	unwantedRunes = "aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ;:\\'\"{[}]\\|+=-_)(*&^%$#@!~`<>?"
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

// API launches the busterm API server.
func API() {
	// Create a logger for the server endpoints.
	logger := log.New(os.Stdout, "", log.Ldate)
	// Create /check_buses route for our server.
	http.HandleFunc("/check_buses", func(w http.ResponseWriter, r *http.Request) {
		// Add headers.
		w.Header().Add("Accept", "application/json")
		w.Header().Add("Content-Type", "application/json")
		logger.Println(r.Method, r.Host, r.RequestURI) // GET (host) endpoint/params

		// Get the naptan code.
		code := r.URL.Query().Get("naptan")
		err := checkCode(code)
		if err != nil {
			w.WriteHeader(400)
			fmt.Fprintf(w, string(invalidNaptan))
			return
		}

		// Get Buses.
		buses, err := getBuses(code)
		if err != nil {
			w.WriteHeader(400)
			fmt.Fprintf(w, string(unable))
			return
		}

		// Turn buses into JSON.
		data, err := json.Marshal(buses)
		if err != nil {
			w.WriteHeader(400)
			fmt.Fprintf(w, string(unable))
			return
		}
		w.WriteHeader(200)
		fmt.Fprintf(w, string(data))
		return
	})

	// Listen on port :7654
	// TODO: For production usecases change 'localhost' to 7654.
	// Only do this when deploying on a real server.
	port := "7654"
	fmt.Println("busterm API is up on port :" + port)
	http.ListenAndServe("localhost:"+port, nil)
}

// PrintBus prints an estimated measure of how close the bus is from the bus stop.
func PrintBus(timestring string, doubledecker bool) string {
	// Emojis for buses.
	var bus = "🚌"
	var stop = "🚏"
	var emoji string
	// roads or (_) for the bus.
	var roads int
	// time unit default is 12.
	var unit = 12
	// converted time and current time.
	var convtime time.Time
	now := time.Now()

	// Check if its a double decker bus.
	if doubledecker == true {
		// Until a double decker bus is introduced into the unicode standard,
		// this one will suffice.
		bus = "🚐"
	}
	// format the date as '2006-01-02'
	fmtdate := (func() string {
		var specifier string
		y, m, d := time.Now().Date()
		if m < 10 {
			specifier = "%d-0%d-%d"
		}
		if d < 10 {
			specifier = "%d-%d-0%d"
		}
		if d < 10 && m < 10 {
			specifier = "%d-0%d-0%d"
		}
		return fmt.Sprintf(specifier, y, m, d)
	})
	// check if the string does not have a colon (:)
	tt := strings.Fields(timestring)[0]
	if strings.ContainsAny(tt, ":") != true {
		// append "m" (minutes) since the time is in minutes.
		i, _ := time.ParseDuration(tt + "m")
		convtime = now.Add(i)
		// append "m" (minutes) since the time is in minutes.
		unit = 5
	} else {
		// parse the time.
		parsedtime, _ := time.Parse("2006-01-02 15:04", fmtdate()+" "+tt)
		convtime = parsedtime
	}
	// check if the units is 5 minutes or 12 hours
	// This is needed to check how close the bus is from the bus stop depending on the units
	if unit == 5 {
		roads = int(convtime.Sub(now).Minutes()) / unit
	} else {
		roads = int(convtime.Sub(now).Hours()) % unit
	}
	// Is the bus "Due"? or is the time less than 5 minutes?
	if roads <= 0 {
		return fmt.Sprintf("%s\r", "_"+strings.Repeat("_", 0)+bus+strings.Repeat("_", unit))
	}
	// Assuming bus is still on its way only print bus stop for Buses which have no time colon.
	if strings.ContainsAny(tt, ":") {
		emoji = fmt.Sprintf("%s\r", "_"+strings.Repeat("_", int(roads))+bus+strings.Repeat("_", unit))
	} else {
		emoji = fmt.Sprintf("%s\r", "_"+stop+strings.Repeat("_", int(roads))+bus+strings.Repeat("_", unit))
	}
	return emoji
}

// PrintTable prints the timetable to the screen.
func PrintTable(bus []Bus, ref string) {
	c := clif.NewColorOutput(os.Stdin)
	// Headers and Rows.
	headers := []string{"Bus", "To", "Time", "Emoji", "Double Decker"}
	rows := [][]string{}
	// Loop over the Buses and append them to the rows.
	for _, b := range bus {
		s := []string{
			strconv.Itoa(b.Service),
			"<warn>" + b.To + "<reset>",
			b.Time,
			PrintBus(b.Time, b.DoubleDecker),
			strconv.FormatBool(b.DoubleDecker),
		}
		rows = append(rows, s)
	}
	table := c.Table(headers, clif.OpenTableStyleLight)
	table.AddRows(rows)

	// Parse current time in simple form. (3:04PM)
	now := time.Now().Format(time.Kitchen)
	// Print the timetable with time and stop reference.
	c.Printf("\rDeparture information for at " + "<query>" + now + "<reset>\n")
	c.Printf("\r\nLegend: \n🚏 : Bus Stop \n🚌 : Normal Bus\n🚐 : Double Decker Bus\n")
	c.Printf("\rStop Ref: <headline>%s<reset>\n\n%s\n", ref, table.Render())
}

// checkCode checks if the NapTAN is valid.
func checkCode(code string) error {
	if len(code) != 8 || strings.ContainsAny(code, unwantedRunes) {
		return errors.New("NapTAN code must be an <error>8 digit number.<reset>\n")
	}
	return nil
}

func main() {
	// Parse arguments.
	var ref string
	c := clif.NewColorOutput(os.Stdin)
	arguments, _ := docopt.Parse(usage, nil, true, "busterm", false)

	// Check NapTAN option.
	if arguments["-n"] == true || arguments["--naptan"] == true {
		code := arguments["<code>"].(string)
		err := checkCode(code)
		if err != nil {
			c.Printf(err.Error())
			os.Exit(1)
		}
		ref = code
		if arguments["-t"] == true {
			fmt.Print("\033[2J")
			for {
				buses, err := getBuses(ref)
				if err != nil {
					c.Printf("<error>%s<reset>\n", err)
					os.Exit(1)
				}
				// Clear the screen and print table.
				// Remove any previous messages and wait 30 seconds.
				c.Printf("\033[1;1H")
				PrintTable(buses, ref)
				fmt.Printf("\r           \r")
				time.Sleep(30 * time.Second)
				fmt.Printf("\rUpdating...")
			}
		}
		// Get Buses.
		buses, err := getBuses(ref)
		if err != nil {
			c.Printf("<error>%s<reset>\n", err)
			os.Exit(1)
		}
		PrintTable(buses, ref)
	}

	// Serve the API.
	if arguments["-a"] == true || arguments["--api"] == true {
		API()
	}
}
