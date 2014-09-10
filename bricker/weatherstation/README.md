weatherstation
==============

This is only a example to print out the values the "weatherstation kit" provide.
It uses all Buttons as one and it toggles only the LCD display.

The weatherstation kit has normally following hardware:
* LCD 20x4 Bricklet
* Barometer Bricklet
* Humidity Bricklet
* Ambient Light Bricklet
* [Temperature Bricklet optional]

The temperature is readout from the Barometer Bricklet,
if no Temperature Bricklet is connected.

The application prints out all messages on the LCD Bricklet.
The lines have following informations:
1. Temperature
2. Humidity
3. Air pressure
4. Illuminance

The buttons toggle the LCD Backlight.
After startup the application turn on the backlight of the LCD bricklet.

    weatherstation [-addr=<connection>] [-console=<true/false>]

      addr: adress of the bricker stack, defaults to localhost:4223.
      console: boolean flag for print the output to the console too, default is false.
