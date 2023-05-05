package plugin_manage

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var PLUGIN_API_VERSION = ParseVersion("v1.0.0-test")

type Version struct {
	Major int
	Minor int
	Patch int
	Desc  string
}

func (v Version) String() string {
	return fmt.Sprintf("v%d.%d.%d-%s", v.Major, v.Minor, v.Patch, v.Desc)
}

var vreg = regexp.MustCompile(`v(\d+)\.?(\d+)\.?(\d*)[-_]?(\w*)`)

func ParseVersion(version string) Version {
	v := vreg.FindStringSubmatch(version)
	if len(v) == 0 {
		return Version{}
	}
	major, _ := strconv.Atoi(v[1])
	minor, _ := strconv.Atoi(v[2])
	patch, _ := strconv.Atoi(v[3])

	return Version{
		Major: major,
		Minor: minor,
		Patch: patch,
		Desc:  v[4],
	}
}

const (
	VersionEqual uint8 = iota
	VersionSmall
	VersionBig
	VersionIncompatible
)

func CompareVersionStr(src, dest string) uint8 {
	return CompareVersion(ParseVersion(src), ParseVersion(dest))
}

func CompareVersion(src, dest Version) uint8 {
	if src.Major > dest.Major {
		return VersionBig
	} else if src.Major < dest.Major {
		return VersionSmall
	}

	if src.Minor > dest.Minor {
		return VersionBig
	} else if src.Minor < dest.Minor {
		return VersionSmall
	}

	if src.Patch > dest.Patch {
		return VersionBig
	} else if src.Patch < dest.Patch {
		return VersionSmall
	}
	return VersionEqual
}

// 跳过 Patch 版本号
func CompareVersion2(src, dest Version) uint8 {
	if src.Major > dest.Major {
		return VersionBig
	} else if src.Major < dest.Major {
		return VersionSmall
	}

	if src.Minor > dest.Minor {
		return VersionBig
	} else if src.Minor < dest.Minor {
		return VersionSmall
	}
	return VersionEqual
}

func ComparePluginApiVersion(pluginV Version) uint8 {
	if pluginV.Major != PLUGIN_API_VERSION.Major {
		return VersionIncompatible
	}
	return CompareVersion2(pluginV, PLUGIN_API_VERSION)
}

func IsSupportPlugin(apiVersion string) bool {
	apiVersions := strings.Split(apiVersion, ",")
	for _, apiVersion := range apiVersions {
		if ComparePluginApiVersion(ParseVersion(apiVersion)) < VersionBig {
			return true
		}
	}
	return false
}
