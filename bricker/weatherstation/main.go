// Copyright 2014 Dirk Jablonowski. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
weatherstation is a example how to program the weatherstation kit in go (golang).

The idea is, that this example application searches for the bricklets and print
out their informations. A LCD 20x4 Bricklet is needed for printing out something.

The program could take two parameters.
One for the connection address. It defaults to localhost:4223.
With the other parameter you could trigger that the output will be printed to
the console too. Default is false (no output on console).

You need the bricker api code.
  go get github.com/dirkjabl/bricker
*/
package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/dirkjabl/bricker"
	"github.com/dirkjabl/bricker/connector/buffered"
	"github.com/dirkjabl/bricker/device"
	"github.com/dirkjabl/bricker/device/bricklet/ambientlight"
	"github.com/dirkjabl/bricker/device/bricklet/barometer"
	"github.com/dirkjabl/bricker/device/bricklet/humidity"
	"github.com/dirkjabl/bricker/device/bricklet/lcd20x4"
	"github.com/dirkjabl/bricker/device/bricklet/temperature"
	"github.com/dirkjabl/bricker/device/enumerate"
	"github.com/dirkjabl/bricker/util/ks0066"
	"os"
)

const (
	period            = uint32(1000) // default callback period in ms
	cn                = "ws"         // connectorname
	bl_lcd            = uint16(212)  // LCD bricklet device identifer
	bl_humidity       = uint16(27)   // Humidity bricklet device identifer
	bl_barometer      = uint16(221)  // Barometer bricklet device identifer
	bl_ambientlight   = uint16(21)   // Ambient Light bricklet device identifer
	bl_temperature    = uint16(216)  // Temperature bricklet device identifer
	brickSubscribed   = uint8(1)     // Flag for bricklet is subscribed
	brickUnsubscribed = uint8(2)     // Flag for bricklet is unsubscribed
	brickUnchange     = uint8(0)     // Flag for bricklet state is unchanged
)

// bricklet type for remember important data
type bricklet struct {
	has bool           // if the bricklet exists
	sub *device.Device // subscriber
	uid uint32         // uid
	cb  func()         // handler for this bricklet type
}

// Data structur, remembers which bricklet exists, what for a address to use and the bricker.
var conf struct {
	addr          string               // address of the stack
	brick         *bricker.Bricker     // bricker
	showOnConsole bool                 // show output from the LCD on the console, too
	bricklets     map[uint16]*bricklet // Map with all supportet bricklets
}

// main routine, will startup.
func main() {
	// need the address of the stack with the weatherstation kit bricklets.
	var addr = flag.String("addr", "localhost:4223",
		"address of the brickd, default is localhost:4223")
	var soc = flag.Bool("console", false,
		"show output from lcd on the console, too: default false")
	flag.Parse()

	// create map for the bricklets
	// conf.bricklets = make(map[uint16]*bricklet)
	conf.bricklets = map[uint16]*bricklet{
		bl_lcd:          &bricklet{has: false, cb: workLcd},          // LCD 20x4
		bl_humidity:     &bricklet{has: false, cb: workHumidity},     // humidity
		bl_barometer:    &bricklet{has: false, cb: workBarometer},    // barometer
		bl_ambientlight: &bricklet{has: false, cb: workAmbientlight}, // ambient light
		bl_temperature:  &bricklet{has: false, cb: workTemp},         // temperature
	}

	// remember the flags
	conf.addr = *addr
	conf.showOnConsole = *soc

	// Create a bricker object
	conf.brick = bricker.New()
	defer conf.brick.Done() // later for stopping the bricker

	// create a connection to a real brick stack
	conn, err := buffered.New(conf.addr, 20, 10)
	if err != nil { // no connection
		fmt.Printf("No connection: %s\n", err.Error())
		return
	}
	defer conn.Done() // later for stopping current connection

	// attach the connector to the bricker
	err = conf.brick.Attach(conn, cn) // ws is the name for this connection
	if err != nil {                   // no bricker, no fun
		fmt.Printf("Could not attach connection to bricker: %s\n", err.Error())
		return
	}
	defer conf.brick.Release(cn) // later to release connection from bricker

	// Look out for the hardware(bricklets) inside the given stack
	hw := make(chan *enumerate.Enumeration, 4)
	en := enumerate.Enumerate("Enumerate", false,
		func(r device.Resulter, err error) {
			if err == nil && r != nil { // only if no error occur
				if v, ok := r.(*enumerate.Enumeration); ok {
					hw <- v
				}
			}
		})
	go hardwareidentify(hw)

	// attach enumeration subscriber to the bricker
	err = conf.brick.Subscribe(en, cn)

	// go on with the program, waiting for a key
	fmt.Printf("Press return for stop.\n")
	_, _ = bufio.NewReader(os.Stdin).ReadByte()
	if conf.bricklets[bl_lcd].has {
		_ = lcd20x4.BacklightOffFuture(conf.brick, cn, conf.bricklets[bl_lcd].uid)
	}
}

