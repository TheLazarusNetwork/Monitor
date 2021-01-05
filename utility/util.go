package utility

import log "github.com/sirupsen/logrus"

// Version Build Version
var Version = "0.1"

// CheckError for checking any errors
func CheckError(message string, err error) {
	if err != nil {
		log.Fatalf("%s %+v", message, err)
	}
}
