package ec2

import (
	"bufio"
	"fmt"
	"net/http"
	"strings"
)

// ImageConstraint specifies a range of possible machine images.
// TODO allow specification of softer constraints?
type ImageConstraint struct {
	UbuntuRelease     string
	Architecture      string
	PersistentStorage bool
	Region            string
	Daily             bool
	Desktop           bool
}

var DefaultImageConstraint = &ImageConstraint{
	UbuntuRelease:     "oneiric",
	Architecture:      "386",
	PersistentStorage: true,
	Region:            "us-east-1",
	Daily:             false,
	Desktop:           false,
}

type ImageSpec struct {
	ImageId string
	Architecture string	// The architecture the image will run on.
	OS string			// The OS the image will run on.
}

// imagesHost holds the address of the images http server.
// It is a variable so that tests can change it to refer to a local
// server when needed.
var imagesHost = "http://uec-images.ubuntu.com"

func FindImageSpec(spec *ImageConstraint) (*ImageSpec, error) {
	// note: original get_image_id added three optional args:
	// DefaultImageId		if found, returns that immediately
	// Region				overrides spec.Region
	// DefaultSeries		used if spec.UbuntuRelease is ""

	hclient := new(http.Client)
	uri := fmt.Sprintf(imagesHost+"/query/%s/%s/%s.current.txt",
		spec.UbuntuRelease,
		either(spec.Desktop, "desktop", "server"), // variant.
		either(spec.Daily, "daily", "released"),   // version.
	)
	resp, err := hclient.Get(uri)
	if err == nil && resp.StatusCode != 200 {
		err = fmt.Errorf("%s", resp.Status)
	}
	if err != nil {
		return nil, fmt.Errorf("error getting instance types: %v", err)
	}
	defer resp.Body.Close()
	ebsMatch := either(spec.PersistentStorage, "ebs", "instance-store")

	// Use Go format for specifying the architecture.
	arch := spec.Architecture
	if arch == "i386" {
		arch = "386"
	}
	r := bufio.NewReader(resp.Body)
	for {
		line, _, err := r.ReadLine()
		if err != nil {
			return nil, fmt.Errorf("cannot find matching image: %v", err)
		}
		f := strings.Split(string(line), "\t")
		if len(f) < 8 {
			continue
		}
		if f[4] != ebsMatch {
			continue
		}
		if f[5] == arch && f[6] == spec.Region {
			return &ImageSpec{
				ImageID: f[7],
				Architecture: spec.Architecture,
				OS: "linux",
			}, nil
		}
	}
	panic("not reached")
}

func either(yes bool, a, b string) string {
	if yes {
		return a
	}
	return b
}