// This handler identify the founded hardware and if possible
// it starts or stops a handler/callback to read out the sensors or to display.
func hardwareidentify(c chan *enumerate.Enumeration) {
	for {
		value := <-c
		var uid uint32
		if value.EnumerationType != enumerate.EnumerationTypeDisconneted {
			// exists and is active
			uid = value.IntUid()
		} else {
			uid = 0
		}
		if bl, ok := conf.bricklets[value.DeviceIdentifer]; ok {
			bl.uid = uid
			bl.cb()
		}
	}
}

// workOnBricklet subscribe or unsubscribe a bricklet handler.
func workOnBricklet(bl *bricklet) uint8 {
	result := brickUnchange
	if bl.uid > 0 {
		if !bl.has {
			bl.has = true
			_ = conf.brick.Subscribe(bl.sub, cn)
			result = brickSubscribed
		}
	} else {
		if bl.has {
			bl.has = false
			_ = conf.brick.Unsubscribe(bl.sub)
			result = brickUnsubscribed
		}
	}
	return result
}

// workLcd register or unregister the needed Subscriber for the LCD.
func workLcd() {
	bl := conf.bricklets[bl_lcd]
	if bl.sub == nil {
		// any button pressed
		bl.sub = lcd20x4.ButtonPressed("Button", bl.uid, handler)
	}
	s := workOnBricklet(bl)
	if s == brickSubscribed {
		// clear the display first
		_ = lcd20x4.ClearDisplayFuture(conf.brick, cn, bl.uid)
		// backlight on
		_ = lcd20x4.BacklightOnFuture(conf.brick, cn, bl.uid)
	}
}

// workHumidity register or unregister the needed Subscriber for the output of the humidity value.
func workHumidity() {
	bl := conf.bricklets[bl_humidity]
	if bl.sub == nil {
		// set handler for the humidity callback (event handler)
		bl.sub = humidity.HumidityPeriod("Humidity", bl.uid, handler)
	}
	s := workOnBricklet(bl)
	if s == brickSubscribed {
		// setting the period for calling the humidity callback
		_ = humidity.SetHumidityCallbackPeriodFuture(conf.brick, cn, bl.uid,
			&device.Period{Value: period})
	} else if s == brickUnsubscribed {
		// unset the period (set to 0) for calling the humidity callback
		_ = humidity.SetHumidityCallbackPeriodFuture(conf.brick, cn, bl.uid,
			&device.Period{Value: 0})
	}
}

// workAmbientlight register or unregister the needed Subscriber for the output of the illuminance value.
func workAmbientlight() {
	bl := conf.bricklets[bl_ambientlight]
	if bl.sub == nil {
		// set handler for the ambient light callback (event handler)
		bl.sub = ambientlight.IlluminancePeriod("Illuminance", bl.uid, handler)
	}
	s := workOnBricklet(bl)
	if s == brickSubscribed {
		// setting the period for calling the ambient light callback
		_ = ambientlight.SetIlluminanceCallbackPeriodFuture(conf.brick, cn, bl.uid,
			&device.Period{Value: period})
	} else if s == brickUnsubscribed {
		// unset the period (set to 0) for calling the ambient light callback
		_ = ambientlight.SetIlluminanceCallbackPeriodFuture(conf.brick, cn, bl.uid,
			&device.Period{Value: 0})
	}
}

