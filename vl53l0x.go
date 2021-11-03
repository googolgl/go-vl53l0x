//--------------------------------------------------------------------------------------------------
// This code is an adaptation and translation of well-formed C++ library to GOLANG for the distance
// measure sensor's family VL53L0X taken from https://github.com/pololu/vl53l0x-arduino:
//      https://github.com/pololu/vl53l0x-arduino/blob/master/VL53L0X.cpp
//      https://github.com/pololu/vl53l0x-arduino/blob/master/VL53L0X.h
//
// Some portion of code taken from Adafruit https://github.com/adafruit/Adafruit_VL53L0X:
//      https://github.com/adafruit/Adafruit_VL53L0X/blob/master/src/core/src/vl53l0x_api.cpp
//      https://github.com/adafruit/Adafruit_VL53L0X/blob/master/src/vl53l0x_def.h
//      https://github.com/adafruit/Adafruit_VL53L0X/blob/master/src/vl53l0x_device.h
//--------------------------------------------------------------------------------------------------

package vl53l0x

import (
	"errors"
	"fmt"
	"time"

	"github.com/googolgl/go-i2c"
)

// VcselPeriodType is a type of VCSEL (vertical cavity surface emitting laser) pulse period.
type VcselPeriodType int

const (
	// pre-range pulse period
	VcselPeriodPreRange VcselPeriodType = iota + 1
	// final range pulse period
	VcselPeriodFinalRange
)

// RangeSpec used to configure sensor for expected distance to measure.
type RangeSpec int

const (
	// Signal rate limit = 0.25 MCPS, laser pulse periods = (14, 10).
	RegularRange RangeSpec = iota + 1
	// Signal rate limit = 0.10 MCPS, laser pulse periods = (18, 14).
	// Use "long range" mode only when "regular" can't detect distance
	// (returned distance value is 8190 mm or more). It's ordinary
	// happens, when distance exceed something about a meter.
	LongRange
)

// String implement Stringer interface.
func (v RangeSpec) String() string {
	switch v {
	case RegularRange:
		return "RegularRange"
	case LongRange:
		return "LongRange"
	default:
		return "<unknown>"
	}
}

// SpeedAccuracySpec used to configure sensor for accuracy/measure time.
// It's clear that to improve accuracy, you should increase
// measure time.
type SpeedAccuracySpec int

const (
	// HighSpeed distance measurement takes 20 ms.
	HighSpeed SpeedAccuracySpec = iota + 1
	// RegularAccuracy distance measurement takes 33 ms.
	RegularAccuracy
	// GoodAccuracy distance measurement takes 66 ms.
	GoodAccuracy
	// HighAccuracy distance measurement takes 100 ms.
	HighAccuracy
	// HighestAccuracy distance measurement takes 200 ms.
	HighestAccuracy
)

// String implement Stringer interface.
func (v SpeedAccuracySpec) String() string {
	switch v {
	case HighSpeed:
		return "HighSpeed"
	case RegularAccuracy:
		return "RegularAccuracy"
	case GoodAccuracy:
		return "GoodAccuracy"
	case HighAccuracy:
		return "HighAccuracy"
	case HighestAccuracy:
		return "HighestAccuracy"
	default:
		return "<unknown>"
	}
}

// Entity contains sensor data and corresponding methods.
type Entity struct {
	// read by init and used when starting measurement;
	// is StopVariable field of VL53L0X_DevData_t structure in API
	stopVariable uint8
	// total measurement timing budget in microseconds
	measurementTimingBudgetUsec uint32
	// default timeout value
	ioTimeout time.Duration
	i2c       *i2c.Options
}

// New creates sensor instance.
func New(i2c *i2c.Options) *Entity {
	return &Entity{
		i2c: i2c,
	}
}

// Config configure sensor expected distance range and time to make a measurement.
func (e *Entity) Config(rng RangeSpec, speed SpeedAccuracySpec) error {

	e.i2c.Log.Debug("Start config")

	switch rng {
	case RegularRange:
		// default is 0.25 MCPS
		err := e.SetSignalRateLimit(0.25)
		if err != nil {
			return err
		}
		// defaults are 14 and 10 PCLKs)
		err = e.SetVcselPulsePeriod(VcselPeriodPreRange, 14)
		if err != nil {
			return err
		}
		err = e.SetVcselPulsePeriod(VcselPeriodFinalRange, 10)
		if err != nil {
			return err
		}

	case LongRange:
		// lower the return signal rate limit (default is 0.25 MCPS)
		err := e.SetSignalRateLimit(0.1)
		if err != nil {
			return err
		}
		// increase laser pulse periods (defaults are 14 and 10 PCLKs)
		err = e.SetVcselPulsePeriod(VcselPeriodPreRange, 18)
		if err != nil {
			return err
		}
		err = e.SetVcselPulsePeriod(VcselPeriodFinalRange, 14)
		if err != nil {
			return err
		}
	}

	switch speed {
	case HighSpeed:
		// reduce timing budget to 20 ms (default is about 33 ms)
		err := e.SetMeasurementTimingBudget(20000)
		if err != nil {
			return err
		}

	case RegularAccuracy:
		// default is about 33 ms
		err := e.SetMeasurementTimingBudget(33000)
		if err != nil {
			return err
		}

	case GoodAccuracy:
		// increase timing budget to 66 ms
		err := e.SetMeasurementTimingBudget(66000)
		if err != nil {
			return err
		}

	case HighAccuracy:
		// increase timing budget to 100 ms
		err := e.SetMeasurementTimingBudget(100000)
		if err != nil {
			return err
		}

	case HighestAccuracy:
		// increase timing budget to 200 ms
		err := e.SetMeasurementTimingBudget(200000)
		if err != nil {
			return err
		}
	}

	e.i2c.Log.Debug("End config")

	return nil
}

// Reset soft-reset the sensor.
// Based on VL53L0X_ResetDevice().
func (e *Entity) Reset() error {
	e.i2c.Log.Debug("Set reset bit")
	if err := e.i2c.WriteRegU8(SOFT_RESET_GO2_SOFT_RESET_N, 0x00); err != nil {
		return err
	}

	// Wait for some time
	err := e.waitUntilOrTimeout(IDENTIFICATION_MODEL_ID,
		func(checkReg byte, err error) (bool, error) {
			return checkReg == 0, err
		})
	if err != nil {
		return err
	}

	// Release reset
	e.i2c.Log.Debug("Release reset bit")
	err = e.i2c.WriteRegU8(SOFT_RESET_GO2_SOFT_RESET_N, 0x01)
	if err != nil {
		return err
	}

	// Wait for some time
	err = e.waitUntilOrTimeout(IDENTIFICATION_MODEL_ID,
		func(checkReg byte, err error) (bool, error) {
			// Skip error like "read /dev/i2c-x: no such device or address"
			// for a while, because sensor in reboot has temporary
			// no connection to I2C-bus. So, that is why we are
			// returning nil instead of err, suppressing this.
			return checkReg != 0, nil
		})
	if err != nil {
		return err
	}
	return nil
}

