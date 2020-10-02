package cmd

import (
	"context"
	errs "errors"
	"fmt"
	"sync"
	"time"

	"github.com/miy4/switchbot-meter-cli/device"

	"github.com/go-ble/ble"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	bdaddr  string
	timeout time.Duration
	output  string
)

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Retrieve the temperature and humidity from the Switchbot Meter and then print the values",
	Long:  `Retrieve the temperature and humidity from the Switchbot Meter and then print the values.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		switch output {
		case "none", "tsv", "json":
			return nil
		default:
			return errs.New(fmt.Sprintf("invalid argument \"%s\" for \"-o, --output\" flag: must be \"none\", \"tsv\" or \"json\"", output))
		}
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		dev, err := device.DefaultDevice()
		if err != nil {
			return errors.Wrap(err, "failed to create a new device")
		}
		ble.SetDefaultDevice(dev)

		innerCtx, cancel := context.WithTimeout(context.Background(), timeout)
		ctx := ble.WithSigHandler(innerCtx, cancel)

		var printable Printable
		if output == "tsv" {
			printable = TabSeparated{}
		} else if output == "json" {
			printable = Json{}
		} else {
			printable = HumanReadable{}
		}

		var once sync.Once
		advHandler := func(adv ble.Advertisement) {
			defer cancel()
			once.Do(func() {
				serviceData := adv.ServiceData()[0].Data

				tempInteger := serviceData[4] & 0b01111111
				tempDecimal := serviceData[3] & 0b00001111
				tempFlag := (serviceData[4] & 0b10000000) >> 7
				humidity := serviceData[5] & 0b01111111
				battery := serviceData[2] & 0b01111111

				measurement := Measurement{
					tempInteger: tempInteger,
					tempDecimal: tempDecimal,
					tempFlag:    tempFlag,
					humidity:    humidity,
					battery:     battery,
				}

				printable.print(measurement)
			})
		}
		err = ble.Scan(ctx, true, advHandler, advFilter)
		switch errors.Cause(err) {
		case nil, context.Canceled:
			// nop
		case context.DeadlineExceeded:
			return errors.Wrap(err, "timed out")
		default:
			return err
		}

		return nil
	},
}

type Measurement struct {
	tempInteger byte
	tempDecimal byte
	tempFlag    byte
	humidity    byte
	battery     byte
}

type Printable interface {
	print(m Measurement)
}

type HumanReadable struct{}

func (hr HumanReadable) print(m Measurement) {
	fmt.Printf("temperature: %.1f℃\n", calculateTemperature(m))
	fmt.Printf("humidity: %d%%\n", m.humidity)
	fmt.Printf("remaining battery: %d%%\n", m.battery)
}

type TabSeparated struct{}

func (tsv TabSeparated) print(m Measurement) {
	temperature := calculateTemperature(m)
	fmt.Printf("%.1f\t%d\t%d\n", temperature, m.humidity, m.battery)
}

type Json struct{}

func (json Json) print(m Measurement) {
	temperature := calculateTemperature(m)
	fmt.Printf("{ \"temperature\": %.1f, \"humidity\": %d, \"battery\": %d }\n", temperature, m.humidity, m.battery)
}

func calculateTemperature(m Measurement) float64 {
	temperature := float64(m.tempInteger) + float64(m.tempDecimal)*0.1
	if m.tempFlag == 0 {
		temperature *= -1
	}
	return temperature
}

func advFilter(adv ble.Advertisement) bool {
	if adv == nil || adv.Addr() == nil || adv.ServiceData() == nil {
		return false
	}

	return adv.Addr().String() == bdaddr && len(adv.ServiceData()) > 0
}

func init() {
	rootCmd.AddCommand(getCmd)
	getCmd.Flags().StringVarP(&bdaddr, "address", "a", "", "bluetooth device address (required)")
	getCmd.MarkFlagRequired("address")
	getCmd.Flags().DurationVarP(&timeout, "timeout", "t", 5*time.Second, "scanning timeout")
	getCmd.Flags().StringVarP(&output, "output", "o", "none", "output formatted as: \"none\", \"tsv\" or \"json\"")
}
