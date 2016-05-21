package main

import ("fmt"
		"net/http"
		//"strconv"
		"io/ioutil"
		//m "math"
		)
func main() {

	req, err := http.Get("http://tsy.acislive.com/pip/stop_simulator.asp?naptan=22001688")
	
	if err == nil{
		defer req.Body.Close()
		response, err := ioutil.ReadAll(req.Body)
		fmt.Println(err)
		fmt.Println("\nThe Server said: ", string(response))
	} else {
		fmt.Println(err)
	}

}