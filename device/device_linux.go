package device

import (
	"github.com/go-ble/ble"
	"github.com/go-ble/ble/linux"
)

func DefaultDevice(opts ...ble.Option) (device ble.Device, err error) {
	return linux.NewDevice(opts...)
}
