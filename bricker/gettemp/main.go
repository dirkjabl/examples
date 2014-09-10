// Copyright 2014 Dirk Jablonowski. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
This is a example to get the actual temperature from the Temperature Bricklet
and print it on the command line.
This is a very simple example.
It should only show the basic structure of a implementation with the bricker api.

To run this example, you should have a Temperature Bricklet and a connection to it
(USB, Ethernet, WLAN) with a running brickd.
This example does not test, if the bricklet exists.
When no bricklet exists it waits forever.

You need the bricker api code.
  go get github.com/dirkjabl/bricker
*/
package main

import (
	"flag"
	"fmt"
	"github.com/dirkjabl/bricker"
	"github.com/dirkjabl/bricker/connector/buffered"
	"github.com/dirkjabl/bricker/device/bricklet/temperature"
)

func main() {
	// flag used here to get the needed parameter
	var addr = flag.String("addr", "localhost:4223",
		"address of the brickd, default is localhost:4223")
	var uid = flag.Int("uid", 42362,
		"UId of the Temperature Bricklet, here exists no useful default")

	// Create a bricker object
	brick := bricker.New()
	defer brick.Done() // later for stopping the bricker

	// create a connection to a real brick stack
	conn, err := buffered.NewUnbuffered(*addr)
	if err != nil { // no connection, no temperature
		fmt.Printf("No connection: %s\n", err.Error())
		return
	}
	defer conn.Done() // later for stopping current connection

	// attach the connector to the bricker
	err = brick.Attach(conn, "local") // local is the name for this connection
	if err != nil {                   // no bricker, no fun
		fmt.Printf("Could not attach connection to bricker: %s\n", err.Error())
		return
	}
	defer brick.Release("local") // later to release connection from bricker

	// Call a subscriber for getting the temperature from the bricklet.
	// In this example it is the future version, so it get the temperature or nil
	// and it waits (synchron) for the notify (callback).
	temp := temperature.GetTemperatureFuture(brick, "local", uint32(*uid))
	if temp != nil { // only if a result exists, it is a pointer(!)
		fmt.Printf("Temperature: %02.02f °C\n", temp.Float64())
	}

	// if you defer not like (?)
	// brick.Release("local") // release connection from bricker
	// conn.Done() // close the connection
	// brick.Done() // stop the bricker
	// That's all folks...
}
