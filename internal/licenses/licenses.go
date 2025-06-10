package licenses

import (
	_ "embed"
)

//go:embed LICENSE
var ProjectLicense string

//go:embed THIRD_PARTY_LICENSES
var ThirdPartyLicenses string

func GetProjectLicense() (string, error) {
	if ProjectLicense == "" {
		return "", &LicenseError{FileName: "LICENSE"}
	}
	return ProjectLicense, nil
}

func GetThirdPartyLicenses() (string, error) {
	if ThirdPartyLicenses == "" {
		return "", &LicenseError{FileName: "THIRD_PARTY_LICENSES"}
	}
	return ThirdPartyLicenses, nil
}

type LicenseError struct {
	FileName string
}

func (e *LicenseError) Error() string {
	return e.FileName + " is missing"
}