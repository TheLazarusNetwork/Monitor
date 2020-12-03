package utility

// CheckError for checking any errors
func CheckError(err error) {
	if err != nil {
		panic(err)
	}
}
