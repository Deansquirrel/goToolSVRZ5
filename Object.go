package goToolSVRZ5

import "time"

type ZlVersion struct {
	ObjectName    string
	ObjectType    string
	ObjectVersion string
	ObjectDate    time.Time
}

type ZlCompany struct {
	CoId        int
	CoAb        string
	CoCode      string
	CoType      int
	CoUserAb    string
	CoUserCode  string
	CoAccCrDate time.Time
}