// GetProductMinorRevision takes revision from sensor hardware.
// Based on VL53L0X_GetProductRevision.
func (e *Entity) GetProductMinorRevision() (byte, error) {
	u8, err := e.i2c.ReadRegU8(IDENTIFICATION_REVISION_ID)
	if err != nil {
		return 0, err
	}
	return (u8 & 0xF0) >> 4, nil
}

// SetAddress change default address of sensor and reopen I2C-connection.
//func (e *Entity) SetAddress(i2cRef **i2c.Options, newAddr byte) error {
func (e *Entity) SetAddress(newAddr byte) error {
	err := e.i2c.WriteRegU8(I2C_SLAVE_DEVICE_ADDRESS, newAddr&0x7F)
	if err != nil {
		return err
	}
	//*i2cRef, err = i2c.New(newAddr, (*i2cRef).GetDev())
	return err
}

// Init initialize sensor using sequence based on VL53L0X_DataInit(),
// VL53L0X_StaticInit(), and VL53L0X_PerformRefCalibration().
// This function does not perform reference SPAD calibration
// (VL53L0X_PerformRefSpadManagement()), since the API user manual says that it
// is performed by ST on the bare modules; it seems like that should work well
// enough unless a cover glass is added.
func (e *Entity) Init() error {

	e.setTimeout(time.Millisecond * 1000)

	// VL53L0X_DataInit() begin

	// "Set I2C standard mode"
	err := e.i2c.WriteRegU8(0x88, 0x00)
	if err != nil {
		return err
	}

	err = e.writeRegValues([]RegBytePair{
		{Reg: 0x80, Value: 0x01},
		{Reg: 0xFF, Value: 0x01},
		{Reg: 0x00, Value: 0x00},
	}...)
	if err != nil {
		return err
	}

	e.stopVariable, err = e.i2c.ReadRegU8(0x91)
	if err != nil {
		return err
	}
	err = e.writeRegValues([]RegBytePair{
		{Reg: 0x00, Value: 0x01},
		{Reg: 0xFF, Value: 0x00},
		{Reg: 0x80, Value: 0x00},
	}...)
	if err != nil {
		return err
	}

	// disable SIGNAL_RATE_MSRC (bit 1) and SIGNAL_RATE_PRE_RANGE (bit 4) limit checks
	u8, err := e.i2c.ReadRegU8(MSRC_CONFIG_CONTROL)
	if err != nil {
		return err
	}

	err = e.i2c.WriteRegU8(MSRC_CONFIG_CONTROL, u8|0x12)
	if err != nil {
		return err
	}

	// set final range signal rate limit to 0.25 MCPS (million counts per second)
	err = e.SetSignalRateLimit(0.25)
	if err != nil {
		return err
	}

	err = e.i2c.WriteRegU8(SYSTEM_SEQUENCE_CONFIG, 0xFF)
	if err != nil {
		return err
	}

	// VL53L0X_DataInit() end

	// VL53L0X_StaticInit() begin

	spadInfo, err := e.getSpadInfo()
	if err != nil {
		return err
	}

	// The SPAD map (RefGoodSpadMap) is read by VL53L0X_get_info_from_device() in
	// the API, but the same data seems to be more easily readable from
	// GLOBAL_CONFIG_SPAD_ENABLES_REF_0 through _6, so read it from there
	spadMap := make([]byte, 6)
	err = e.readRegBytes(GLOBAL_CONFIG_SPAD_ENABLES_REF_0, spadMap)
	//err = v.i2c.ReadBytes(GLOBAL_CONFIG_SPAD_ENABLES_REF_0, spadMap)
	if err != nil {
		return err
	}

	// -- VL53L0X_set_reference_spads() begin (assume NVM values are valid)

	err = e.writeRegValues([]RegBytePair{
		{Reg: 0xFF, Value: 0x01},
		{Reg: DYNAMIC_SPAD_REF_EN_START_OFFSET, Value: 0x00},
		{Reg: DYNAMIC_SPAD_NUM_REQUESTED_REF_SPAD, Value: 0x2C},
		{Reg: 0xFF, Value: 0x00},
		{Reg: GLOBAL_CONFIG_REF_EN_START_SELECT, Value: 0xB4},
	}...)
	if err != nil {
		return err
	}

	var firstSpadToEnable byte
	if spadInfo.TypeIsAperture {
		// 12 is the first aperture spad
		firstSpadToEnable = 12
	}
	var spadsEnabled byte

	var i byte
	for i = 0; i < 48; i++ {
		if i < firstSpadToEnable || spadsEnabled == spadInfo.Count {
			// This bit is lower than the first one that should be enabled, or
			// (reference_spad_count) bits have already been enabled, so zero this bit
			spadMap[i/8] &= ^(1 << (i % 8))
		} else if (spadMap[i/8]>>(i%8))&0x1 != 0 {
			spadsEnabled++
		}
	}

	err = e.writeBytes(GLOBAL_CONFIG_SPAD_ENABLES_REF_0, spadMap)
	if err != nil {
		return err
	}

	// -- VL53L0X_set_reference_spads() end

	// -- VL53L0X_load_tuning_settings() begin
	// DefaultTuningSettings from vl53l0x_tuning.h

	err = e.writeRegValues([]RegBytePair{
		{Reg: 0xFF, Value: 0x01},
		{Reg: 0x00, Value: 0x00},
	}...)
	if err != nil {
		return err
	}

	err = e.writeRegValues([]RegBytePair{
		{Reg: 0xFF, Value: 0x00},
		{Reg: 0x09, Value: 0x00},
		{Reg: 0x10, Value: 0x00},
		{Reg: 0x11, Value: 0x00},
	}...)
	if err != nil {
		return err
	}

	err = e.writeRegValues([]RegBytePair{
		{Reg: 0x24, Value: 0x01},
		{Reg: 0x25, Value: 0xFF},
		{Reg: 0x75, Value: 0x00},
	}...)
	if err != nil {
		return err
	}

	err = e.writeRegValues([]RegBytePair{
		{Reg: 0xFF, Value: 0x01},
		{Reg: 0x4E, Value: 0x2C},
		{Reg: 0x48, Value: 0x00},
		{Reg: 0x30, Value: 0x20},
	}...)
	if err != nil {
		return err
	}

	err = e.writeRegValues([]RegBytePair{
		{Reg: 0xFF, Value: 0x00},
		{Reg: 0x30, Value: 0x09},
		{Reg: 0x54, Value: 0x00},
		{Reg: 0x31, Value: 0x04},
		{Reg: 0x32, Value: 0x03},
		{Reg: 0x40, Value: 0x83},
		{Reg: 0x46, Value: 0x25},
		{Reg: 0x60, Value: 0x00},
		{Reg: 0x27, Value: 0x00},
		{Reg: 0x50, Value: 0x06},
		{Reg: 0x51, Value: 0x00},
		{Reg: 0x52, Value: 0x96},
		{Reg: 0x56, Value: 0x08},
		{Reg: 0x57, Value: 0x30},
		{Reg: 0x61, Value: 0x00},
		{Reg: 0x62, Value: 0x00},
		{Reg: 0x64, Value: 0x00},
		{Reg: 0x65, Value: 0x00},
		{Reg: 0x66, Value: 0xA0},
	}...)
	if err != nil {
		return err
	}

	err = e.writeRegValues([]RegBytePair{
		{Reg: 0xFF, Value: 0x01},
		{Reg: 0x22, Value: 0x32},
		{Reg: 0x47, Value: 0x14},
		{Reg: 0x49, Value: 0xFF},
		{Reg: 0x4A, Value: 0x00},
	}...)
	if err != nil {
		return err
	}

	err = e.writeRegValues([]RegBytePair{
		{Reg: 0xFF, Value: 0x00},
		{Reg: 0x7A, Value: 0x0A},
		{Reg: 0x7B, Value: 0x00},
		{Reg: 0x78, Value: 0x21},
	}...)
	if err != nil {
		return err
	}

	err = e.writeRegValues([]RegBytePair{
		{Reg: 0xFF, Value: 0x01},
		{Reg: 0x23, Value: 0x34},
		{Reg: 0x42, Value: 0x00},
		{Reg: 0x44, Value: 0xFF},
		{Reg: 0x45, Value: 0x26},
		{Reg: 0x46, Value: 0x05},
		{Reg: 0x40, Value: 0x40},
		{Reg: 0x0E, Value: 0x06},
		{Reg: 0x20, Value: 0x1A},
		{Reg: 0x43, Value: 0x40},
	}...)
	if err != nil {
		return err
	}

	err = e.writeRegValues([]RegBytePair{
		{Reg: 0xFF, Value: 0x00},
		{Reg: 0x34, Value: 0x03},
		{Reg: 0x35, Value: 0x44},
	}...)
	if err != nil {
		return err
	}

	err = e.writeRegValues([]RegBytePair{
		{Reg: 0xFF, Value: 0x01},
		{Reg: 0x31, Value: 0x04},
		{Reg: 0x4B, Value: 0x09},
		{Reg: 0x4C, Value: 0x05},
		{Reg: 0x4D, Value: 0x04},
	}...)
	if err != nil {
		return err
	}

	err = e.writeRegValues([]RegBytePair{
		{Reg: 0xFF, Value: 0x00},
		{Reg: 0x44, Value: 0x00},
		{Reg: 0x45, Value: 0x20},
		{Reg: 0x47, Value: 0x08},
		{Reg: 0x48, Value: 0x28},
		{Reg: 0x67, Value: 0x00},
		{Reg: 0x70, Value: 0x04},
		{Reg: 0x71, Value: 0x01},
		{Reg: 0x72, Value: 0xFE},
		{Reg: 0x76, Value: 0x00},
		{Reg: 0x77, Value: 0x00},
	}...)
	if err != nil {
		return err
	}

	err = e.writeRegValues([]RegBytePair{
		{Reg: 0xFF, Value: 0x01},
		{Reg: 0x0D, Value: 0x01},
	}...)
	if err != nil {
		return err
	}

	err = e.writeRegValues([]RegBytePair{
		{Reg: 0xFF, Value: 0x00},
		{Reg: 0x80, Value: 0x01},
		{Reg: 0x01, Value: 0xF8},
	}...)
	if err != nil {
		return err
	}

	err = e.writeRegValues([]RegBytePair{
		{Reg: 0xFF, Value: 0x01},
		{Reg: 0x8E, Value: 0x01},
		{Reg: 0x00, Value: 0x01},
		{Reg: 0xFF, Value: 0x00},
		{Reg: 0x80, Value: 0x00},
	}...)
	if err != nil {
		return err
	}

	// -- VL53L0X_load_tuning_settings() end

	// "Set interrupt config to new sample ready"
	// -- VL53L0X_SetGpioConfig() begin

	err = e.i2c.WriteRegU8(SYSTEM_INTERRUPT_CONFIG_GPIO, 0x04)
	if err != nil {
		return err
	}
	u8, err = e.i2c.ReadRegU8(GPIO_HV_MUX_ACTIVE_HIGH)
	if err != nil {
		return err
	}
	err = e.writeRegValues([]RegBytePair{
		{Reg: GPIO_HV_MUX_ACTIVE_HIGH, Value: u8 & ^byte(0x10)}, // active low
		{Reg: SYSTEM_INTERRUPT_CLEAR, Value: 0x01},
	}...)
	if err != nil {
		return err
	}

	// -- VL53L0X_SetGpioConfig() end

	u32, err := e.getMeasurementTimingBudget()
	if err != nil {
		return err
	}
	e.measurementTimingBudgetUsec = u32

	// "Disable MSRC and TCC by default"
	// MSRC = Minimum Signal Rate Check
	// TCC = Target CentreCheck
	// -- VL53L0X_SetSequenceStepEnable() begin

	err = e.i2c.WriteRegU8(SYSTEM_SEQUENCE_CONFIG, 0xE8)
	if err != nil {
		return err
	}

	// -- VL53L0X_SetSequenceStepEnable() end

	// "Recalculate timing budget"
	err = e.SetMeasurementTimingBudget(e.measurementTimingBudgetUsec)
	if err != nil {
		return err
	}

	// VL53L0X_StaticInit() end

	// VL53L0X_PerformRefCalibration() begin (VL53L0X_perform_ref_calibration())

	// -- VL53L0X_perform_vhv_calibration() begin

	err = e.i2c.WriteRegU8(SYSTEM_SEQUENCE_CONFIG, 0x01)
	if err != nil {
		return err
	}
	err = e.performSingleRefCalibration(0x40)
	if err != nil {
		return err
	}

	// -- VL53L0X_perform_vhv_calibration() end

	// -- VL53L0X_perform_phase_calibration() begin

	err = e.i2c.WriteRegU8(SYSTEM_SEQUENCE_CONFIG, 0x02)
	if err != nil {
		return err
	}
	err = e.performSingleRefCalibration(0x00)
	if err != nil {
		return err
	}

	// -- VL53L0X_perform_phase_calibration() end

	// "restore the previous Sequence Config"
	err = e.i2c.WriteRegU8(SYSTEM_SEQUENCE_CONFIG, 0xE8)
	if err != nil {
		return err
	}

	// VL53L0X_PerformRefCalibration() end

	return nil
}

