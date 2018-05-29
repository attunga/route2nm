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
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Holds information about the new route including information used for sorting
type route struct {
	ipaddress string // An ip address
	netmask   string // The network Prefix as a string
	gateway   string // The network gateway
	ipValue   int    // A calculated value from all octets used for sorting
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

	// Sorts by IPValue gives a proper sort by value rather than string representation.
	sort.SliceStable(routes, func(i, j int) bool { return routes[i].ipValue < routes[j].ipValue })

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
		ipRoute.ipValue = getIPValue(ipAndMask[0])

		// append route to Routes
		routes = append(routes, *ipRoute)

	}

	return routes
}

// Get the value of the IP address,  this is purely used for sorting
func getIPValue(IPAddress string) int {

	ipValue := 0

	// Split the IP address into for single numbers
	ipSplit := strings.Split(IPAddress, ".")

	firstOctect, err := strconv.Atoi(ipSplit[0])
	if err != nil {
		// handle error
		fmt.Println(err)
		os.Exit(2)
	}
	ipValue = ipValue + (firstOctect * 9000000)

	secondOctect, err := strconv.Atoi(ipSplit[1])
	if err != nil {
		// handle error
		fmt.Println(err)
		os.Exit(2)
	}
	ipValue = ipValue + (secondOctect * 60000)

	thirdOctect, err := strconv.Atoi(ipSplit[2])
	if err != nil {
		// handle error
		fmt.Println(err)
		os.Exit(2)
	}
	ipValue = ipValue + (thirdOctect * 300)

	forthOctect, err := strconv.Atoi(ipSplit[3])
	if err != nil {
		// handle error
		fmt.Println(err)
		os.Exit(2)
	}
	ipValue = ipValue + forthOctect

	return ipValue * 100
}

func getFileName() string {

	runningProgram := os.Args[0] // For clarity, the name of the executable used to start the program

	// Check we got arguments passed in,  exit if none
	if len(os.Args) < 2 {
		fmt.Println("Please give the name of the file to be converted as a parameter")
		fmt.Println("Usage: ", runningProgram, "[routefile]")
		fmt.Println("For further instruction use", runningProgram, " --help")
		os.Exit(0)
	}

	// We only worry about the first parameter it could be a filename or a --help request.
	filename := strings.Trim(os.Args[1], " ")

	if filename == "--help" || filename == "-help" {
		fmt.Println(getHelp(runningProgram))
		os.Exit(0)
	}

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

	// As we are just reading in the file - there is no need to close it,  it is not opened for writing
	fileBytes, err := ioutil.ReadFile(logfile)
	if err != nil {
		// We have already checked for a file existance so at this state and error would be that it cannot be read
		fmt.Println("File to be processed could not be read")
		fmt.Println(err)
		os.Exit(0)
	}

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

// This function prioritises ease of code review over succinctness - so at the end.
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
		// If no match, it is most likely in the long form already.
		return shortNetmask
	}

}

func getHelp(runningProgram string) string {

	helpMessage := "Route2NM Help \n"
	helpMessage += " \n"
	helpMessage += " Route2NM is a small utility used to convert the older network style routes to the newer \n"
	helpMessage += " Network Manager Format.  This utility is of most use when upgrading servers from \n"
	helpMessage += " servers using the non network manger format such as RHEL/CentOS 5/6 to RHEL/CentOS 7 \n"
	helpMessage += " \n"
	helpMessage += "Usage: " + runningProgram + " [routefile]     Convert Older Route File to Newer Network Manager Format\n"
	helpMessage += " \n"
	helpMessage += " The older route file will be in the format of:\n"
	helpMessage += "   192.168.1.100/24 via 192.168.151.1 dev eth0 \n"
	helpMessage += " \n"
	helpMessage += " And will convert the route to the newer format of:\n"
	helpMessage += "   ADDRESS10=192.168.1.100 \n"
	helpMessage += "   NETMASK10=255.255.255.0 \n"
	helpMessage += "   GATEWAY10=192.168.151.1 \n"
	helpMessage += "   METRIC10=0 \n"
	helpMessage += " \n"
	helpMessage += "Parameters \n"
	helpMessage += "--help:  This Help display \n"
	helpMessage += " \n"
	return helpMessage
}
