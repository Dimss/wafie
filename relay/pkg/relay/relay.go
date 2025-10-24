package relay

import wv1 "github.com/Dimss/wafie/api/gen/wafie/v1"

type Relay interface {
	Start(*wv1.RelayOptions)
	Stop(*wv1.RelayOptions)
	Status()
}