// SetSignalRateLimit set the return signal rate limit check value in units of MCPS
// (mega counts per second). "This represents the amplitude of the signal reflected
// from the target and detected by the device"; setting this limit presumably determines
// the minimum measurement necessary for the sensor to report a valid reading.
// Setting a lower limit increases the potential range of the sensor but also
// seems to increase the likelihood of getting an inaccurate reading because of
// unwanted reflections from objects other than the intended target.
// Defaults to 0.25 MCPS as initialized by the ST API and this library.
func (e *Entity) SetSignalRateLimit(limitMcps float32) error {
	if limitMcps < 0 || limitMcps > 511.99 {
		return errors.New("out of MCPS range")
	}
	// Q9.7 fixed point format (9 integer bits, 7 fractional bits)
	err := e.i2c.WriteRegU16BE(FINAL_RANGE_CONFIG_MIN_COUNT_RATE_RTN_LIMIT,
		uint16(limitMcps*(1<<7)))
	return err
}

// GetSignalRateLimit gets the return signal rate limit check value in MCPS.
func (e *Entity) GetSignalRateLimit() (float32, error) {
	u16, err := e.i2c.ReadRegU16BE(FINAL_RANGE_CONFIG_MIN_COUNT_RATE_RTN_LIMIT)
	if err != nil {
		return 0, err
	}
	limit := float32(u16) / (1 << 7)
	return limit, nil
}

