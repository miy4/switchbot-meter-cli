package cmd

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/go-ble/ble"
	"github.com/miy4/switchbot-meter-cli/device"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var duration time.Duration

// scanCmd represents the scan command
var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Start scan your BLE devices for available Switchbot services",
	Long: `Start scan your BLE devices for available Switchbot services.

You can press ctrl-c to cancel the scan process.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dev, err := device.DefaultDevice()
		if err != nil {
			return errors.Wrap(err, "failed to create a new device")
		}
		ble.SetDefaultDevice(dev)

		ctx := ble.WithSigHandler(context.WithTimeout(context.Background(), duration))
		err = ble.Scan(ctx, true, advHandler, nil)
		if err != nil &&
			errors.Cause(err) != context.Canceled &&
			errors.Cause(err) != context.DeadlineExceeded {
			return errors.Wrap(err, "failed to scan ble devices")
		}

		return nil
	},
}

func advHandler(a ble.Advertisement) {
	buf := bytes.NewBufferString(fmt.Sprintf("[%s] RSSI: %3d,", a.Addr(), a.RSSI()))
	comma := ""
	if len(a.LocalName()) > 0 {
		buf.WriteString(fmt.Sprintf(" Name: %s", a.LocalName()))
		comma = ","
	}
	if len(a.Services()) > 0 {
		buf.WriteString(fmt.Sprintf("%s Svcs: %v", comma, a.Services()))
		comma = ","
	}
	if len(a.ManufacturerData()) > 0 {
		buf.WriteString(fmt.Sprintf("%s MD: %X", comma, a.ManufacturerData()))
	}

	fmt.Println(buf.String())
}

func init() {
	rootCmd.AddCommand(scanCmd)
	scanCmd.Flags().DurationVarP(&duration, "duration", "d", 5*time.Second, "scanning duration")
}
