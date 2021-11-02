VL53L0X time-of-flight ranging sensor
=====================================

[![Build Status](https://travis-ci.org/d2r2/go-vl53l0x.svg?branch=master)](https://travis-ci.org/d2r2/go-vl53l0x)
[![Go Report Card](https://goreportcard.com/badge/github.com/d2r2/go-vl53l0x)](https://goreportcard.com/report/github.com/d2r2/go-vl53l0x)
[![GoDoc](https://godoc.org/github.com/d2r2/go-vl53l0x?status.svg)](https://godoc.org/github.com/d2r2/go-vl53l0x)
[![MIT License](http://img.shields.io/badge/License-MIT-yellow.svg)](./LICENSE)

VL53L0X ([general specification](https://raw.github.com/d2r2/go-vl53l0x/master/docs/vl53l0x.pdf), [native C code API specification](https://raw.github.com/d2r2/go-vl53l0x/master/docs/en.DM00279088.pdf)) is a world’s smallest time-of-flight ranging and gesture detection sensor from STMicroelectronics. Easily integrated with Arduino and Raspberry PI via i2c communication interface:
![image](https://raw.github.com/d2r2/go-vl53l0x/master/docs/cjvl53l0xv2.jpg)

Sensor functionality is based on laser diode emission with following photodetector signal registration. Average time duration between emission and registration is a "time-of-flight", which translates to range distance.

Here is a library written in [Go programming language](https://golang.org/) for Raspberry PI and counterparts, which gives you in the output measured range value (making all necessary i2c-bus interacting and values computing).

This library is an adaptation and translation of well-formed C++ code written by www.pololu.com to Golang, taken from https://github.com/pololu/vl53l0x-arduino.


Golang usage
------------

```go
func main() {
    // Create new connection to i2c-bus on 0 line with address 0x29.
    // Use i2cdetect utility to find device address over the i2c-bus
    i2c, err := i2c.New(0x29, "/dev/i2c-0")
    if err != nil {
        log.Fatal(err)
    }
    defer i2c.Close()

    sensor := vl53l0x.New()
    // It's highly recommended to reset sensor each time before repeated initialization.
    err = sensor.Reset(i2c)
    if err != nil {
        log.Fatal(err)
    }
    // By default, sensor initialized with "RegularRange" and "RegularAccuracy" parameters.
    err = sensor.Init(i2c)
    if err != nil {
        log.Fatal(err)
    }
    rng, err := sensor.ReadRangeSingleMillimeters(i2c)
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Measured range = %v mm", rng)
```


Getting help
------------

GoDoc [documentation](http://godoc.org/github.com/d2r2/go-vl53l0x)


Installation
------------

```bash
$ go get -u github.com/d2r2/go-vl53l0x
```

Connecting Multiple Sensors
------------

I2C only allows one address-per-device so you have to make sure each I2C device has a unique address.
The default address for the VL53L0X is 0x29 but you can change this in software.
To set the new address you can do it one of two ways. During initialization, instead of calling lox.begin(),
call lox.begin(0x30) to set the address to 0x30. Or you can, later, call lox.setAddress(0x30) at any time.
The good news is its easy to change, the annoying part is each other sensor has to be in shutdown. You
can shutdown each sensor by wiring up to the XSHUT pin to a microcontroller pin. Then perform
something like this pseudo-code:
1. Reset all sensors by setting all of their XSHUT pins low for delay(10), then set all XSHUT high to bring
out of reset
2. Keep sensor #1 awake by keeping XSHUT pin high
3. Put all other sensors into shutdown by pulling XSHUT pins low
4. Initialize sensor #1 with lox.begin( new_i2c_address) Pick any number but 0x29 and it must be under
0x7F. Going with 0x30 to 0x3F is probably OK.
5. Keep sensor #1 awake, and now bring sensor #2 out of reset by setting its XSHUT pin high.
6. Initialize sensor #2 with lox.begin( new_i2c_address) Pick any number but 0x29 and whatever you
set the first sensor to
7. Repeat for each sensor, turning each one on, setting a unique address.
Note you must do this every time you turn on the power, the addresses are not permanent!


Troubleshooting
---------------

- *How to obtain fresh Golang installation to RPi device (either any RPi clone):*
If your RaspberryPI golang installation taken by default from repository is outdated, you may consider
to install actual golang manually from official Golang [site](https://golang.org/dl/). Download
tar.gz file containing armv6l in the name. Follow installation instructions.

- *How to enable I2C bus on RPi device:*
If you employ RaspberryPI, use raspi-config utility to activate i2c-bus on the OS level.
Go to "Interfacing Options" menu, to active I2C bus.
Probably you will need to reboot to load i2c kernel module.
Finally you should have device like /dev/i2c-1 present in the system.

- *How to find I2C bus allocation and device address:*
Use i2cdetect utility in format "i2cdetect -y X", where X may vary from 0 to 5 or more,
to discover address occupied by peripheral device. To install utility you should run
`apt install i2c-tools` on debian-kind system. `i2cdetect -y 1` sample output:
    ```
         0  1  2  3  4  5  6  7  8  9  a  b  c  d  e  f
    00:          -- -- -- -- -- -- -- -- -- -- -- -- --
    10: -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- --
    20: -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- --
    30: -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- --
    40: -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- --
    50: -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- --
    60: -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- --
    70: -- -- -- -- -- -- 76 --    
    ```


Contact
-------

Please use [Github issue tracker](https://github.com/d2r2/go-vl53l0x/issues) for filing bugs or feature requests.


License
-------

Go-vl53l0x is licensed under MIT License.
