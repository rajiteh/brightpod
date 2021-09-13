package blower

import (
	"fmt"
	"strings"
	"time"
)

type Blower struct {
	id               string
	firmwareVersion  float64
	firmwareRevision float64
	IsFanRunning     bool
	mode             string
	fs               float64
	fanPower         int
	rpm              int
	temperature      float64
	lastKeepAlive    time.Time
}

var BLOWER_MODES = map[int]string{
	0: "eco",
	3: "auto",
	2: "off",
	1: "on",
}

func ModeAsString(mode int) (string, error) {
	if value, ok := BLOWER_MODES[mode]; ok {
		return value, nil
	}
	return "", fmt.Errorf("Blower mode: %d does not exist", mode)
}

func ModeAsInt(mode string) (int, error) {
	for modeInt, modeStr := range BLOWER_MODES {
		if modeStr == mode {
			return modeInt, nil
		}
	}
	return 0, fmt.Errorf("Blower mode: %s does not exist", mode)
}

func New(id string, fanPower int, temperature float64, rpm int, firmwareVersion, firmwareRevision, fs float64) (*Blower, error) {
	blower := &Blower{
		id:               id,
		firmwareVersion:  firmwareVersion,
		firmwareRevision: firmwareRevision,
		fs:               fs,
	}
	if err := blower.SetFanPower(fanPower); err != nil {
		return nil, err
	}

	if err := blower.SetRPM(rpm); err != nil {
		return nil, err
	}

	if err := blower.SetTemperature(temperature); err != nil {
		return nil, err
	}

	return blower, nil
}

func (blower *Blower) GenerateStausPayload() string {
	fanRunningFlag := 0
	if blower.IsFanRunning {
		fanRunningFlag = 1
	}
	mode, _ := ModeAsInt(blower.mode) // ignore errors because we know it's already set proper
	payload := []string{
		fmt.Sprintf("%d", fanRunningFlag),
		fmt.Sprintf("%d", blower.fanPower),
		fmt.Sprintf("%d", blower.rpm),
		fmt.Sprintf("%.1f", blower.temperature),
		fmt.Sprintf("%d", mode),
	}
	return strings.Join(payload, " ")
}

func (blower *Blower) SetModeFromString(mode string) error {
	if _, err := ModeAsInt(mode); err == nil {
		blower.mode = mode
		return nil
	} else {
		return err
	}
}

func (blower *Blower) SetModeFromInt(mode int) error {
	if modeStr, err := ModeAsString(mode); err == nil {
		blower.mode = modeStr
		return nil
	} else {
		return err
	}
}

func (blower *Blower) Mode() string {
	return blower.mode
}

func (blower *Blower) SetFanPower(power int) error {
	if power < 0 || power > 12 {
		return fmt.Errorf("fan power must be postive and less than 12, recieved: %d", power)
	}
	blower.fanPower = power
	return nil
}

func (blower *Blower) SetRPM(rpm int) error {
	if rpm > 6000 {
		return fmt.Errorf("fan rpm must be less than 6000, recieved: %d", rpm)
	}
	blower.rpm = rpm
	return nil
}

func (blower *Blower) SetTemperature(temp float64) error {
	if temp > 30 || temp < 15 {
		return fmt.Errorf("fan rpm must be less than 30 and more than 15, recieved: %f", temp)
	}
	blower.temperature = temp
	return nil
}

func (blower *Blower) UpdateLastContact() {
	blower.lastKeepAlive = time.Now()
}
