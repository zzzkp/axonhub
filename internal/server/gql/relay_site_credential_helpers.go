package gql

func stringPtrOrNil(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func intPtrOrNil(value int) *int {
	if value <= 0 {
		return nil
	}
	return &value
}