// TCC: Target CentreCheck
// MSRC: Minimum Signal Rate Check
// DSS: Dynamic Spad Selection
type SequenceStepEnables struct {
	TCC        bool
	MSRC       bool
	DSS        bool
	PreRange   bool
	FinalRange bool
}

type SequenceStepTimeouts struct {
	PreRangeVcselPeriodPclks   uint16
	FinalRangeVcselPeriodPclks uint16

	MsrcDssTccMclks uint16
	PreRangeMclks   uint16
	FinalRangeMclks uint16

	MsrcDssTccUsec uint32
	PreRangeUsec   uint32
	FinalRangeUsec uint32
}

// Get sequence step enables.
// Based on VL53L0X_GetSequenceStepEnables().
func (e *Entity) getSequenceStepEnables() (*SequenceStepEnables, error) {

	e.i2c.Log.Debug("Start getting sequence step enables")

	sequenceConfig, err := e.i2c.ReadRegU8(SYSTEM_SEQUENCE_CONFIG)
	if err != nil {
		return nil, err
	}

	se := &SequenceStepEnables{
		TCC:        (sequenceConfig>>4)&0x1 != 0,
		DSS:        (sequenceConfig>>3)&0x1 != 0,
		MSRC:       (sequenceConfig>>2)&0x1 != 0,
		PreRange:   (sequenceConfig>>6)&0x1 != 0,
		FinalRange: (sequenceConfig>>7)&0x1 != 0,
	}
	return se, nil
}

// Decode VCSEL (vertical cavity surface emitting laser) pulse period in PCLKs
// from register value. Based on VL53L0X_decode_vcsel_period().
func (e *Entity) decodeVcselPeriod(value byte) byte {
	return (value + 1) << 1
}

// Encode VCSEL pulse period register value from period in PCLKs.
// Based on VL53L0X_encode_vcsel_period().
func (e *Entity) encodeVcselPeriod(periodPclks byte) byte {
	return periodPclks>>1 - 1
}

// Calculate macro period in *nanoseconds* from VCSEL period in PCLKs.
// Based on Entity_calc_macro_period_ps().
// PLL_period_ps = 1655; macro_period_vclks = 2304.
func (e *Entity) calcMacroPeriod(vcselPeriodPclks uint16) uint32 {
	return (uint32(vcselPeriodPclks)*2304*1655 + 500) / 1000
}

// Convert sequence step timeout from MCLKs to microseconds with given VCSEL period in PCLKs.
// Based on Entity_calc_timeout_us().
func (e *Entity) timeoutMclksToMicroseconds(timeoutPeriodMclks uint16, vcselPeriodPclks uint16) uint32 {
	macroPeriodNsec := e.calcMacroPeriod(vcselPeriodPclks)
	return (uint32(timeoutPeriodMclks)*macroPeriodNsec + macroPeriodNsec/2) / 1000
}

// Convert sequence step timeout from microseconds to MCLKs with given VCSEL period in PCLKs.
// Based on Entity_calc_timeout_mclks().
func (e *Entity) timeoutMicrosecondsToMclks(timeoutPeriodUsec uint32, vcselPeriodPclks uint16) uint32 {
	macroPeriodNsec := e.calcMacroPeriod(vcselPeriodPclks)
	return (timeoutPeriodUsec*1000 + macroPeriodNsec/2) / macroPeriodNsec
}

