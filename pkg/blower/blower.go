package blower

import (
	"fmt"
	"strings"
	"time"
)

type Blower struct {
	ID               string
	FirmwareVersion  float64
	FirmwareRevision float64
	IsFanRunning     bool
	mode             int
	Fs               float64
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

func New(id string, fanPower int, temperature float64, rpm int) (*Blower, error) {
	blower := &Blower{
		ID:          id,
		temperature: temperature,
		rpm:         rpm,
	}
	err := blower.SetFanPower(fanPower)
	if err != nil {
		return nil, err
	}
	return blower, nil
}

func (blower *Blower) GenerateStausPayload() string {
	fanRunningFlag := 0
	if blower.IsFanRunning {
		fanRunningFlag = 1
	}
	payload := []string{
		fmt.Sprintf("%d", fanRunningFlag),
		fmt.Sprintf("%d", blower.fanPower),
		fmt.Sprintf("%d", blower.rpm),
		fmt.Sprintf("%.1f", blower.temperature),
		fmt.Sprintf("%d", blower.mode),
	}
	return strings.Join(payload, " ")
}

func (blower *Blower) SetMode(mode string) error {
	if modeInt, err := ModeAsInt(mode); err == nil {
		blower.mode = modeInt
		return nil
	} else {
		return err
	}
}

func (blower *Blower) SetFanPower(power int) error {
	if power < 0 || power > 12 {
		return fmt.Errorf("fan power must be postive and less than 12, recieved: %d", power)
	}
	blower.fanPower = power
	return nil
}

func (blower *Blower) SetFanRPM(rpm int) error {
	if rpm > 6000 {
		return fmt.Errorf("fan rpm must be less than 6000, recieved: %d", rpm)
	}
	blower.rpm = rpm
	return nil
}

func (blower *Blower) Touch() {
	blower.lastKeepAlive = time.Now()
}
