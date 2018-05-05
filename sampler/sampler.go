package sampler

import (
	"github.com/CraigKelly/grample/model"
)

// A Sampler samplers from the given model
type Sampler interface {
	Init(*model.Model) error
	Sample() ([]*model.Variable, error)
}