// SetVcselPulsePeriod set the VCSEL (vertical cavity surface emitting laser) pulse period
// for the given period type (pre-range or final range) to the given value in PCLKs.
// Longer periods seem to increase the potential range of the sensor.
// Valid values are (even numbers only):
//  pre:  12 to 18 (initialized default: 14),
//  final: 8 to 14 (initialized default: 10).
// Based on Entity_set_vcsel_pulse_period().
func (e *Entity) SetVcselPulsePeriod(tpe VcselPeriodType, periodPclks uint8) error {
	vcselPeriodReg := e.encodeVcselPeriod(periodPclks)

	enables, err := e.getSequenceStepEnables()
	if err != nil {
		return err
	}
	timeouts, err := e.getSequenceStepTimeouts(*enables)
	if err != nil {
		return err
	}

	// "Apply specific settings for the requested clock period"
	// "Re-calculate and apply timeouts, in macro periods"

	// "When the VCSEL period for the pre or final range is changed,
	// the corresponding timeout must be read from the device using
	// the current VCSEL period, then the new VCSEL period can be
	// applied. The timeout then must be written back to the device
	// using the new VCSEL period.
	//
	// For the MSRC timeout, the same applies - this timeout being
	// dependant on the pre-range vcsel period."

	if tpe == VcselPeriodPreRange {
		// "Set phase check limits"
		switch periodPclks {
		case 12:
			err := e.i2c.WriteRegU8(PRE_RANGE_CONFIG_VALID_PHASE_HIGH, 0x18)
			if err != nil {
				return err
			}
		case 14:
			err := e.i2c.WriteRegU8(PRE_RANGE_CONFIG_VALID_PHASE_HIGH, 0x30)
			if err != nil {
				return err
			}
		case 16:
			err := e.i2c.WriteRegU8(PRE_RANGE_CONFIG_VALID_PHASE_HIGH, 0x40)
			if err != nil {
				return err
			}
		case 18:
			err := e.i2c.WriteRegU8(PRE_RANGE_CONFIG_VALID_PHASE_HIGH, 0x50)
			if err != nil {
				return err
			}
		default:
			// invalid period
			return errors.New("invalid period")
		}
		err = e.i2c.WriteRegU8(PRE_RANGE_CONFIG_VALID_PHASE_LOW, 0x08)
		if err != nil {
			return err
		}

		// apply new VCSEL period
		err = e.i2c.WriteRegU8(PRE_RANGE_CONFIG_VCSEL_PERIOD, vcselPeriodReg)
		if err != nil {
			return err
		}

		// update timeouts

		// set_sequence_step_timeout() begin
		// (SequenceStepId == Entity_SEQUENCESTEP_PRE_RANGE)

		newPreRangeTimeoutMclks := e.timeoutMicrosecondsToMclks(timeouts.PreRangeUsec,
			uint16(periodPclks))

		err = e.i2c.WriteRegU16BE(PRE_RANGE_CONFIG_TIMEOUT_MACROP_HI,
			e.encodeTimeout(uint16(newPreRangeTimeoutMclks)))
		if err != nil {
			return err
		}

		// set_sequence_step_timeout() end

		// set_sequence_step_timeout() begin
		// (SequenceStepId == Entity_SEQUENCESTEP_MSRC)

		newMsrcTimeoutMclks := e.timeoutMicrosecondsToMclks(timeouts.MsrcDssTccUsec,
			uint16(periodPclks))

		if newMsrcTimeoutMclks > 256 {
			newMsrcTimeoutMclks = 255
		} else {
			newMsrcTimeoutMclks--
		}
		err = e.i2c.WriteRegU8(MSRC_CONFIG_TIMEOUT_MACROP, uint8(newMsrcTimeoutMclks))
		if err != nil {
			return err
		}

		// set_sequence_step_timeout() end
	} else if tpe == VcselPeriodFinalRange {
		switch periodPclks {
		case 8:
			err := e.writeRegValues([]RegBytePair{
				{Reg: FINAL_RANGE_CONFIG_VALID_PHASE_HIGH, Value: 0x10},
				{Reg: FINAL_RANGE_CONFIG_VALID_PHASE_LOW, Value: 0x08},
				{Reg: GLOBAL_CONFIG_VCSEL_WIDTH, Value: 0x02},
				{Reg: ALGO_PHASECAL_CONFIG_TIMEOUT, Value: 0x0C},
				{Reg: 0xFF, Value: 0x01},
				{Reg: ALGO_PHASECAL_LIM, Value: 0x30},
				{Reg: 0xFF, Value: 0x00},
			}...)
			if err != nil {
				return err
			}
		case 10:
			err := e.writeRegValues([]RegBytePair{
				{Reg: FINAL_RANGE_CONFIG_VALID_PHASE_HIGH, Value: 0x28},
				{Reg: FINAL_RANGE_CONFIG_VALID_PHASE_LOW, Value: 0x08},
				{Reg: GLOBAL_CONFIG_VCSEL_WIDTH, Value: 0x03},
				{Reg: ALGO_PHASECAL_CONFIG_TIMEOUT, Value: 0x09},
				{Reg: 0xFF, Value: 0x01},
				{Reg: ALGO_PHASECAL_LIM, Value: 0x20},
				{Reg: 0xFF, Value: 0x00},
			}...)
			if err != nil {
				return err
			}
		case 12:
			err := e.writeRegValues([]RegBytePair{
				{Reg: FINAL_RANGE_CONFIG_VALID_PHASE_HIGH, Value: 0x38},
				{Reg: FINAL_RANGE_CONFIG_VALID_PHASE_LOW, Value: 0x08},
				{Reg: GLOBAL_CONFIG_VCSEL_WIDTH, Value: 0x03},
				{Reg: ALGO_PHASECAL_CONFIG_TIMEOUT, Value: 0x08},
				{Reg: 0xFF, Value: 0x01},
				{Reg: ALGO_PHASECAL_LIM, Value: 0x20},
				{Reg: 0xFF, Value: 0x00},
			}...)
			if err != nil {
				return err
			}
		case 14:
			err := e.writeRegValues([]RegBytePair{
				{Reg: FINAL_RANGE_CONFIG_VALID_PHASE_HIGH, Value: 0x48},
				{Reg: FINAL_RANGE_CONFIG_VALID_PHASE_LOW, Value: 0x08},
				{Reg: GLOBAL_CONFIG_VCSEL_WIDTH, Value: 0x03},
				{Reg: ALGO_PHASECAL_CONFIG_TIMEOUT, Value: 0x07},
				{Reg: 0xFF, Value: 0x01},
				{Reg: ALGO_PHASECAL_LIM, Value: 0x20},
				{Reg: 0xFF, Value: 0x00},
			}...)
			if err != nil {
				return err
			}
		default:
			// invalid period
			return errors.New("invalid period")
		}

		// apply new VCSEL period
		err = e.i2c.WriteRegU8(FINAL_RANGE_CONFIG_VCSEL_PERIOD, vcselPeriodReg)
		if err != nil {
			return err
		}

		// update timeouts

		// set_sequence_step_timeout() begin
		// (SequenceStepId == Entity_SEQUENCESTEP_FINAL_RANGE)

		// "For the final range timeout, the pre-range timeout
		//  must be added. To do this both final and pre-range
		//  timeouts must be expressed in macro periods MClks
		//  because they have different vcsel periods."

		newFinalRangeTimeoutMclks := e.timeoutMicrosecondsToMclks(timeouts.FinalRangeUsec,
			uint16(periodPclks))

		if enables.PreRange {
			newFinalRangeTimeoutMclks += uint32(timeouts.PreRangeMclks)
		}

		err = e.i2c.WriteRegU16BE(FINAL_RANGE_CONFIG_TIMEOUT_MACROP_HI,
			e.encodeTimeout(uint16(newFinalRangeTimeoutMclks)))
		if err != nil {
			return err
		}

		// set_sequence_step_timeout end
	} else {
		// invalid type
		return errors.New("invalid type")
	}

	// "Finally, the timing budget must be re-applied"

	err = e.SetMeasurementTimingBudget(e.measurementTimingBudgetUsec)
	if err != nil {
		return err
	}

	// "Perform the phase calibration. This is needed after changing on vcsel period."
	// Entity_perform_phase_calibration() begin

	sequenceConfig, err := e.i2c.ReadRegU8(SYSTEM_SEQUENCE_CONFIG)
	if err != nil {
		return err
	}
	err = e.i2c.WriteRegU8(SYSTEM_SEQUENCE_CONFIG, 0x02)
	if err != nil {
		return err
	}
	err = e.performSingleRefCalibration(0x0)
	if err != nil {
		return err
	}
	err = e.i2c.WriteRegU8(SYSTEM_SEQUENCE_CONFIG, sequenceConfig)
	if err != nil {
		return err
	}

	// Entity_perform_phase_calibration() end

	return nil
}