// workBarometer register or unregister the needed Subscriber for the output
// of the air pressure value.
func workBarometer() {
	bl := conf.bricklets[bl_barometer]
	if bl.sub == nil {
		bl.sub = barometer.AirPressurePeriod("AirPressure", bl.uid, handler)
	}
	s := workOnBricklet(bl)
	if s == brickSubscribed {
		_ = barometer.SetAirPressureCallbackPeriodFuture(conf.brick, cn, bl.uid,
			&device.Period{Value: period})
	} else if s == brickUnsubscribed {
		_ = barometer.SetAirPressureCallbackPeriodFuture(conf.brick, cn, bl.uid,
			&device.Period{Value: 0})
	}
}

// workTemp register or unregister the needed Subscriber for the output of the temperature value.
func workTemp() {
	bl := conf.bricklets[bl_temperature]
	if bl.sub == nil {
		bl.sub = temperature.TemperaturePeriod("Temperature", bl.uid, handler)
	}
	s := workOnBricklet(bl)
	if s == brickSubscribed {
		temperature.SetTemperatureCallbackPeriodFuture(conf.brick, cn, bl.uid,
			&device.Period{Value: period})
	} else if s == brickUnsubscribed {
		temperature.SetTemperatureCallbackPeriodFuture(conf.brick, cn, bl.uid,
			&device.Period{Value: 0})
	}
}

// handler is the default handler for output the values
func handler(r device.Resulter, err error) {
	if err == nil && r != nil {
		go workOnResult(r)
	}
}

// workOnResult is the base routine to work on every sensor data input (event, callback, ...)
func workOnResult(r device.Resulter) {
	if r == nil { // no data, no work
		return
	}
	var line uint8
	var txt string = ""
	switch v := r.(type) { // type switch
	default:
		// do nothing
	case *humidity.Humidity:
		line = uint8(1)
		txt = fmt.Sprintf("Hum.: %6.2f %%RH      ", v.Float64())
	case *ambientlight.Illuminance:
		line = uint8(3)
		txt = fmt.Sprintf("Ill.: %6.2f lx", v.Float64())
	case *barometer.AirPressure:
		line = uint8(2)
		txt = fmt.Sprintf("Air.: %7.2f mbar", v.Float64())
		if !conf.bricklets[bl_temperature].has {
			sub := barometer.GetChipTemperature("", conf.bricklets[bl_barometer].uid, handler)
			_ = conf.brick.Subscribe(sub, cn)
		}
	case *barometer.Temperature:
		line = uint8(0)
		txt = fmt.Sprintf("Tem.: %5.2f °C", v.Float64())
	case *temperature.Temperature:
		line = uint8(0)
		txt = fmt.Sprintf("Tem.: %5.2f °C", v.Float64())
	case *lcd20x4.Button:
		// toggle backlight of the LCD on button press
		bl := lcd20x4.IsBacklightOnFuture(conf.brick, cn, conf.bricklets[bl_lcd].uid)
		if bl != nil {
			if bl.IsOn {
				_ = lcd20x4.BacklightOffFuture(conf.brick, cn, conf.bricklets[bl_lcd].uid)
			} else {
				_ = lcd20x4.BacklightOnFuture(conf.brick, cn, conf.bricklets[bl_lcd].uid)
			}
		}
	}
	if txt != "" {
		if conf.bricklets[bl_lcd].has { // could only output with LCD
			ltl := ks0066.NewLcdTextLine(line, uint8(0), txt)
			sub := lcd20x4.WriteLine("", conf.bricklets[bl_lcd].uid, ltl,
				func(r device.Resulter, err error) {
					if err != nil {
						fmt.Printf("Error by writing line: %s\n", r) // output simple on terminal/CLI
					}
				})
			_ = conf.brick.Subscribe(sub, cn)
		}
		if conf.showOnConsole { // output simple on terminal/CLI
			fmt.Printf("%s\n", txt)
		}
	}
}
