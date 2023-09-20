package store

import (
	"encoding/json"
	"net"

	"github.com/bmc-toolbox/common"
	sconsts "github.com/metal-toolbox/rivets/serverservice"
	"github.com/pkg/errors"
	sservice "go.hollow.sh/serverservice/pkg/api/v1"
)

func (s *Serverservice) bmcAddressFromAttributes(attributes []sservice.Attributes) (net.IP, error) {
	ip := net.IP{}

	bmcAttribute := findAttribute(sconsts.ServerAttributeNSBmcAddress, attributes)
	if bmcAttribute == nil {
		return ip, errors.Wrap(ErrBMCAddress, "not found: "+sconsts.ServerAttributeNSBmcAddress)
	}

	data := map[string]string{}
	if err := json.Unmarshal(bmcAttribute.Data, &data); err != nil {
		return ip, errors.Wrap(ErrBMCAddress, err.Error())
	}

	if data["address"] == "" {
		return ip, errors.Wrap(ErrBMCAddress, "value undefined: "+sconsts.ServerAttributeNSBmcAddress)
	}

	return net.ParseIP(data["address"]), nil
}

func (s *Serverservice) vendorModelFromAttributes(attributes []sservice.Attributes) (deviceVendor, deviceModel, deviceSerial string, err error) {
	vendorAttrs := map[string]string{}

	vendorAttribute := findAttribute(sconsts.ServerAttributeNSVendor, attributes)
	if vendorAttribute == nil {
		return deviceVendor,
			deviceModel,
			deviceSerial,
			ErrVendorModelAttributes
	}

	if err := json.Unmarshal(vendorAttribute.Data, &vendorAttrs); err != nil {
		return deviceVendor,
			deviceModel,
			deviceSerial,
			errors.Wrap(ErrVendorModelAttributes, "server vendor attribute: "+err.Error())
	}

	deviceVendor = common.FormatVendorName(vendorAttrs["vendor"])
	deviceModel = common.FormatProductName(vendorAttrs["model"])
	deviceSerial = vendorAttrs["serial"]

	if deviceVendor == "" {
		return deviceVendor,
			deviceModel,
			deviceSerial,
			errors.Wrap(ErrVendorModelAttributes, "device vendor unknown")
	}

	if deviceModel == "" {
		return deviceVendor,
			deviceModel,
			deviceSerial,
			errors.Wrap(ErrVendorModelAttributes, "device model unknown")
	}

	return
}

func findAttribute(ns string, attributes []sservice.Attributes) *sservice.Attributes {
	for _, attribute := range attributes {
		if attribute.Namespace == ns {
			return &attribute
		}
	}

	return nil
}
