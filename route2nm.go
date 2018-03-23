/*
 * Network to Network Manager Route Converter
 * Copyright (c) Lindsay Steele - 2018.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package main

import (
	"fmt"
	"os"
	"io/ioutil"
	"bufio"
	"strings"
	"regexp"
	"sort"
	"log"
	"strconv"
	"time"
)

// Create a type that handles a single network routing line

type route struct {
	ipaddress string // An ip address
	netmask   string // The network Prefix as a string
	gateway   string // The network gateway
	firstIPOctet int // The first Octect of the IP Address used for sorting
}

func main() {

	// Get the network files to parse,  and validate that it exists
	filename := getFileName()

	// Create empty slice of struct pointers.
	routes := []route{}

	// Get a string from the file
	oldRoutesFile := getFileString(filename)

	// Loop through the string looking at each line and extracting possible routes
	// put any routes found into a slice of routes
	routes = getRoutes(oldRoutesFile, routes)

	// Sort the slice by IP Addresses ... needs work .... special function maybe but for now it sorts similar addresses
	// together which is handy for fault finding
	sort.SliceStable(routes, func(i, j int) bool { return routes[i].firstIPOctet < routes[j].firstIPOctet })

	// Get Routes in the Network Manager Format
	routesNMFormat := getRoutesNMFormat(routes)

	//Display New Routes to Screen
	fmt.Println(routesNMFormat)

	// Write New Routes to Disk
	fmt.Println(writeProcessedLogFileToDisk(getNextFileName(filename, 0), routesNMFormat))
}

func getRoutesNMFormat(routes []route) string {

	routeNMFormat := ""
	count := 0

	for _, newRoute := range routes {

		routeNMFormat = routeNMFormat + "ADDRESS" + fmt.Sprint(count) + "=" + newRoute.ipaddress + "\n"
		routeNMFormat = routeNMFormat + "NETMASK" + fmt.Sprint(count) + "=" + newRoute.netmask + "\n"
		routeNMFormat = routeNMFormat + "GATEWAY" + fmt.Sprint(count) + "=" + newRoute.gateway + "\n"
		routeNMFormat = routeNMFormat + "METRIC" + fmt.Sprint(count) + "=0\n"

		count++
	}

	return routeNMFormat
}

func getRoutes(oldRoutesFile string, routes []route) []route {

	// Regex to Detect IP Addresses in line

	// Loop through the String
	scanner := bufio.NewScanner(strings.NewReader(oldRoutesFile))
	for scanner.Scan() {

		// Trim Line of spaces
		routeLine := strings.Trim(scanner.Text(), " ")
		if len(routeLine) < 1 {
			continue
		}

		// If this line starts with an number which indicate an IP address then we process it, otherwise with move on
		match, _ := regexp.MatchString("^([1-9])", routeLine)
		if !match {
			//fmt.Println(routeLine, "does not match number")
			continue
		}

		// Get a route struct
		ipRoute := new(route)

		// At this stage we have a line starting with a number, we need to extract the parts

		// Get the IP and Split into IP and Mask
		lineSplit := strings.Split(routeLine, " ")

		ipAndMask := strings.Split(lineSplit[0], "/")

		// Assign values to ipRoute Object
		ipRoute.ipaddress = ipAndMask[0]
		ipRoute.netmask = getExpandedNetmask(ipAndMask[1])
		ipRoute.gateway = lineSplit[2] // may need better error checking
		ipRoute.firstIPOctet = getFirstOctet(ipAndMask[0])

		// append route to Routes
		routes = append(routes, *ipRoute)

		//fmt.Println(scanner.Text())
	}

	// Reject Line that says Default = Maybe Export this line into a Seperate File??

	// Reject lines starting with a hash or that are empty

	// Extract elements from Found Line

	return routes
}
func getFirstOctet(IPAddress string) int {

	ipSplit := strings.Split(IPAddress, ".")

	firstOctect, err := strconv.Atoi(ipSplit[0])
	if err != nil {
		// handle error
		fmt.Println(err)
		os.Exit(2)
	}

	return firstOctect
}



func getFileName() string {

	// Check we got arguments passwed in,  exit if none
	if len(os.Args) < 2 {
		fmt.Println("Please give the name of the file to be converted")
		os.Exit(0)
	}

	// First parameters is looked at only
	filename := os.Args[1]

	//Check if file exists
	if !fileExists(filename) {
		fmt.Println("Cannot find the file specified.")
		os.Exit(0)
	}

	return filename
}

// Function to Return true/false depending on whether a file exists
func fileExists(filename string) bool {

	if _, err := os.Stat(filename); os.IsNotExist(err) {
		//File does not exist
		return false
	}
	// File does exist
	return true
}

// Function to read the file into a string
func getFileString(logfile string) string {

	fileBytes, err := ioutil.ReadFile(logfile)
	if err != nil {
		// Process a log file name error here ...
		//log.Fatal(err)
		fmt.Println(err)
		os.Exit(0)
	}
	//defer ioutil.close(logfile)
	return string(fileBytes)
}

func writeProcessedLogFileToDisk(filename string, routesNMFormat string) string {

	err := ioutil.WriteFile(filename, []byte(routesNMFormat), 0644)

	if err != nil {
		log.Fatal(err)
		return err.Error()
	}

	return "\nNew Filename  " + filename + " written successfully"
}

func getNextFileName(fileame string, count int) string {

	// As we are using recursion in this function just do a sanity check on the number of files created to avoid unseen file system errors
	if count > 98 {
		fmt.Println("Too many sanitised files found for today - or file system error")
		os.Exit(0)
	}

	// Pad Integer
	var strCount = strconv.Itoa(count)
	if len(strCount) < 2 {
		strCount = "0" + strCount
	}

	// Get Todays Day in a String
	t := time.Now()
	dateString := t.Format("2006-01-02")

	// check if the filename exists
	filename := fileame + "_" + dateString + "_" + strCount
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		//fmt.Println("File does not exist", filename)
		return filename
	}

	// Recursion back to the same function if the file exists
	count++
	return getNextFileName(fileame, count)
}


// Messy Messy Function,  must be a better way to do this ... sticking at the end of the file
func getExpandedNetmask(shortNetmask string) string {

	switch shortNetmask {
	case "4":
		return "240.0.0.0 "
	case "5":
		return "248.0.0.0 "
	case "6":
		return "252.0.0.0 "
	case "7":
		return "254.0.0.0 "
	case "8":
		return "255.0.0.0"
	case "9":
		return "255.128.0.0 "
	case "10":
		return "255.192.0.0 "
	case "11":
		return "255.224.0.0 "
	case "12":
		return "255.240.0.0 "
	case "13":
		return "255.248.0.0 "
	case "14":
		return "255.252.0.0"
	case "15":
		return "255.254.0.0"
	case "16":
		return "255.255.0.0"
	case "17":
		return "255.255.128.0"
	case "18":
		return "255.255.192.0"
	case "19":
		return "255.255.224.0"
	case "20":
		return "255.255.240.0"
	case "21":
		return "255.255.248.0"
	case "22":
		return "255.255.252.0"
	case "23":
		return "255.255.254.0"
	case "24":
		return "255.255.255.0"
	case "25":
		return "255.255.255.128"
	case "26":
		return "255.255.255.192"
	case "27":
		return "255.255.255.224"
	case "28":
		return "255.255.255.240"
	case "29":
		return "255.255.255.248"
	case "30":
		return "255.255.255.252"
	case "32":
		return "255.255.255.255"
	default:
		// If no match, maybe it is in the long format already?
		return shortNetmask
	}

}