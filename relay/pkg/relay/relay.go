package relay

type Relay interface {
	Start()
	Stop()
	Status()
}
