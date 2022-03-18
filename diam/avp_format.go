package diam

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Address struct {
	family uint16
	addr   []byte
}

func NewAddress(family uint16, address []byte) Address {
	return Address{
		family: family,
		addr:   address,
	}
}

//TODO error check
func IPv4ToByte(ipv4 string) []byte {
	parts := strings.Split(ipv4, ".")
	if len(parts) != 4 {
		panic(fmt.Sprintf("Bad ip address to convert:%s", ipv4))
	}
	ret := make([]byte, 4)
	for i := 0; i < 4; i++ {
		intVar, _ := strconv.Atoi(parts[i])
		ret[i] = byte(intVar)
	}
	return ret
}

func AVP_Integer32(code uint32, value int32, mandatory bool, vendor_id uint32) AVP {
	return Basic_AVP(code, Avp_Integer32, value, mandatory, vendor_id)
}

func AVP_Unsigned32(code uint32, value uint32, mandatory bool, vendor_id uint32) AVP {
	return Basic_AVP(code, Avp_Unsigned32, value, mandatory, vendor_id)
}

func AVP_Integer64(code uint32, value int64, mandatory bool, vendor_id uint32) AVP {
	return Basic_AVP(code, Avp_Integer64, value, mandatory, vendor_id)
}

func AVP_Unsigned64(code uint32, value uint64, mandatory bool, vendor_id uint32) AVP {
	return Basic_AVP(code, Avp_Unsigned64, value, mandatory, vendor_id)
}

func AVP_Float32(code uint32, value float32, mandatory bool, vendor_id uint32) AVP {
	return Basic_AVP(code, Avp_Float32, value, mandatory, vendor_id)
}

func AVP_Float64(code uint32, value float64, mandatory bool, vendor_id uint32) AVP {
	return Basic_AVP(code, Avp_Float64, value, mandatory, vendor_id)
}

func AVP_OctetString(code uint32, value []byte, mandatory bool, vendor_id uint32) AVP {
	return Basic_AVP(code, Avp_OctetString, value, mandatory, vendor_id)
}

func AVP_UTF8String(code uint32, value string, mandatory bool, vendor_id uint32) AVP {
	return Basic_AVP(code, Avp_UTF8String, value, mandatory, vendor_id)
}

func AVP_Enumerated(code uint32, value int32, mandatory bool, vendor_id uint32) AVP {
	return Basic_AVP(code, Avp_Enumerated, value, mandatory, vendor_id)
}

func AVP_Time(code uint32, value time.Time, mandatory bool, vendor_id uint32) AVP {
	return Basic_AVP(code, Avp_Time, value, mandatory, vendor_id)
}

func AVP_Address(code uint32, value Address, mandatory bool, vendor_id uint32) AVP {
	return Basic_AVP(code, Avp_Address, value, mandatory, vendor_id)
}

func AVP_Group(code uint32, value []AVP, mandatory bool, vendor_id uint32) AVP {
	return Basic_AVP(code, Avp_Grouped, value, mandatory, vendor_id)
}