// Get the VCSEL pulse period in PCLKs for the given period type.
// Based on Entity_get_vcsel_pulse_period().
func (e *Entity) getVcselPulsePeriod(tpe VcselPeriodType) (byte, error) {

	e.i2c.Log.Debug("Start getting VCSEL pulse period")

	switch tpe {
	case VcselPeriodPreRange:
		u8, err := e.i2c.ReadRegU8(PRE_RANGE_CONFIG_VCSEL_PERIOD)
		if err != nil {
			return 0, err
		}
		return e.decodeVcselPeriod(u8), nil
	case VcselPeriodFinalRange:
		u8, err := e.i2c.ReadRegU8(FINAL_RANGE_CONFIG_VCSEL_PERIOD)
		if err != nil {
			return 0, err
		}
		return e.decodeVcselPeriod(u8), nil
	default:
		return 0, errors.New("invalid VCSEL period type specified")
	}
}

// StartContinuous start continuous ranging measurements. If period_ms (optional) is 0 or not
// given, continuous back-to-back mode is used (the sensor takes measurements as
// often as possible); otherwise, continuous timed mode is used, with the given
// inter-measurement period in milliseconds determining how often the sensor
// takes a measurement. Based on Entity_StartMeasurement().
func (e *Entity) StartContinuous(periodMs uint32) error {

	e.i2c.Log.Debug("Start continuous")

	err := e.writeRegValues([]RegBytePair{
		{Reg: 0x80, Value: 0x01},
		{Reg: 0xFF, Value: 0x01},
		{Reg: 0x00, Value: 0x00},
		{Reg: 0x91, Value: e.stopVariable},
		{Reg: 0x00, Value: 0x01},
		{Reg: 0xFF, Value: 0x00},
		{Reg: 0x80, Value: 0x00},
	}...)
	if err != nil {
		return err
	}
	if periodMs != 0 {
		// continuous timed mode

		// Entity_SetInterMeasurementPeriodMilliSeconds() begin

		oscCalibrateVal, err := e.i2c.ReadRegU16BE(OSC_CALIBRATE_VAL)
		if err != nil {
			return err
		}

		if oscCalibrateVal != 0 {
			periodMs *= uint32(oscCalibrateVal)
		}

		err = e.i2c.WriteRegU32BE(SYSTEM_INTERMEASUREMENT_PERIOD, periodMs)
		if err != nil {
			return err
		}

		// Entity_SetInterMeasurementPeriodMilliSeconds() end

		err = e.i2c.WriteRegU8(SYSRANGE_START, 0x04) // Entity_REG_SYSRANGE_MODE_TIMED
		if err != nil {
			return err
		}
	} else {
		// continuous back-to-back mode
		err = e.i2c.WriteRegU8(SYSRANGE_START, 0x02) // Entity_REG_SYSRANGE_MODE_BACKTOBACK
		if err != nil {
			return err
		}
	}
	return nil
}

// StopContinuous stop continuous measurements.
// Based on Entity_StopMeasurement().
func (e *Entity) StopContinuous(i2c *i2c.Options) error {

	i2c.Log.Debug("Stop continuous")

	err := e.writeRegValues([]RegBytePair{
		{Reg: SYSRANGE_START, Value: 0x01}, // Entity_REG_SYSRANGE_MODE_SINGLESHOT
		{Reg: 0xFF, Value: 0x01},
		{Reg: 0x00, Value: 0x00},
		{Reg: 0x91, Value: 0x00},
		{Reg: 0x00, Value: 0x01},
		{Reg: 0xFF, Value: 0x00},
	}...)
	return err
}

// Read measured distance from the sensor.
func (e *Entity) readRangeMillimeters() (uint16, error) {

	err := e.waitUntilOrTimeout(RESULT_INTERRUPT_STATUS,
		func(checkReg byte, err error) (bool, error) {
			return checkReg&0x07 != 0, err
		})
	if err != nil {
		return 0, err
	}

	// assumptions: Linearity Corrective Gain is 1000 (default);
	// fractional ranging is not enabled
	rng, err := e.i2c.ReadRegU16BE(RESULT_RANGE_STATUS + 10)
	if err != nil {
		return 0, err
	}
	err = e.i2c.WriteRegU8(SYSTEM_INTERRUPT_CLEAR, 0x01)
	if err != nil {
		return 0, err
	}

	return rng, nil
}

// ReadRangeContinuousMillimeters returns a range reading in millimeters
// when continuous mode is active (readRangeSingleMillimeters() also calls
// this function after starting a single-shot range measurement).
func (e *Entity) ReadRangeContinuousMillimeters() (uint16, error) {

	e.i2c.Log.Debug("Read range continuous")

	return e.readRangeMillimeters()
}

// ReadRangeSingleMillimeters performs a single-shot range measurement and returns the reading in
// millimeters based on Entity_PerformSingleRangingMeasurement().
func (e *Entity) ReadRangeSingleMillimeters() (uint16, error) {

	e.i2c.Log.Debug("Read range single")

	err := e.writeRegValues([]RegBytePair{
		{Reg: 0x80, Value: 0x01},
		{Reg: 0xFF, Value: 0x01},
		{Reg: 0x00, Value: 0x00},
		{Reg: 0x91, Value: e.stopVariable},
		{Reg: 0x00, Value: 0x01},
		{Reg: 0xFF, Value: 0x00},
		{Reg: 0x80, Value: 0x00},
		{Reg: SYSRANGE_START, Value: 0x01},
	}...)
	if err != nil {
		return 0, err
	}

	// "Wait until start bit has been cleared"
	err = e.waitUntilOrTimeout(SYSRANGE_START,
		func(checkReg byte, err error) (bool, error) {
			return checkReg&0x01 == 0, err
		})
	if err != nil {
		return 0, err
	}
	return e.readRangeMillimeters()
}

// Decode sequence step timeout in MCLKs from register value
// based on Entity_decode_timeout()
// Note: the original function returned a uint32_t, but the return value is
// always stored in a uint16_t.
func (e *Entity) decodeTimeout(regVal uint16) uint16 {
	// format: "(LSByte * 2^MSByte) + 1"
	return (regVal&0x00FF)<<((regVal&0xFF00)>>8) + 1
}

// Encode sequence step timeout register value from timeout in MCLKs
// based on Entity_encode_timeout()
// Note: the original function took a uint16_t, but the argument passed to it
// is always a uint16_t.
func (e *Entity) encodeTimeout(timeoutMclks uint16) uint16 {
	// format: "(LSByte * 2^MSByte) + 1"
	var lsByte uint32
	var msByte uint16

	if timeoutMclks > 0 {
		lsByte = uint32(timeoutMclks) - 1
		for lsByte&0xFFFFFF00 > 0 {
			lsByte >>= 1
			msByte++
		}
		return msByte<<8 | uint16(lsByte&0xFF)
	}
	return 0
}

