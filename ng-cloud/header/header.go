package header

const (
	Base           = iota + 300
	JsonParseError = Base + 1
	JsonParamNil   = Base + 2
	OpenFileError  = Base + 3
	DBError        = Base + 4
)
