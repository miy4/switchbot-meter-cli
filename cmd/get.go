package cmd

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-ble/ble"
	"github.com/miy4/switchbot-meter-cli/device"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	bdaddr  string
	timeout time.Duration
)

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Retrieve the temperature and humidity from the Switchbot Meter and then print the values",
	Long:  `Retrieve the temperature and humidity from the Switchbot Meter and then print the values.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dev, err := device.DefaultDevice()
		if err != nil {
			return errors.Wrap(err, "failed to create a new device")
		}
		ble.SetDefaultDevice(dev)

		innerCtx, cancel := context.WithTimeout(context.Background(), timeout)
		ctx := ble.WithSigHandler(innerCtx, cancel)

		var once sync.Once
		advHandler := func(adv ble.Advertisement) {
			defer cancel()
			once.Do(func() {
				serviceData := adv.ServiceData()[0].Data

				temperature := float64(serviceData[4]&0b01111111) +
					float64(serviceData[3]&0b00001111)*0.1
				tempFlag := (serviceData[4] & 0b10000000) >> 7
				if tempFlag == 0 {
					temperature *= -1
				}

				humidity := serviceData[5] & 0b01111111
				battery := serviceData[2] & 0b01111111

				fmt.Printf("temperature: %.1f℃\n", temperature)
				fmt.Printf("humidity: %d%%\n", humidity)
				fmt.Printf("remaining battery: %d%%\n", battery)
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
}