// Get sequence step timeouts
// based on get_sequence_step_timeout(),
// but gets all timeouts instead of just the requested one, and also stores
// intermediate values.
func (e *Entity) getSequenceStepTimeouts(enables SequenceStepEnables) (*SequenceStepTimeouts, error) {

	e.i2c.Log.Debug("Start getting sequence step timeouts")

	timeouts := &SequenceStepTimeouts{}

	u8, err := e.getVcselPulsePeriod(VcselPeriodPreRange)
	if err != nil {
		return nil, err
	}
	timeouts.PreRangeVcselPeriodPclks = uint16(u8)

	u8, err = e.i2c.ReadRegU8(MSRC_CONFIG_TIMEOUT_MACROP)
	if err != nil {
		return nil, err
	}
	timeouts.MsrcDssTccMclks = uint16(u8) + 1

	timeouts.MsrcDssTccUsec = e.timeoutMclksToMicroseconds(timeouts.MsrcDssTccMclks,
		timeouts.PreRangeVcselPeriodPclks)

	u16, err := e.i2c.ReadRegU16BE(PRE_RANGE_CONFIG_TIMEOUT_MACROP_HI)
	if err != nil {
		return nil, err
	}
	timeouts.PreRangeMclks = e.decodeTimeout(u16)

	timeouts.PreRangeUsec = e.timeoutMclksToMicroseconds(timeouts.PreRangeMclks,
		timeouts.PreRangeVcselPeriodPclks)

	u8, err = e.getVcselPulsePeriod(VcselPeriodFinalRange)
	if err != nil {
		return nil, err
	}
	timeouts.FinalRangeVcselPeriodPclks = uint16(u8)

	u16, err = e.i2c.ReadRegU16BE(FINAL_RANGE_CONFIG_TIMEOUT_MACROP_HI)
	if err != nil {
		return nil, err
	}
	timeouts.FinalRangeMclks = e.decodeTimeout(u16)

	if enables.PreRange {
		timeouts.FinalRangeMclks -= timeouts.PreRangeMclks
	}

	timeouts.FinalRangeUsec = e.timeoutMclksToMicroseconds(timeouts.FinalRangeMclks,
		timeouts.FinalRangeVcselPeriodPclks)

	return timeouts, nil
}

// SetMeasurementTimingBudget set the measurement timing budget in microseconds,
// which is the time allowed for one measurement; the ST API and this library take care
// of splitting the timing budget among the sub-steps in the ranging sequence. A longer timing
// budget allows for more accurate measurements. Increasing the budget by a
// factor of N decreases the range measurement standard deviation by a factor of
// sqrt(N). Defaults to about 33 milliseconds; the minimum is 20 ms.
// Based on Entity_set_measurement_timing_budget_micro_seconds().
func (e *Entity) SetMeasurementTimingBudget(budgetUsec uint32) error {
	const (
		StartOverhead      = 1320 // note that this is different than the value in get_
		EndOverhead        = 960
		MsrcOverhead       = 660
		TccOverhead        = 590
		DssOverhead        = 690
		PreRangeOverhead   = 660
		FinalRangeOverhead = 550
		MinTimingBudget    = 20000
	)

	e.i2c.Log.Debug("Start setting measurement timing budget")

	if budgetUsec < MinTimingBudget {
		return errors.New("budget is lower than minimum allowed")
	}
	var usedBudgetUsec uint32 = StartOverhead + EndOverhead

	enables, err := e.getSequenceStepEnables()
	if err != nil {
		return err
	}
	e.i2c.Log.Debugf("Sequence step enables = %#v", enables)
	timeouts, err := e.getSequenceStepTimeouts(*enables)
	if err != nil {
		return err
	}
	e.i2c.Log.Debugf("Sequence step timeouts = %#v", timeouts)

	if enables.TCC {
		usedBudgetUsec += timeouts.MsrcDssTccUsec + TccOverhead
	}

	if enables.DSS {
		usedBudgetUsec += 2 * (timeouts.MsrcDssTccUsec + DssOverhead)
	} else if enables.MSRC {
		usedBudgetUsec += timeouts.MsrcDssTccUsec + MsrcOverhead
	}

	if enables.PreRange {
		usedBudgetUsec += timeouts.PreRangeUsec + PreRangeOverhead
	}

	if enables.FinalRange {
		usedBudgetUsec += FinalRangeOverhead

		// "Note that the final range timeout is determined by the timing
		// budget and the sum of all other timeouts within the sequence.
		// If there is no room for the final range timeout, then an error
		// will be set. Otherwise the remaining time will be applied to
		// the final range."

		if usedBudgetUsec > budgetUsec {
			// "Requested timeout too big."
			return errors.New("requested timeout too big")
		}

		finalRangeTimeoutUsec := budgetUsec - usedBudgetUsec

		// set_sequence_step_timeout() begin
		// (SequenceStepId == Entity_SEQUENCESTEP_FINAL_RANGE)

		// "For the final range timeout, the pre-range timeout
		//  must be added. To do this both final and pre-range
		//  timeouts must be expressed in macro periods MClks
		//  because they have different vcsel periods."

		e.i2c.Log.Debug("set_sequence_step_timeout() begin")

		finalRangeTimeoutMclks := e.timeoutMicrosecondsToMclks(finalRangeTimeoutUsec,
			timeouts.FinalRangeVcselPeriodPclks)

		if enables.PreRange {
			finalRangeTimeoutMclks += uint32(timeouts.PreRangeMclks)
		}

		err = e.i2c.WriteRegU16BE(FINAL_RANGE_CONFIG_TIMEOUT_MACROP_HI,
			e.encodeTimeout(uint16(finalRangeTimeoutMclks)))
		if err != nil {
			return err
		}

		e.i2c.Log.Debug("set_sequence_step_timeout() end")

		// set_sequence_step_timeout() end

		e.measurementTimingBudgetUsec = budgetUsec // store for internal reuse
	}

	e.i2c.Log.Debug("End setting measurement timing budget")

	return nil
}

