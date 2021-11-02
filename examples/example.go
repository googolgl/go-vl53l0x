package main

import (
	"context"
	"os"
	"syscall"
	"vl53l0x"

	shell "github.com/d2r2/go-shell"

	i2cDev "github.com/googolgl/go-i2c"
)

func main() {
	// Create new connection to i2c-bus on 1 line with address 0x40.
	// Use i2cdetect utility to find device address over the i2c-bus
	i2c, err := i2cDev.New(0x29, "/dev/i2c-0")
	if err != nil {
		i2c.Log.Fatal(err)
	}
	defer i2c.Close()

	i2c.Log.Infoln("**********************************************************************************************")
	i2c.Log.Infoln("*** !!! READ THIS !!!")
	i2c.Log.Infoln("*** You can change verbosity of output, by modifying logging level of modules \"i2c\", \"vl53l0x\".")
	i2c.Log.Infoln("*** Uncomment/comment corresponding lines with call to ChangePackageLogLevel(...)")
	i2c.Log.Infoln("*** !!! READ THIS !!!")
	i2c.Log.Infoln("**********************************************************************************************")
	// Uncomment/comment next line to suppress/increase verbosity of output
	//logger.ChangePackageLogLevel("i2c", logger.InfoLevel)
	//logger.ChangePackageLogLevel("vl53l0x", logger.InfoLevel)

	sensor := vl53l0x.New(i2c)
	i2c.Log.Infoln("**********************************************************************************************")
	i2c.Log.Infoln("*** Reset/initialize sensor")
	i2c.Log.Infoln("**********************************************************************************************")
	err = sensor.Reset()
	if err != nil {
		i2c.Log.Fatalf("Error reseting sensor: %s", err)
	}

	if err := sensor.SetAddress(0x2a); err != nil {
		i2c.Log.Fatal(err)
	}
	i2c.Close()

	i2c2, err := i2cDev.New(0x2a, "/dev/i2c-0")
	if err != nil {
		i2c2.Log.Fatal(err)
	}
	defer i2c2.Close()

	sensor = vl53l0x.New(i2c2)

	// It's highly recommended to reset sensor before repeated initialization.
	// By default, sensor initialized with "RegularRange" and "RegularAccuracy" parameters.
	err = sensor.Init()
	if err != nil {
		i2c.Log.Fatalf("Failed to initialize sensor: %s", err)
	}
	rev, err := sensor.GetProductMinorRevision()
	if err != nil {
		i2c.Log.Fatalf("Error getting sensor minor revision: %s", err)
	}
	i2c.Log.Infof("Sensor minor revision = %d", rev)

	i2c.Log.Infoln("**********************************************************************************************")
	i2c.Log.Infoln("*** Сonfigure sensor")
	i2c.Log.Infoln("**********************************************************************************************")
	rngConfig := vl53l0x.RegularRange
	speedConfig := vl53l0x.HighAccuracy
	i2c.Log.Infof("Configure sensor with  %q and %q", rngConfig, speedConfig)
	err = sensor.Config(rngConfig, speedConfig)
	if err != nil {
		i2c.Log.Fatalf("Failed to initialize sensor: %s", err)
	}

	i2c.Log.Infoln("**********************************************************************************************")
	i2c.Log.Infoln("*** Single shot range measurement mode")
	i2c.Log.Infoln("**********************************************************************************************")
	rng, err := sensor.ReadRangeSingleMillimeters()
	if err != nil {
		i2c.Log.Fatalf("Failed to measure range: %s", err)
	}
	i2c.Log.Infof("Measured range = %v mm", rng)

	i2c.Log.Infoln("**********************************************************************************************")
	i2c.Log.Infoln("*** Change default address")
	i2c.Log.Infoln("**********************************************************************************************")

	/*if err := sensor.SetAddress(0x2a); err != nil {
		i2c.Log.Fatal(err)
	}*/

	i2c.Log.Infoln("**********************************************************************************************")
	i2c.Log.Infoln("*** Continuous shot range measurement mode")
	i2c.Log.Infoln("**********************************************************************************************")
	var freq uint32 = 20
	times := 50
	i2c.Log.Infof("Made measurement each %d milliseconds, %d times", freq, times)
	err = sensor.StartContinuous(freq)
	if err != nil {
		i2c.Log.Fatalf("Can't start continuous measures: %s", err)
	}
	// create context with cancellation possibility
	ctx, cancel := context.WithCancel(context.Background())
	// use done channel as a trigger to exit from signal waiting goroutine
	done := make(chan struct{})
	defer close(done)
	// build actual signals list to control
	signals := []os.Signal{os.Kill, os.Interrupt}
	if shell.IsLinuxMacOSFreeBSD() {
		signals = append(signals, syscall.SIGTERM)
	}
	// run goroutine waiting for OS termination events, including keyboard Ctrl+C
	shell.CloseContextOnSignals(cancel, done, signals...)

	for i := 0; i < times; i++ {
		rng, err = sensor.ReadRangeContinuousMillimeters()
		if err != nil {
			i2c.Log.Fatalf("Failed to measure range: %s", err)
		}
		i2c.Log.Infof("Measured range = %v mm", rng)
		select {
		// Check for termination request.
		case <-ctx.Done():
			err = sensor.StopContinuous(i2c)
			if err != nil {
				i2c.Log.Fatal(err)
			}
			i2c.Log.Fatal(ctx.Err())
		default:
		}
	}
	err = sensor.StopContinuous(i2c)
	if err != nil {
		i2c.Log.Fatalf("Error stopping continuous measures: %s", err)
	}

	i2c.Log.Infoln("**********************************************************************************************")
	i2c.Log.Infoln("*** Reconfigure sensor")
	i2c.Log.Infoln("**********************************************************************************************")
	rngConfig = vl53l0x.RegularRange
	speedConfig = vl53l0x.RegularAccuracy
	i2c.Log.Infof("Reconfigure sensor with %q and %q", rngConfig, speedConfig)
	err = sensor.Config(rngConfig, speedConfig)
	if err != nil {
		i2c.Log.Fatalf("Failed to initialize sensor: %s", err)
	}

	i2c.Log.Infoln("**********************************************************************************************")
	i2c.Log.Infoln("*** Single shot range measurement mode")
	i2c.Log.Infoln("**********************************************************************************************")
	rng, err = sensor.ReadRangeSingleMillimeters()
	if err != nil {
		i2c.Log.Fatalf("Failed to measure range: %s", err)
	}
	i2c.Log.Infof("Measured range = %v mm", rng)

}
