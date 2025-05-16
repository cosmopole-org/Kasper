package kasper

type Config struct {
	Id  string
	Log func(...interface{})
}