// Get the measurement timing budget in microseconds
// based on Entity_get_measurement_timing_budget_micro_seconds()
// in us (microseconds).
func (e *Entity) getMeasurementTimingBudget() (uint32, error) {
	const (
		StartOverhead      = 1910 // note that this is different than the value in set_
		EndOverhead        = 960
		MsrcOverhead       = 660
		TccOverhead        = 590
		DssOverhead        = 690
		PreRangeOverhead   = 660
		FinalRangeOverhead = 550
	)

	var budgetUsec uint32 = StartOverhead + EndOverhead

	enables, err := e.getSequenceStepEnables()
	if err != nil {
		return 0, err
	}
	timeouts, err := e.getSequenceStepTimeouts(*enables)
	if err != nil {
		return 0, err
	}

	if enables.TCC {
		budgetUsec += timeouts.MsrcDssTccUsec + TccOverhead
	}

	if enables.DSS {
		budgetUsec += 2 * (timeouts.MsrcDssTccUsec + DssOverhead)
	} else if enables.MSRC {
		budgetUsec += timeouts.MsrcDssTccUsec + MsrcOverhead
	}

	if enables.PreRange {
		budgetUsec += timeouts.PreRangeUsec + PreRangeOverhead
	}

	if enables.FinalRange {
		budgetUsec += timeouts.FinalRangeUsec + FinalRangeOverhead
	}

	e.measurementTimingBudgetUsec = budgetUsec // store for internal reuse

	return budgetUsec, nil
}

// SpadInfo keeps information about sensor
// SPAD (single photon avalanche diode) photodetector structure.
type SpadInfo struct {
	Count          byte
	TypeIsAperture bool
}

// Get reference SPAD (single photon avalanche diode) count and type
// based on VL53L0X_get_info_from_device(),
// but only gets reference SPAD count and type.
func (e *Entity) getSpadInfo() (*SpadInfo, error) {
	var tmp uint8

	err := e.writeRegValues([]RegBytePair{
		{Reg: 0x80, Value: 0x01},
		{Reg: 0xFF, Value: 0x01},
		{Reg: 0x00, Value: 0x00},
	}...)
	if err != nil {
		return nil, err
	}

	err = e.i2c.WriteRegU8(0xFF, 0x06)
	if err != nil {
		return nil, err
	}
	u8, err := e.i2c.ReadRegU8(0x83)
	if err != nil {
		return nil, err
	}
	err = e.writeRegValues([]RegBytePair{
		{Reg: 0x83, Value: u8 | 0x04},
		{Reg: 0xFF, Value: 0x07},
		{Reg: 0x81, Value: 0x01},
	}...)
	if err != nil {
		return nil, err
	}

	err = e.i2c.WriteRegU8(0x80, 0x01)
	if err != nil {
		return nil, err
	}

	err = e.writeRegValues([]RegBytePair{
		{Reg: 0x94, Value: 0x6b},
		{Reg: 0x83, Value: 0x00},
	}...)
	if err != nil {
		return nil, err
	}
	err = e.waitUntilOrTimeout(0x83,
		func(checkReg byte, err error) (bool, error) {
			return checkReg != 0, err
		})
	if err != nil {
		return nil, err
	}
	err = e.i2c.WriteRegU8(0x83, 0x01)
	if err != nil {
		return nil, err
	}
	tmp, err = e.i2c.ReadRegU8(0x92)
	if err != nil {
		return nil, err
	}

	si := &SpadInfo{Count: tmp & 0x7F, TypeIsAperture: (tmp>>7)&0x01 != 0}

	err = e.writeRegValues([]RegBytePair{
		{Reg: 0x81, Value: 0x00},
		{Reg: 0xFF, Value: 0x06},
	}...)
	if err != nil {
		return nil, err
	}
	u8, err = e.i2c.ReadRegU8(0x83)
	if err != nil {
		return nil, err
	}
	err = e.writeRegValues([]RegBytePair{
		{Reg: 0x83, Value: u8 & ^byte(0x04)},
		{Reg: 0xFF, Value: 0x01},
		{Reg: 0x00, Value: 0x01},
	}...)
	if err != nil {
		return nil, err
	}

	err = e.writeRegValues([]RegBytePair{
		{Reg: 0xFF, Value: 0x00},
		{Reg: 0x80, Value: 0x00},
	}...)
	if err != nil {
		return nil, err
	}

	return si, nil
}

// Based on VL53L0X_perform_single_ref_calibration().
func (e *Entity) performSingleRefCalibration(vhvInitByte uint8) error {
	err := e.i2c.WriteRegU8(SYSRANGE_START, 0x01|vhvInitByte) // VL53L0X_REG_SYSRANGE_MODE_START_STOP
	if err != nil {
		return err
	}
	err = e.waitUntilOrTimeout(RESULT_INTERRUPT_STATUS,
		func(checkReg byte, err error) (bool, error) {
			return checkReg&0x07 != 0, err
		})
	if err != nil {
		return err
	}
	err = e.writeRegValues([]RegBytePair{
		{Reg: SYSTEM_INTERRUPT_CLEAR, Value: 0x01},
		{Reg: SYSRANGE_START, Value: 0x00},
	}...)
	if err != nil {
		return err
	}
	return nil
}

// Set timeout duration for operations which could be
// terminated on timeout events.
func (e *Entity) setTimeout(timeout time.Duration) {
	e.ioTimeout = timeout
}

// Raise timeout event if execution time exceed value in Vl53l0x.ioTimeout.
func (e *Entity) checkTimeoutExpired(startTime time.Time) bool {
	left := time.Since(startTime)
	return e.ioTimeout > 0 && left > e.ioTimeout
}

// Read specific register in the loop until condition is true,
// or wait for timeout event.
func (e *Entity) waitUntilOrTimeout(reg byte, breakWhen func(chechReg byte, err error) (bool, error)) error {
	st := time.Now()
	for {
		u8, err := e.i2c.ReadRegU8(reg)
		f, err2 := breakWhen(u8, err)
		if err2 != nil {
			return err2
		} else if f {
			break
		}
		if e.checkTimeoutExpired(st) {
			return fmt.Errorf("timeout occurs; last read register 0x%x equal to 0x%x", reg, u8)
		}
	}
	return nil
}

// Write an arbitrary number of bytes from the given array to the sensor,
// starting at the given register.
func (e *Entity) writeBytes(reg byte, buf []byte) error {
	b := append([]byte{reg}, buf...)
	_, err := e.i2c.WriteBytes(b)
	return err
}

// Keeps pair of register and value to write to.
// Used as a bunch of registers which should be
// initialized with corresponding values.
type RegBytePair struct {
	Reg   byte
	Value uint8
}

// Write bunch of registers with with corresponding values.
func (e *Entity) writeRegValues(pairs ...RegBytePair) error {
	for _, pair := range pairs {
		err := e.i2c.WriteRegU8(pair.Reg, pair.Value)
		if err != nil {
			return err
		}
	}
	return nil
}

// Read an arbitrary number of bytes from the sensor, starting at the given
// register, into the given array.
func (e *Entity) readRegBytes(reg byte, dest []byte) error {
	_, err := e.i2c.WriteBytes([]byte{reg})
	if err != nil {
		return err
	}
	_, err = e.i2c.ReadBytes(dest)
	return err
}
